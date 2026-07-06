//go:build unit_test

package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobl9/nobl9-go/manifest"
	v1alphaSLO "github.com/nobl9/nobl9-go/manifest/v1alpha/slo"
	"github.com/nobl9/nobl9-go/sdk"
	dataSourceV1 "github.com/nobl9/nobl9-go/sdk/endpoints/datasource/v1"

	"github.com/nobl9/sloctl/internal/printer"
)

func TestValidateSLICommandIsRegistered(t *testing.T) {
	cmd, _, err := NewRootCmd().Find([]string{"validate", "sli"})

	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "sli", cmd.Name())
}

func TestValidateSLIArgumentsRequireOutputForJQ(t *testing.T) {
	for name, test := range map[string]struct {
		SetOutput     bool
		ExpectedError string
	}{
		"reject jq without output": {
			ExpectedError: "--jq flag can only be set if --output flag is also provided",
		},
		"allow jq with output": {
			SetOutput: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			cmd := (&ValidateCmd{}).NewSLICmd(func() *sdk.Client { return nil })
			require.NoError(t, cmd.PersistentFlags().Set("jq", ".results"))
			if test.SetOutput {
				require.NoError(t, cmd.PersistentFlags().Set(printer.OutputFlagName, string(printer.JSONFormat)))
			}

			err := cmd.Args(cmd, []string{"checkout"})

			if test.ExpectedError == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, test.ExpectedError)
			}
		})
	}
}

func TestValidateSLIExecutesRequestsInParallelAndPreservesResultOrder(t *testing.T) {
	timeRange := dataSourceV1.TimeRange{
		From: time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 7, 2, 10, 15, 0, 0, time.UTC),
	}
	rt := newBlockingValidateSLIRoundTripper(t, 2)
	validateSLI := &ValidateSLICmd{validate: &ValidateCmd{client: newValidateSLITestClient(t, rt)}}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	candidates := []validateSLICandidate{
		rawMetricCandidate("checkout", "default", "latency", "up"),
		countMetricsCandidate("checkout", "default", "availability", "good", "total"),
	}

	resultsCh := make(chan []validateSLIResult, 1)
	go func() {
		resultsCh <- validateSLI.executeCandidates(cmd, timeRange, candidates)
	}()

	select {
	case <-rt.allStarted:
	case <-time.After(time.Second):
		t.Fatal("expected both query requests to start before any response was released")
	}
	close(rt.release)

	results := <-resultsCh
	require.Len(t, results, 2)
	assert.Equal(t, "latency", results[0].Objective)
	assert.Equal(t, "availability", results[1].Objective)
	require.Len(t, results[0].TimeSeries, 1)
	assert.Equal(t, "raw", results[0].TimeSeries[0].Name)
	require.Len(t, results[1].TimeSeries, 2)
	assert.Equal(t, "good", results[1].TimeSeries[0].Name)
	assert.Equal(t, "total", results[1].TimeSeries[1].Name)

	requests := rt.requests()
	require.Len(t, requests, 2)
	for _, request := range requests {
		assert.Equal(t, "/api/agentcommander/v2/commands/timeseries/execute", request.Path)
		assert.Equal(t, "source", request.Body.DataSource.Name)
		assert.Equal(t, "default", request.Body.DataSource.Project)
		assert.Equal(t, manifest.KindDirect, request.Body.DataSource.Kind)
		assert.Equal(t, timeRange.From, request.Body.Command.Payload.TimeRange.From)
		assert.Equal(t, timeRange.To, request.Body.Command.Payload.TimeRange.To)
	}
}

