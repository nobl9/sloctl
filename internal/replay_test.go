package internal

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/manifest/v1alpha"
	v1alphaParser "github.com/nobl9/nobl9-go/manifest/v1alpha/parser"
	"github.com/nobl9/nobl9-go/sdk"
	objectsV1 "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v1"
	replayV1 "github.com/nobl9/nobl9-go/sdk/endpoints/replay/v1"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplayRunContinuesAfterRequestFailures(t *testing.T) {
	for _, queued := range []bool{true, false} {
		for _, scenario := range []string{
			"success", "partial API failure", "all API failures", "partial refresh failure", "all refresh failures",
		} {
			t.Run(fmt.Sprintf("queued=%t/%s", queued, scenario), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					testReplayRun(t, queued, scenario)
				})
			})
		}
	}
}

func testReplayRun(t *testing.T, queued bool, scenario string) {
	t.Helper()
	apiURL, err := url.Parse("https://api.example.com")
	require.NoError(t, err)
	idpURL, err := url.Parse("https://idp.example.com")
	require.NoError(t, err)
	t.Setenv("REPLAY_TEST_CLIENT_ID", "test-client")
	t.Setenv("REPLAY_TEST_CLIENT_SECRET", "test-secret")
	config, err := sdk.ReadConfig(sdk.ConfigOptionNoConfigFile(), sdk.ConfigOptionEnvPrefix("REPLAY_TEST_"))
	require.NoError(t, err)
	config.URL, config.OktaOrgURL, config.OktaAuthServer = apiURL, idpURL, "test"
	config.Timeout = 0

	fixture := replayRunFixture{queued: queued, scenario: scenario}
	transport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = transport })
	http.DefaultTransport = replayRoundTripper(func(request *http.Request) (*http.Response, error) {
		return fixture.roundTrip(t, request)
	})
	client, err := sdk.NewClient(config)
	require.NoError(t, err)
	useGenericObjects := v1alphaParser.UseGenericObjects
	v1alphaParser.UseGenericObjects = true
	t.Cleanup(func() { v1alphaParser.UseGenericObjects = useGenericObjects })
	path := filepath.Join(t.TempDir(), "replays.json")
	replays := []ReplayConfig{
		{Project: "project", SLO: "slo-a", From: time.Now().Add(-time.Hour)},
		{Project: "project", SLO: "slo-b", From: time.Now().Add(-time.Hour)},
		{Project: "project", SLO: "slo-c", From: time.Now().Add(-time.Hour)},
	}
	data, err := json.Marshal(replays)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))
	replay := ReplayCmd{client: client, configPaths: []string{path}}
	cmd := &cobra.Command{}
	cmd.SetContext(t.Context())
	var output bytes.Buffer
	cmd.SetOut(&output)

	err = replay.Run(cmd)
	// Let the progress bar's internal ticker observe completion before the bubble exits.
	time.Sleep(100 * time.Millisecond)

	failed := 0
	switch scenario {
	case "partial API failure":
		failed = 1
	case "partial refresh failure":
		failed = 2
	case "all API failures", "all refresh failures":
		failed = 3
	}
	if failed == 0 {
		require.NoError(t, err)
		assert.Contains(t, output.String(), "Successfully finished operations for all 3 SLOs.")
	} else {
		require.EqualError(t, err, fmt.Sprintf("%d of 3 Replay requests failed", failed))
		assert.Contains(t, output.String(), fmt.Sprintf(
			"Successfully finished operations for %d and failed for %d SLOs:", 3-failed, failed))
	}
	assert.Contains(t, output.String(), "[3/3]")
	if strings.Contains(scenario, "refresh") {
		assert.EqualValues(t, failed+1, fixture.tokenRequests.Load())
		assert.Contains(t, output.String(), "token refresh rejected")
		assert.Len(t, fixture.attempts, 3-failed)
	} else {
		assert.Equal(t, []string{"slo-a", "slo-b", "slo-c"}, fixture.attempts)
		if failed > 0 {
			assert.Contains(t, output.String(), "replay rejected")
		}
	}
	if !queued {
		assert.Len(t, fixture.statusChecks, 3-failed)
	}
	if failed > 0 {
		_, summary, found := strings.Cut(output.String(), "SLOs:\n - ")
		require.True(t, found)
		assert.Contains(t, summary, `"slo":"slo-b"`)
		if failed > 1 {
			assert.Contains(t, summary, `"slo":"slo-c"`)
		}
		if failed == 3 {
			assert.Contains(t, summary, `"slo":"slo-a"`)
		}
	}
}

