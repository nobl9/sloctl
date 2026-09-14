package internal

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/manifest/v1alpha"
	"github.com/nobl9/nobl9-go/sdk"
	replayV1 "github.com/nobl9/nobl9-go/sdk/endpoints/replay/v1"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplayConfigDecodesSourceSLOIntoRunRequest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "replay-20260824T084511Z.yaml")
	replayConfig := `- slo: target-slo
  project: target-project
  from: 2026-08-24T08:45:11Z
  sourceSLO:
    slo: source-slo
    project: source-project
    objectivesMap:
      - source: source-availability
        target: target-availability
      - source: source-latency
        target: target-latency
`
	require.NoError(t, os.WriteFile(path, []byte(replayConfig), 0o600))

	replay := ReplayCmd{}
	configs, err := replay.readConfigFile(path)
	require.NoError(t, err)
	require.Len(t, configs, 1)

	request := configs[0].ToReplay(time.Date(2026, time.August, 24, 9, 0, 11, 0, time.UTC))
	assert.Equal(t, replayV1.RunRequest{
		Project: "target-project",
		SLO:     "target-slo",
		Duration: replayV1.Duration{
			Unit:  replayV1.DurationUnitMinute,
			Value: 20,
		},
		SourceSLO: &replayV1.SourceSLO{
			SLO:     "source-slo",
			Project: "source-project",
			ObjectivesMap: []replayV1.SourceSLOItem{
				{Source: "source-availability", Target: "target-availability"},
				{Source: "source-latency", Target: "target-latency"},
			},
		},
	}, request)
}

func TestRunReplayPreservesConflictError(t *testing.T) {
	apiURL, err := url.Parse("https://example.com/api")
	require.NoError(t, err)
	client, err := sdk.NewClient(&sdk.Config{
		DisableOkta:  true,
		Organization: "test-organization",
		Project:      "target-project",
		URL:          apiURL,
	})
	require.NoError(t, err)
	client.HTTP = &http.Client{
		Transport: replayRoundTripper(func(*http.Request) (*http.Response, error) {
			recorder := httptest.NewRecorder()
			recorder.Header().Set("Content-Type", "application/json")
			recorder.WriteHeader(http.StatusConflict)
			require.NoError(t, json.NewEncoder(recorder).Encode(sdk.APIErrors{
				Errors: []sdk.APIError{{Title: "Replay already exists"}},
			}))
			return recorder.Result(), nil
		}),
	}
	replay := ReplayCmd{client: client}

	err = replay.runReplay(t.Context(), ReplayConfig{
		Project: "target-project",
		SLO:     "target-slo",
		From:    time.Now().Add(-10 * time.Minute),
	})

	require.EqualError(
		t,
		err,
		"Replay for SLO: 'target-slo' in project: 'target-project' already exist",
	)
}

func TestDecodeReplaySLOAllowsCompositeWithoutIndicator(t *testing.T) {
	object := v1alpha.GenericObject{
		"apiVersion": manifest.VersionV1alpha,
		"kind":       manifest.KindSLO,
		"metadata": map[string]any{
			"name":    "composite-slo",
			"project": "target-project",
		},
		"spec": map[string]any{
			"objectives": []any{
				map[string]any{
					"name":        "composite",
					"displayName": "Composite",
					"target":      0.95,
					"composite":   map[string]any{},
				},
			},
		},
	}

	slo, err := decodeReplaySLO(object)

	require.NoError(t, err)
	assert.Equal(t, "composite-slo", slo.name)
	assert.Equal(t, "target-project", slo.project)
	assert.True(t, slo.hasCompositeObjectives)
	assert.Zero(t, slo.metricSource)
}

type replayRoundTripper func(*http.Request) (*http.Response, error)

func (f replayRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestMatchReplaysToSLOsReportsUnmatchedSLOs(t *testing.T) {
	replays := []ReplayConfig{{Project: "project", SLO: "missing-slo"}}

	matched, missing := matchReplaysToSLOs(replays, nil)

	assert.Empty(t, matched)
	assert.Equal(t, []string{"'missing-slo' SLO in 'project' Project"}, missing)
}

func TestRunReplaysSendsRecalculationOnlyForCompositeSLOs(t *testing.T) {
	compositeSpec := map[string]any{"objectives": []any{map[string]any{
		"name": "composite", "target": 0.95, "composite": map[string]any{},
	}}}
	regularSpec := map[string]any{"indicator": map[string]any{
		"metricSource": map[string]any{"name": "ds", "project": "target-project"},
	}}
	for name, test := range map[string]struct {
		spec         map[string]any
		expectedType string
	}{
		"composite SLO": {spec: compositeSpec, expectedType: string(replayV1.ReplayTypeRecalculation)},
		"regular SLO":   {spec: regularSpec, expectedType: ""},
	} {
		t.Run(name, func(t *testing.T) {
			apiURL, err := url.Parse("https://example.com/api")
			require.NoError(t, err)
			client, err := sdk.NewClient(&sdk.Config{
				DisableOkta:  true,
				Organization: "test-organization",
				Project:      "target-project",
				URL:          apiURL,
			})
			require.NoError(t, err)

			var runRequest map[string]any
			var availabilityQuery url.Values
			client.HTTP = &http.Client{
				Transport: replayRoundTripper(func(request *http.Request) (*http.Response, error) {
					recorder := httptest.NewRecorder()
					recorder.Header().Set("Content-Type", "application/json")
					switch {
					case strings.Contains(request.URL.Path, "timemachine/availability"):
						availabilityQuery = request.URL.Query()
						assert.NoError(t, json.NewEncoder(recorder).Encode(
							replayV1.ReplayAvailability{Available: true},
						))
					case request.Method == http.MethodPost && strings.HasSuffix(request.URL.Path, "/timetravel"):
						assert.NoError(t, json.NewDecoder(request.Body).Decode(&runRequest))
						recorder.WriteHeader(http.StatusCreated)
					default:
						assert.NoError(t, json.NewEncoder(recorder).Encode([]v1alpha.GenericObject{{
							"apiVersion": manifest.VersionV1alpha,
							"kind":       manifest.KindSLO,
							"metadata":   map[string]any{"name": "target-slo", "project": "target-project"},
							"spec":       test.spec,
						}}))
					}
					return recorder.Result(), nil
				}),
			}

			replay := ReplayCmd{client: client, playlistsAvailable: true}
			cmd := &cobra.Command{}
			cmd.SetOut(io.Discard)
			cmd.SetContext(t.Context())

			failed, err := replay.RunReplays(cmd, []ReplayConfig{{
				Project: "target-project",
				SLO:     "target-slo",
				From:    time.Now().Add(-10 * time.Minute),
			}})

			require.NoError(t, err)
			assert.Zero(t, failed)
			replayType, hasReplayType := runRequest["replayType"]
			assert.Equal(t, test.expectedType != "", hasReplayType)
			if hasReplayType {
				assert.Equal(t, test.expectedType, replayType)
			}
			assert.Equal(t, test.expectedType, availabilityQuery.Get("type"))
		})
	}
}