func TestValidateSLIMapsAPIErrorsToFailedResult(t *testing.T) {
	rt := validateSLIErrorRoundTripper{t: t}
	validateSLI := &ValidateSLICmd{validate: &ValidateCmd{client: newValidateSLITestClient(t, rt)}}
	result := validateSLI.executeCandidate(
		context.Background(),
		dataSourceV1.TimeRange{
			From: time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC),
			To:   time.Date(2026, 7, 2, 10, 15, 0, 0, time.UTC),
		},
		rawMetricCandidate("checkout", "default", "latency", "bad query"),
	)

	assert.Equal(t, validateSLIStatusFailed, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "datasource command failed", result.Errors[0].Title)
	assert.Equal(t, "datasource_command_failed", result.Errors[0].Code)
	assert.Equal(t, "invalid query", result.Errors[0].Detail)
	assert.Empty(t, result.TimeSeries)
}

func TestValidateSLIBuildCandidatesFiltersObjectivesAndCountsCountMetricsAsOneQuery(t *testing.T) {
	query := "up"
	validateSLI := &ValidateSLICmd{objectiveFilter: "availability"}
	slo := v1alphaSLO.SLO{
		Metadata: v1alphaSLO.Metadata{Name: "checkout", Project: "default"},
		Spec: v1alphaSLO.Spec{
			Indicator: &v1alphaSLO.Indicator{MetricSource: validateSLITestMetricSource()},
			Objectives: []v1alphaSLO.Objective{
				{
					ObjectiveBase: v1alphaSLO.ObjectiveBase{Name: "latency"},
					RawMetric: &v1alphaSLO.RawMetricSpec{
						MetricQuery: &v1alphaSLO.MetricSpec{
							Prometheus: &v1alphaSLO.PrometheusMetric{PromQL: &query},
						},
					},
				},
				{
					ObjectiveBase: v1alphaSLO.ObjectiveBase{Name: "availability"},
					CountMetrics: &v1alphaSLO.CountMetricsSpec{
						GoodMetric:  prometheusMetricSpec("good"),
						TotalMetric: prometheusMetricSpec("total"),
					},
				},
			},
		},
	}

	candidates, err := validateSLI.buildCandidates([]v1alphaSLO.SLO{slo})

	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.Equal(t, validateSLIMetricCount, candidates[0].Metric)
	assert.Equal(t, "availability", candidates[0].Objective)
}

func TestValidateSLIOutputOmitsEmptyFieldsAndSerializesValuesAsPairs(t *testing.T) {
	output := validateSLIOutput{
		TimeRange: validateSLITimeRange{
			From: time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC),
			To:   time.Date(2026, 7, 2, 10, 15, 0, 0, time.UTC),
		},
		Results: []validateSLIResult{
			{
				SLO:       "checkout",
				Project:   "default",
				Objective: "latency",
				Metric:    validateSLIMetricRaw,
				Status:    validateSLIStatusSuccess,
				TimeSeries: []validateSLITimeSeries{{
					Name: "raw",
					Values: []validateSLIValue{{
						Timestamp: 1782987721,
						Value:     0.41255780025567085,
					}},
				}},
			},
		},
	}

	data, err := json.Marshal(output)

	require.NoError(t, err)
	assert.NotContains(t, string(data), "errors")
	assert.JSONEq(t, `{
		"timeRange": {
			"from": "2026-07-02T10:00:00Z",
			"to": "2026-07-02T10:15:00Z"
		},
		"results": [{
			"slo": "checkout",
			"project": "default",
			"objective": "latency",
			"metric": "rawMetric",
			"status": "success",
			"timeseries": [{
				"name": "raw",
				"values": [[1782987721, 0.41255780025567085]]
			}]
		}]
	}`, string(data))
}

func TestValidateSLIPrintDefaultsToHumanSummary(t *testing.T) {
	output := validateSLITestOutput(validateSLIStatusSuccess)
	var out bytes.Buffer
	validateSLI := &ValidateSLICmd{}
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	err := validateSLI.print(cmd, output)

	require.NoError(t, err)
	assert.Equal(t, `Valid

Validated 1 SLI query for 1 SLO.
Time range: 2026-07-02T10:00:00Z - 2026-07-02T10:15:00Z

checkout/default latency rawMetric
  raw: 2 points, min 0.41255780025567085, max 0.5
`, out.String())
}

