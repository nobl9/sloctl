package internal

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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

func TestReplayConfigSetsRecalculationOnlyForComposites(t *testing.T) {
	from := time.Date(2026, time.August, 24, 8, 45, 11, 0, time.UTC)
	timeNow := time.Date(2026, time.August, 24, 9, 0, 11, 0, time.UTC)
	for _, test := range []struct {
		name         string
		isComposite  bool
		expectedType replayV1.ReplayType
	}{
		{
			name:         "composite SLO is replayed in recalculation mode",
			isComposite:  true,
			expectedType: replayV1.ReplayTypeRecalculation,
		},
		{
			name:         "regular SLO leaves the type unset so the server applies its default",
			isComposite:  false,
			expectedType: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := ReplayConfig{
				Project:     "project",
				SLO:         "slo",
				From:        from,
				isComposite: test.isComposite,
			}
			assert.Equal(t, test.expectedType, config.ToReplay(timeNow).ReplayType)
		})
	}
}

func TestVerifySLOsNoLongerRejectsComposites(t *testing.T) {
	replays := []ReplayConfig{{Project: "project", SLO: "composite-slo"}}
	slos := []replaySLO{{
		name:                   "composite-slo",
		project:                "project",
		hasCompositeObjectives: true,
	}}

	filtered, missing := matchReplaysToSLOs(replays, slos)

	require.Empty(t, missing)
	require.Len(t, filtered, 1)
	assert.True(t, filtered[0].isComposite)
}

func TestVerifySLOsReportsUnmatchedSLOs(t *testing.T) {
	replays := []ReplayConfig{{Project: "project", SLO: "missing-slo"}}

	filtered, missing := matchReplaysToSLOs(replays, nil)

	assert.Empty(t, filtered)
	assert.Equal(t, []string{"'missing-slo' SLO in 'project' Project"}, missing)
}

func TestReplayConfigAlwaysNamesATypeForTheAvailabilityCheck(t *testing.T) {
	composite := ReplayConfig{isComposite: true}
	regular := ReplayConfig{}

	assert.Equal(t, replayV1.ReplayTypeRecalculation, composite.availabilityReplayType())
	assert.Equal(t, replayV1.ReplayTypeReimportAndRecalculation, regular.availabilityReplayType())
}

