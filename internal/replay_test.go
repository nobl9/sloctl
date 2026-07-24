package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	v1alphaParser "github.com/nobl9/nobl9-go/manifest/v1alpha/parser"
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/stretchr/testify/require"
)

func TestReplayCmd_verifySLOs_ProjectScopedResponsesWithoutMetadataProject(t *testing.T) {
	const defaultProject = "default"
	slosByProject := map[string]string{
		"replay-project-a": "replay-slo-a",
		"replay-project-b": "replay-slo-b",
	}
	useGenericObjects := v1alphaParser.UseGenericObjects
	useJSONNumber := v1alphaParser.UseJSONNumber
	v1alphaParser.UseGenericObjects = true
	v1alphaParser.UseJSONNumber = true
	t.Cleanup(func() {
		v1alphaParser.UseGenericObjects = useGenericObjects
		v1alphaParser.UseJSONNumber = useJSONNumber
	})

	client, err := sdk.NewClient(&sdk.Config{
		DisableOkta: true,
		Project:     defaultProject,
	})
	require.NoError(t, err)
	client.HTTP = &http.Client{Transport: replayRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		project := r.Header.Get(sdk.HeaderProject)
		sloName, ok := slosByProject[project]
		if !ok {
			return nil, fmt.Errorf("unexpected project header: %q", project)
		}

		switch r.URL.Path {
		case "/get/slo":
			if name := r.URL.Query().Get("name"); name != sloName {
				return nil, fmt.Errorf("unexpected SLO name for project %q: %q", project, name)
			}
			return replayJSONResponse([]map[string]any{
				{
					"apiVersion": "n9/v1alpha",
					"kind":       "SLO",
					"metadata": map[string]any{
						"name": sloName,
					},
					"spec": map[string]any{
						"indicator": map[string]any{
							"metricSource": map[string]any{
								"name":    "replay-source",
								"project": project,
								"kind":    "Agent",
							},
						},
					},
				},
			})
		case "/internal/timemachine/availability":
			return replayJSONResponse(map[string]any{"available": true})
		default:
			return nil, fmt.Errorf("unexpected request path: %s", r.URL.Path)
		}
	})}

	replays := make([]ReplayConfig, 0, len(slosByProject))
	for project, sloName := range slosByProject {
		replays = append(replays, ReplayConfig{
			Project: project,
			SLO:     sloName,
			From:    time.Now().Add(-time.Hour),
		})
	}
	err = (&ReplayCmd{
		client:             client,
		project:            defaultProject,
		playlistsAvailable: true,
	}).verifySLOs(t.Context(), replays)
	require.NoError(t, err)
}

func TestReplayCmd_readConfigFile_WithSourceSLO(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "replay.yaml")
	err := os.WriteFile(path, []byte(`- slo: prometheus-latency
  project: default
  from: 2023-03-02T16:00:00Z
  sourceSLO:
    slo: my-service-latency
    project: my-service-test-project
    objectivesMap:
      - source: acceptable
        target: objective-1
      - source: alarming
        target: objective-2
`), 0o600)
	require.NoError(t, err)

	replays, err := (&ReplayCmd{}).readConfigFile(path)
	require.NoError(t, err)
	require.Len(t, replays, 1)
	require.NotNil(t, replays[0].SourceSLO)
	require.Equal(t, "my-service-latency", replays[0].SourceSLO.SLO)
	require.Equal(t, "my-service-test-project", replays[0].SourceSLO.Project)
	require.Len(t, replays[0].SourceSLO.ObjectivesMap, 2)
	require.Equal(t, "acceptable", replays[0].SourceSLO.ObjectivesMap[0].Source)
	require.Equal(t, "objective-1", replays[0].SourceSLO.ObjectivesMap[0].Target)

	request := replays[0].ToReplay(time.Date(2023, 3, 2, 17, 0, 0, 0, time.UTC))
	require.Equal(t, "prometheus-latency", request.SLO)
	require.Same(t, replays[0].SourceSLO, request.SourceSLO)
}

type replayRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f replayRoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func replayJSONResponse(value any) (*http.Response, error) {
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(rec).Encode(value); err != nil {
		return nil, err
	}
	return rec.Result(), nil
}