func TestValidateSLIPrintHumanSummaryForFailures(t *testing.T) {
	output := validateSLITestOutput(validateSLIStatusFailed)
	output.Results[0].TimeSeries = nil
	output.Results[0].Errors = []sdk.APIError{{
		Title:  "datasource command failed",
		Detail: "invalid query",
	}}
	var out bytes.Buffer
	validateSLI := &ValidateSLICmd{}
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	err := validateSLI.print(cmd, output)

	require.NoError(t, err)
	assert.Equal(t, `Invalid

Validated 1 SLI query for 1 SLO.
Time range: 2026-07-02T10:00:00Z - 2026-07-02T10:15:00Z

checkout/default latency rawMetric
  datasource command failed: invalid query
`, out.String())
}

func TestValidateSLIPrintHumanSummaryFormatsJSONErrorDetails(t *testing.T) {
	output := validateSLITestOutput(validateSLIStatusFailed)
	output.Results[0].TimeSeries = nil
	output.Results[0].Errors = []sdk.APIError{{
		Title:  "datasource command failed",
		Detail: `{"status":"error","errorType":"bad_data","error":"1:118: parse error: unclosed left parenthesis"}`,
	}}
	var out bytes.Buffer
	validateSLI := &ValidateSLICmd{}
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	err := validateSLI.print(cmd, output)

	require.NoError(t, err)
	assert.Equal(t, `Invalid

Validated 1 SLI query for 1 SLO.
Time range: 2026-07-02T10:00:00Z - 2026-07-02T10:15:00Z

checkout/default latency rawMetric
  datasource command failed:
  {
    "status": "error",
    "errorType": "bad_data",
    "error": "1:118: parse error: unclosed left parenthesis"
  }
`, out.String())
}

func TestValidateSLIPrintHumanSummaryFormatsEmbeddedJSONErrorDetails(t *testing.T) {
	output := validateSLITestOutput(validateSLIStatusFailed)
	output.Results[0].TimeSeries = nil
	output.Results[0].Errors = []sdk.APIError{{
		Title: "datasource command failed",
		Detail: `countMetrics.good: failed to query metrics: unexpected status code 400: ` +
			`{"status":"error","errorType":"bad_data","error":"1:118: parse error: unclosed left parenthesis"}: ` +
			`invalid query`,
	}}
	var out bytes.Buffer
	validateSLI := &ValidateSLICmd{}
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	err := validateSLI.print(cmd, output)

	require.NoError(t, err)
	assert.Equal(t, `Invalid

Validated 1 SLI query for 1 SLO.
Time range: 2026-07-02T10:00:00Z - 2026-07-02T10:15:00Z

checkout/default latency rawMetric
  datasource command failed:
  countMetrics.good: failed to query metrics: unexpected status code 400:
  {
    "status": "error",
    "errorType": "bad_data",
    "error": "1:118: parse error: unclosed left parenthesis"
  }
  invalid query
`, out.String())
}

func TestValidateSLIPrintUsesStructuredOutputWhenOutputFlagIsSet(t *testing.T) {
	output := validateSLITestOutput(validateSLIStatusSuccess)
	var out bytes.Buffer
	validateSLI := &ValidateSLICmd{
		printer: printer.NewPrinter(printer.Config{
			Output:       &out,
			OutputFormat: printer.JSONFormat,
			SupportedFromats: []printer.Format{
				printer.YAMLFormat,
				printer.JSONFormat,
				printer.CSVFormat,
			},
		}),
	}
	cmd := &cobra.Command{}
	validateSLI.printer.MustRegisterFlags(cmd)
	require.NoError(t, cmd.PersistentFlags().Set(printer.OutputFlagName, string(printer.JSONFormat)))

	err := validateSLI.print(cmd, output)

	require.NoError(t, err)
	assert.JSONEq(t, `{
		"timeRange": {
			"from": "2026-07-02T10:00:00Z",
			"to": "2026-07-02T10:15:00Z"
		},
		"results": [{
			"slo": "checkout",
			"project": "default",
			"objective": "latency",
			"metric": "rawMetric",
			"status": "success",
			"timeseries": [{
				"name": "raw",
				"values": [[1782987721, 0.41255780025567085], [1782987781, 0.5]]
			}]
		}]
	}`, out.String())
}

