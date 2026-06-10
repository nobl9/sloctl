package internal

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
