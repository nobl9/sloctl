package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/manifest/v1alpha"
	"github.com/nobl9/nobl9-go/sdk"
	replayV1 "github.com/nobl9/nobl9-go/sdk/endpoints/replay/v1"
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