func TestValidateSLICSVRowsUseBlankValuesForErrors(t *testing.T) {
	output := validateSLIOutput{
		Results: []validateSLIResult{
			{
				SLO:     "checkout",
				Project: "default",
				Metric:  validateSLIMetricRaw,
				Status:  validateSLIStatusFailed,
				Errors: []sdk.APIError{{
					Title:  "datasource command failed",
					Code:   "datasource_command_failed",
					Detail: "invalid query",
				}},
			},
		},
	}

	rows := output.CSVRows()

	require.Len(t, rows, 1)
	assert.Empty(t, rows[0].TimeSeries)
	assert.Empty(t, rows[0].Timestamp)
	assert.Empty(t, rows[0].Value)
	assert.Equal(t, "datasource command failed", rows[0].ErrorTitle)
}

func validateSLITestOutput(status string) validateSLIOutput {
	return validateSLIOutput{
		TimeRange: validateSLITimeRange{
			From: time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC),
			To:   time.Date(2026, 7, 2, 10, 15, 0, 0, time.UTC),
		},
		Results: []validateSLIResult{
			{
				SLO:       "checkout",
				Project:   "default",
				Objective: "latency",
				Metric:    validateSLIMetricRaw,
				Status:    status,
				TimeSeries: []validateSLITimeSeries{{
					Name: "raw",
					Values: []validateSLIValue{
						{Timestamp: 1782987721, Value: 0.41255780025567085},
						{Timestamp: 1782987781, Value: 0.5},
					},
				}},
			},
		},
	}
}

func TestValidateSLITimeRangeValidation(t *testing.T) {
	now := time.Date(2026, 7, 2, 10, 15, 0, 0, time.UTC)

	for name, test := range map[string]struct {
		Options validateSLITimeRangeOptions
	}{
		"last exceeds limit": {
			Options: validateSLITimeRangeOptions{Last: time.Hour + time.Second},
		},
		"from is after to": {
			Options: validateSLITimeRangeOptions{
				From:    now,
				To:      now.Add(-time.Minute),
				HasFrom: true,
				HasTo:   true,
			},
		},
		"explicit range exceeds limit": {
			Options: validateSLITimeRangeOptions{
				From:    now.Add(-2 * time.Hour),
				To:      now,
				HasFrom: true,
				HasTo:   true,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := validateSLITimeRangeFlags(test.Options)

			require.Error(t, err)
		})
	}
}

type validateSLIRequest struct {
	Path string
	Body validateSLIRequestBody
}

type validateSLIRequestBody struct {
	DataSource v1alphaSLO.MetricSourceSpec `json:"datasource"`
	Command    struct {
		Payload struct {
			RawMetric    *v1alphaSLO.RawMetricSpec    `json:"rawMetric"`
			CountMetrics *v1alphaSLO.CountMetricsSpec `json:"countMetrics"`
			TimeRange    dataSourceV1.TimeRange       `json:"timeRange"`
		} `json:"payload"`
	} `json:"command"`
}

type blockingValidateSLIRoundTripper struct {
	t          *testing.T
	waitFor    int32
	started    int32
	allStarted chan struct{}
	release    chan struct{}
	mu         sync.Mutex
	seen       []validateSLIRequest
}

func newBlockingValidateSLIRoundTripper(t *testing.T, waitFor int32) *blockingValidateSLIRoundTripper {
	return &blockingValidateSLIRoundTripper{
		t:          t,
		waitFor:    waitFor,
		allStarted: make(chan struct{}),
		release:    make(chan struct{}),
	}
}