// Guards the wiring, not just the pieces: the replay type is decided from the
// SLO fetched during verification, so a config that loses that detail on its way
// to the run request sends the mode the server rejects for composites.
func TestRunReplaysSendsRecalculationForCompositeSLO(t *testing.T) {
	apiURL, err := url.Parse("https://example.com/api")
	require.NoError(t, err)
	client, err := sdk.NewClient(&sdk.Config{
		DisableOkta:  true,
		Organization: "test-organization",
		Project:      "target-project",
		URL:          apiURL,
	})
	require.NoError(t, err)

	var runRequest replayV1.RunRequest
	var availabilityType string
	client.HTTP = &http.Client{
		Transport: replayRoundTripper(func(request *http.Request) (*http.Response, error) {
			recorder := httptest.NewRecorder()
			recorder.Header().Set("Content-Type", "application/json")
			switch {
			case strings.Contains(request.URL.Path, "timemachine/availability"):
				availabilityType = request.URL.Query().Get("type")
				require.NoError(t, json.NewEncoder(recorder).Encode(
					replayV1.ReplayAvailability{Available: true}))
			case request.Method == http.MethodPost && strings.HasSuffix(request.URL.Path, "/timetravel"):
				require.NoError(t, json.NewDecoder(request.Body).Decode(&runRequest))
				recorder.WriteHeader(http.StatusCreated)
			default:
				require.NoError(t, json.NewEncoder(recorder).Encode([]v1alpha.GenericObject{{
					"apiVersion": manifest.VersionV1alpha,
					"kind":       manifest.KindSLO,
					"metadata": map[string]any{
						"name":    "composite-slo",
						"project": "target-project",
					},
					"spec": map[string]any{
						"objectives": []any{map[string]any{
							"name":      "composite",
							"target":    0.95,
							"composite": map[string]any{},
						}},
					},
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
		SLO:     "composite-slo",
		From:    time.Now().Add(-10 * time.Minute),
	}})

	require.NoError(t, err)
	assert.Zero(t, failed)
	assert.Equal(t, replayV1.ReplayTypeRecalculation, runRequest.ReplayType)
	assert.Equal(t, string(replayV1.ReplayTypeRecalculation), availabilityType)
}

// Without queues each next replay in a bulk run asks about a longer window, to
// account for the earlier ones still occupying the data source. Composites have
// no data source, so they must all ask about the window the user requested.
func TestVerifySLOsDoesNotPadTheWindowForComposites(t *testing.T) {
	apiURL, err := url.Parse("https://example.com/api")
	require.NoError(t, err)
	client, err := sdk.NewClient(&sdk.Config{
		DisableOkta:  true,
		Organization: "test-organization",
		Project:      "target-project",
		URL:          apiURL,
	})
	require.NoError(t, err)

	var mu sync.Mutex
	durations := make(map[string]string)
	client.HTTP = &http.Client{
		Transport: replayRoundTripper(func(request *http.Request) (*http.Response, error) {
			recorder := httptest.NewRecorder()
			recorder.Header().Set("Content-Type", "application/json")
			if strings.Contains(request.URL.Path, "timemachine/availability") {
				mu.Lock()
				durations[request.URL.Query().Get("sloName")] = request.URL.Query().Get("durationValue")
				mu.Unlock()
				require.NoError(t, json.NewEncoder(recorder).Encode(
					replayV1.ReplayAvailability{Available: true}))
				return recorder.Result(), nil
			}
			objects := make([]v1alpha.GenericObject, 0, 2)
			for _, name := range []string{"composite-one", "composite-two"} {
				objects = append(objects, v1alpha.GenericObject{
					"apiVersion": manifest.VersionV1alpha,
					"kind":       manifest.KindSLO,
					"metadata":   map[string]any{"name": name, "project": "target-project"},
					"spec": map[string]any{
						"objectives": []any{map[string]any{
							"name":      "composite",
							"target":    0.95,
							"composite": map[string]any{},
						}},
					},
				})
			}
			require.NoError(t, json.NewEncoder(recorder).Encode(objects))
			return recorder.Result(), nil
		}),
	}

	replay := ReplayCmd{client: client, playlistsAvailable: false}
	from := time.Now().Add(-10 * time.Minute)

	_, err = replay.verifySLOs(t.Context(), []ReplayConfig{
		{Project: "target-project", SLO: "composite-one", From: from},
		{Project: "target-project", SLO: "composite-two", From: from},
	})

	require.NoError(t, err)
	assert.Equal(t, durations["composite-one"], durations["composite-two"],
		"the second composite must not be asked about a padded window")
}

// A composite queued ahead of a regular SLO must not push that SLO's window out:
// the padding exists for replays that occupy the data source, and a composite
// does not.
func TestVerifySLOsPadsOnlyForPrecedingDataSourceReplays(t *testing.T) {
	apiURL, err := url.Parse("https://example.com/api")
	require.NoError(t, err)
	client, err := sdk.NewClient(&sdk.Config{
		DisableOkta:  true,
		Organization: "test-organization",
		Project:      "target-project",
		URL:          apiURL,
	})
	require.NoError(t, err)

	composites := map[string]bool{"composite-slo": true, "regular-first": false, "regular-last": false}
	var mu sync.Mutex
	durations := make(map[string]string)
	client.HTTP = &http.Client{
		Transport: replayRoundTripper(func(request *http.Request) (*http.Response, error) {
			recorder := httptest.NewRecorder()
			recorder.Header().Set("Content-Type", "application/json")
			if strings.Contains(request.URL.Path, "timemachine/availability") {
				mu.Lock()
				durations[request.URL.Query().Get("sloName")] = request.URL.Query().Get("durationValue")
				mu.Unlock()
				require.NoError(t, json.NewEncoder(recorder).Encode(
					replayV1.ReplayAvailability{Available: true}))
				return recorder.Result(), nil
			}
			objects := make([]v1alpha.GenericObject, 0, len(composites))
			for name, composite := range composites {
				spec := map[string]any{"indicator": map[string]any{
					"metricSource": map[string]any{"name": "ds", "project": "target-project"},
				}}
				if composite {
					spec = map[string]any{"objectives": []any{map[string]any{
						"name": "composite", "target": 0.95, "composite": map[string]any{},
					}}}
				}
				objects = append(objects, v1alpha.GenericObject{
					"apiVersion": manifest.VersionV1alpha,
					"kind":       manifest.KindSLO,
					"metadata":   map[string]any{"name": name, "project": "target-project"},
					"spec":       spec,
				})
			}
			require.NoError(t, json.NewEncoder(recorder).Encode(objects))
			return recorder.Result(), nil
		}),
	}

	replay := ReplayCmd{client: client, playlistsAvailable: false}
	from := time.Now().Add(-10 * time.Minute)

	_, err = replay.verifySLOs(t.Context(), []ReplayConfig{
		{Project: "target-project", SLO: "regular-first", From: from},
		{Project: "target-project", SLO: "composite-slo", From: from},
		{Project: "target-project", SLO: "regular-last", From: from},
	})
	require.NoError(t, err)

	first, err := strconv.Atoi(durations["regular-first"])
	require.NoError(t, err)
	last, err := strconv.Atoi(durations["regular-last"])
	require.NoError(t, err)
	assert.Equal(t, int(averageReplayDuration.Minutes()), last-first,
		"the composite between them must not add a slot to the offset")
}