type replayRunFixture struct {
	queued               bool
	scenario             string
	tokenRequests        atomic.Int32
	availabilityRequests atomic.Int32
	attempts             []string
	statusChecks         []string
}

func (f *replayRunFixture) roundTrip(t *testing.T, request *http.Request) (*http.Response, error) {
	t.Helper()
	recorder := httptest.NewRecorder()
	recorder.Header().Set("Content-Type", "application/json")
	var response any
	switch request.URL.Path {
	case "/oauth2/test/v1/token":
		if f.tokenRequests.Add(1) > 1 {
			recorder.WriteHeader(http.StatusUnauthorized)
			response = map[string]string{"error": "token refresh rejected"}
			break
		}
		claims, err := json.Marshal(map[string]any{
			"iss": "https://idp.example.com/oauth2/test", "cid": "test-client",
			"exp":        time.Now().Add(10 * time.Minute).Unix(),
			"m2mProfile": map[string]string{"organization": "test-organization", "environment": "api.example.com"},
		})
		require.NoError(t, err)
		token := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`)) + "." +
			base64.RawURLEncoding.EncodeToString(claims) + ".c2lnbmF0dXJl"
		response = map[string]string{"access_token": token}
	case endpointPlanInfo:
		response = PlaylistConfiguration{EnabledPlaylists: f.queued}
	case "/get/slo":
		if f.scenario == "preflight failure" {
			response = []v1alpha.GenericObject{}
			break
		}
		return replaySLOTestResponse(t, "project", request.URL.Query()[objectsV1.QueryKeyName]), nil
	case "/internal/timemachine/availability":
		response = replayV1.ReplayAvailability{Available: true}
		if f.availabilityRequests.Add(1) == 3 && f.scenario == "all refresh failures" {
			time.Sleep(11 * time.Minute)
		}
	case "/timetravel":
		assert.Equal(t, http.MethodPost, request.Method)
		var run replayV1.RunRequest
		require.NoError(t, json.NewDecoder(request.Body).Decode(&run))
		f.attempts = append(f.attempts, run.SLO)
		if f.scenario == "all API failures" || f.scenario == "partial API failure" && run.SLO == "slo-b" {
			recorder.WriteHeader(http.StatusBadRequest)
			response = sdk.APIErrors{Errors: []sdk.APIError{{Title: "replay rejected"}}}
		} else {
			recorder.WriteHeader(http.StatusCreated)
			if f.queued && f.scenario == "partial refresh failure" {
				time.Sleep(11 * time.Minute)
			}
		}
	case "/timetravel/slo-a", "/timetravel/slo-b", "/timetravel/slo-c":
		f.statusChecks = append(f.statusChecks, strings.TrimPrefix(request.URL.Path, "/timetravel/"))
		response = map[string]any{"status": map[string]string{"status": "completed"}}
		if f.scenario == "partial refresh failure" {
			time.Sleep(11 * time.Minute)
		}
	default:
		t.Errorf("unexpected request: %s %s", request.Method, request.URL)
		recorder.WriteHeader(http.StatusNotFound)
	}
	if response != nil {
		require.NoError(t, json.NewEncoder(recorder).Encode(response))
	}
	return recorder.Result(), nil
}

func TestReplayCLIExitStatus(t *testing.T) {
	binaryDir := t.TempDir()
	build := exec.CommandContext(t.Context(), "make", "build", "BIN_DIR="+binaryDir)
	build.Dir = ".."
	buildOutput, err := build.CombinedOutput()
	require.NoError(t, err, "%s", buildOutput)

	for _, scenario := range []string{"success", "partial API failure", "all API failures", "preflight failure"} {
		t.Run(scenario, func(t *testing.T) {
			fixture := replayRunFixture{queued: true, scenario: scenario}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				response, responseErr := fixture.roundTrip(t, request)
				if !assert.NoError(t, responseErr) {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				defer func() { _ = response.Body.Close() }()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(response.StatusCode)
				_, copyErr := io.Copy(w, response.Body)
				assert.NoError(t, copyErr)
			}))
			t.Cleanup(server.Close)
			path := filepath.Join(t.TempDir(), "replays.json")
			require.NoError(t, os.WriteFile(path, []byte(`[
				{"project":"project","slo":"slo-a","from":"2026-01-01T00:00:00Z"},
				{"project":"project","slo":"slo-b","from":"2026-01-01T00:00:00Z"},
				{"project":"project","slo":"slo-c","from":"2026-01-01T00:00:00Z"}
			]`), 0o600))
			command := exec.CommandContext(t.Context(), filepath.Join(binaryDir, "sloctl"),
				"--no-config-file", "replay", "--file", path)
			command.Env = slices.DeleteFunc(os.Environ(), func(value string) bool {
				return strings.HasPrefix(value, "SLOCTL_")
			})
			command.Env = append(command.Env, "SLOCTL_NO_NOTIFICATIONS=1", "SLOCTL_DISABLE_OKTA=true",
				"SLOCTL_ORGANIZATION=test-organization", "SLOCTL_URL="+server.URL)

			var stderr bytes.Buffer
			command.Stderr = &stderr
			stdout, runErr := command.Output()
			server.Close()

			if scenario == "success" {
				require.NoError(t, runErr, "%s", stderr.String())
				assert.Contains(t, stderr.String(), "Successfully finished operations for all 3 SLOs.")
			} else {
				var exitErr *exec.ExitError
				require.ErrorAs(t, runErr, &exitErr, "%s", stderr.String())
				assert.Equal(t, 1, exitErr.ExitCode(), "%s", stderr.String())
			}
			assert.Empty(t, stdout)
			if scenario == "preflight failure" {
				assert.Empty(t, fixture.attempts)
				assert.Contains(t, stderr.String(), "Some of the SLOs marked for Replay were not found")
			} else {
				assert.Equal(t, []string{"slo-a", "slo-b", "slo-c"}, fixture.attempts)
			}
			switch scenario {
			case "partial API failure":
				assert.Contains(t, stderr.String(), "Error: 1 of 3 Replay requests failed\n")
			case "all API failures":
				assert.Contains(t, stderr.String(), "Error: 3 of 3 Replay requests failed\n")
			}
		})
	}
}

func TestGetReplaySLOsBatchesByProject(t *testing.T) {
	for _, count := range []int{0, 1, 49, 50, 51, 100, 101} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			replays := make([]ReplayConfig, 0, count+1)
			wantNames := make([]string, count)
			for i := range count {
				name := fmt.Sprintf("slo-%03d", i)
				wantNames[i] = name
				replays = append(replays, ReplayConfig{
					Project: "target-project", SLO: name,
					SourceSLO: &replayV1.SourceSLO{Project: "source-project", SLO: name},
				})
			}
			if count > 0 {
				replays = append(replays, replays[0])
			}
			fetched := make(map[string][]string)
			batchSizes := make(map[string][]int)
			client := newReplaySLOTestClient(t, func(request *http.Request) (*http.Response, error) {
				assert.Equal(t, "/get/slo", request.URL.Path)
				assert.Equal(t, http.MethodGet, request.Method)
				project := request.Header.Get(sdk.HeaderProject)
				assert.Contains(t, []string{"target-project", "source-project"}, project)
				names := request.URL.Query()[objectsV1.QueryKeyName]
				assert.NotEmpty(t, names)
				assert.LessOrEqual(t, len(names), 50)
				fetched[project] = append(fetched[project], names...)
				batchSizes[project] = append(batchSizes[project], len(names))
				return replaySLOTestResponse(t, project, names), nil
			})
			replay := ReplayCmd{client: client}

			slos, err := replay.getReplaySLOs(t.Context(), replays)

			require.NoError(t, err)
			assert.Len(t, slos, count*2)
			if count == 0 {
				assert.Empty(t, fetched)
				return
			}
			for _, project := range []string{"target-project", "source-project"} {
				assert.Equal(t, wantNames, fetched[project])
				assert.Len(t, batchSizes[project], int(math.Ceil(float64(count)/50)))
			}
			matched, missing := matchReplaysToSLOs(replays, slos)
			assert.Empty(t, missing)
			assert.Equal(t, replays, matched)
		})
	}
}

func TestGetReplaySLOsStopsOnBatchFailure(t *testing.T) {
	for _, batchErr := range []error{errors.New("fetch failed"), context.Canceled} {
		t.Run(batchErr.Error(), func(t *testing.T) {
			replays := make([]ReplayConfig, 101)
			for i := range replays {
				replays[i] = ReplayConfig{Project: "project", SLO: fmt.Sprintf("slo-%03d", i)}
			}
			requests := 0
			client := newReplaySLOTestClient(t, func(request *http.Request) (*http.Response, error) {
				requests++
				if requests == 2 {
					return nil, batchErr
				}
				return replaySLOTestResponse(t, "project", request.URL.Query()[objectsV1.QueryKeyName]), nil
			})
			replay := ReplayCmd{client: client}

			slos, err := replay.getReplaySLOs(t.Context(), replays)

			require.ErrorIs(t, err, batchErr)
			assert.ErrorContains(t, err, "failed to get SLOs in 'project' Project")
			assert.Nil(t, slos)
			assert.Equal(t, 2, requests)
		})
	}
}

func TestVerifySLOsReportsMissingSLOAfterFetchingAllBatches(t *testing.T) {
	replays := make([]ReplayConfig, 101)
	for i := range replays {
		replays[i] = ReplayConfig{Project: "project", SLO: fmt.Sprintf("slo-%03d", i)}
	}
	requests := 0
	client := newReplaySLOTestClient(t, func(request *http.Request) (*http.Response, error) {
		requests++
		assert.Equal(t, "/get/slo", request.URL.Path)
		names := request.URL.Query()[objectsV1.QueryKeyName]
		names = slices.DeleteFunc(names, func(name string) bool { return name == "slo-000" })
		return replaySLOTestResponse(t, "project", names), nil
	})
	replay := ReplayCmd{client: client}

	verified, err := replay.verifySLOs(t.Context(), replays)

	require.EqualError(t, err, "Some of the SLOs marked for Replay were not found or"+
		" you don't have permissions to view them: \n - 'slo-000' SLO in 'project' Project")
	assert.Nil(t, verified)
	assert.Equal(t, 3, requests)
}

func newReplaySLOTestClient(t *testing.T, transport func(*http.Request) (*http.Response, error)) *sdk.Client {
	t.Helper()
	useGenericObjects := v1alphaParser.UseGenericObjects
	v1alphaParser.UseGenericObjects = true
	t.Cleanup(func() { v1alphaParser.UseGenericObjects = useGenericObjects })
	client, err := sdk.NewClient(&sdk.Config{DisableOkta: true, Project: sdk.ProjectsWildcard})
	require.NoError(t, err)
	client.HTTP = &http.Client{Transport: replayRoundTripper(transport)}
	return client
}

func replaySLOTestResponse(t *testing.T, project string, names []string) *http.Response {
	t.Helper()
	objects := make([]v1alpha.GenericObject, len(names))
	for i, name := range names {
		objects[i] = v1alpha.GenericObject{
			"apiVersion": "n9/v1alpha", "kind": "SLO",
			"metadata": map[string]any{"name": name, "project": project},
			"spec":     map[string]any{},
		}
	}
	recorder := httptest.NewRecorder()
	require.NoError(t, json.NewEncoder(recorder).Encode(objects))
	return recorder.Result()
}

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

func BenchmarkMatchReplaysToSLOs(b *testing.B) {
	for _, count := range []int{1000, 15000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			replays := make([]ReplayConfig, count)
			slos := make([]replaySLO, 0, count*2)
			for i := range count {
				name := fmt.Sprintf("slo-%05d", i)
				replays[i] = ReplayConfig{
					Project: "target-project", SLO: name,
					SourceSLO: &replayV1.SourceSLO{Project: "source-project", SLO: name},
				}
				slos = append(slos,
					replaySLO{project: "target-project", name: name},
					replaySLO{project: "source-project", name: name},
				)
			}
			b.ResetTimer()
			for range b.N {
				matched, missing := matchReplaysToSLOs(replays, slos)
				if len(matched) != count || len(missing) != 0 {
					b.Fatalf("matched %d replays with %d missing SLOs", len(matched), len(missing))
				}
			}
		})
	}
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