func (r *blockingValidateSLIRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r.t.Helper()
	request := decodeValidateSLIRequest(r.t, req)
	r.mu.Lock()
	r.seen = append(r.seen, request)
	r.mu.Unlock()

	if atomic.AddInt32(&r.started, 1) == r.waitFor {
		close(r.allStarted)
	}
	<-r.release

	rec := httptest.NewRecorder()
	require.Equal(r.t, "/api/agentcommander/v2/commands/timeseries/execute", req.URL.Path)
	if request.Body.Command.Payload.CountMetrics != nil {
		require.NoError(r.t, json.NewEncoder(rec).Encode(dataSourceV1.QueryResponse{
			TimeSeries: []dataSourceV1.TimeSeries{
				{Measurement: "good_count", Timestamps: []int64{1782987721}, Values: []float64{98}},
				{Measurement: "total_count", Timestamps: []int64{1782987721}, Values: []float64{100}},
			},
		}))
		return rec.Result(), nil
	}
	require.NoError(r.t, json.NewEncoder(rec).Encode(dataSourceV1.QueryResponse{
		TimeSeries: []dataSourceV1.TimeSeries{
			{Measurement: "raw_metric", Timestamps: []int64{1782987721}, Values: []float64{0.42}},
		},
	}))
	return rec.Result(), nil
}

func (r *blockingValidateSLIRoundTripper) requests() []validateSLIRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]validateSLIRequest(nil), r.seen...)
}

type validateSLIErrorRoundTripper struct {
	t *testing.T
}

func (r validateSLIErrorRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r.t.Helper()
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/json")
	rec.WriteHeader(http.StatusUnprocessableEntity)
	require.NoError(r.t, json.NewEncoder(rec).Encode(sdk.APIErrors{
		Errors: []sdk.APIError{{
			Title:  "datasource command failed",
			Code:   "datasource_command_failed",
			Detail: "invalid query",
		}},
	}))
	return rec.Result(), nil
}

func decodeValidateSLIRequest(t *testing.T, req *http.Request) validateSLIRequest {
	t.Helper()
	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	require.NoError(t, req.Body.Close())
	req.Body = io.NopCloser(bytes.NewReader(body))
	var request validateSLIRequestBody
	require.NoError(t, json.Unmarshal(body, &request))
	return validateSLIRequest{Path: req.URL.Path, Body: request}
}

func newValidateSLITestClient(t *testing.T, rt http.RoundTripper) *sdk.Client {
	t.Helper()
	apiURL, err := url.Parse("https://example.com/api")
	require.NoError(t, err)
	client, err := sdk.NewClient(&sdk.Config{
		DisableOkta:  true,
		Organization: "test-org",
		Project:      "default",
		URL:          apiURL,
	})
	require.NoError(t, err)
	client.HTTP = &http.Client{Transport: rt}
	return client
}

func rawMetricCandidate(slo, project, objective, query string) validateSLICandidate {
	return validateSLICandidate{
		SLO:        slo,
		Project:    project,
		Objective:  objective,
		Metric:     validateSLIMetricRaw,
		DataSource: validateSLITestMetricSource(),
		Query: dataSourceV1.Query{
			RawMetric: &v1alphaSLO.RawMetricSpec{MetricQuery: prometheusMetricSpec(query)},
		},
	}
}

func countMetricsCandidate(slo, project, objective, good, total string) validateSLICandidate {
	return validateSLICandidate{
		SLO:        slo,
		Project:    project,
		Objective:  objective,
		Metric:     validateSLIMetricCount,
		DataSource: validateSLITestMetricSource(),
		Query: dataSourceV1.Query{
			CountMetrics: &v1alphaSLO.CountMetricsSpec{
				GoodMetric:  prometheusMetricSpec(good),
				TotalMetric: prometheusMetricSpec(total),
			},
		},
	}
}

func validateSLITestMetricSource() v1alphaSLO.MetricSourceSpec {
	return v1alphaSLO.MetricSourceSpec{
		Name:    "source",
		Project: "default",
		Kind:    manifest.KindDirect,
	}
}

func prometheusMetricSpec(query string) *v1alphaSLO.MetricSpec {
	return &v1alphaSLO.MetricSpec{
		Prometheus: &v1alphaSLO.PrometheusMetric{PromQL: &query},
	}
}
