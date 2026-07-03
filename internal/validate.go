package internal

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nobl9/govy/pkg/govy"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/nobl9/nobl9-go/manifest"
	v1alphaSLO "github.com/nobl9/nobl9-go/manifest/v1alpha/slo"
	"github.com/nobl9/nobl9-go/sdk"
	dataSourceV1 "github.com/nobl9/nobl9-go/sdk/endpoints/datasource/v1"
	objectsV1 "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v1"

	"github.com/nobl9/sloctl/internal/csv"
	"github.com/nobl9/sloctl/internal/flags"
	"github.com/nobl9/sloctl/internal/printer"
)

const (
	defaultValidateSLILast     = 15 * time.Minute
	maxValidateSLITimeRange    = time.Hour
	maxValidateSLIQueryCount   = 50
	validateSLIStatusSuccess   = "success"
	validateSLIStatusFailed    = "failed"
	validateSLIMetricRaw       = "rawMetric"
	validateSLIMetricCount     = "countMetrics"
	validateSLIMetricIndicator = "indicator.rawMetric"
)

var errSLIValidationFailed = stderrors.New("SLI validation failed")

type ValidateCmd struct {
	client *sdk.Client
}

func (r *RootCmd) NewValidateCmd() *cobra.Command {
	validate := &ValidateCmd{}
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate Nobl9 resources.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(validate.NewSLICmd(r.GetClient))
	return cmd
}

type ValidateSLICmd struct {
	validate *ValidateCmd
	printer  *printer.Printer

	definitionPaths []string
	project         string
	sloFilter       string
	objectiveFilter string
	last            time.Duration
	from            time.Time
	to              time.Time

	sloName           string
	projectFlagWasSet bool
	now               func() time.Time
}

func (v *ValidateCmd) NewSLICmd(clientProvider func() *sdk.Client) *cobra.Command {
	validateSLI := &ValidateSLICmd{
		validate: v,
		printer: printer.NewPrinter(printer.Config{
			OutputFormat: printer.YAMLFormat,
			SupportedFromats: []printer.Format{
				printer.YAMLFormat,
				printer.JSONFormat,
				printer.CSVFormat,
			},
		}),
		last: defaultValidateSLILast,
		now:  func() time.Time { return time.Now().UTC() },
	}

	cmd := &cobra.Command{
		Use:   "sli [slo-name]",
		Short: "Validate SLI queries by querying data source values.",
		Long: "Validate SLI queries by querying data source values for SLO manifests or an existing SLO. " +
			"By default, it validates the last 15 minutes.",
		Args: validateSLI.arguments,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			validateSLI.validate.client = clientProvider()
		},
		RunE: func(cmd *cobra.Command, args []string) error { return validateSLI.Run(cmd) },
	}

	validateSLI.printer.MustRegisterFlags(cmd)
	registerFileFlag(cmd, false, &validateSLI.definitionPaths)
	cmd.Flags().StringVarP(&validateSLI.project, "project", "p", "",
		"Specifies the Project for the SLO selected by name, or assigns a default Project to SLOs read from a file.")
	cmd.Flags().StringVar(&validateSLI.sloFilter, "slo", "", "Filters SLOs read from a file by name.")
	cmd.Flags().StringVar(&validateSLI.objectiveFilter, "objective", "", "Filters SLO objectives by name.")
	cmd.Flags().DurationVar(&validateSLI.last, "last", defaultValidateSLILast,
		"Sets a relative validation time range ending now. Maximum value is 1h.")
	flags.RegisterTimeVar(cmd, &validateSLI.from, "from", "Sets the validation time range start.")
	flags.RegisterTimeVar(cmd, &validateSLI.to, "to", "Sets the validation time range end.")

	return cmd
}

func (v *ValidateSLICmd) Run(cmd *cobra.Command) error {
	if v.validate.client == nil {
		return stderrors.New("sloctl client is not initialized")
	}
	v.projectFlagWasSet = cmd.Flags().Changed("project")
	if v.projectFlagWasSet {
		v.validate.client.Config.Project = v.project
	} else if v.project == "" {
		v.project = v.validate.client.Config.Project
	}
	if v.project == sdk.ProjectsWildcard && len(v.definitionPaths) == 0 {
		return errProjectWildcardIsNotAllowed
	}
	if err := v.printer.Validate(); err != nil {
		return err
	}
	timeRange, err := v.resolveTimeRange(cmd)
	if err != nil {
		return err
	}
	slos, err := v.loadSLOs(cmd)
	if err != nil {
		return err
	}
	candidates, err := v.buildCandidates(slos)
	if err != nil {
		return err
	}
	if err = validateSLIPlanValidation.Validate(validateSLIPlan{QueryCount: len(candidates)}); err != nil {
		return err
	}
	spinner := NewSpinner(fmt.Sprintf("Validating %d SLI queries...", len(candidates)))
	spinner.Go()
	results := v.executeCandidates(cmd, timeRange, candidates)
	spinner.Stop()

	output := validateSLIOutput{
		TimeRange: validateSLITimeRange{From: timeRange.From, To: timeRange.To},
		Results:   results,
	}
	if err = v.print(cmd, output); err != nil {
		return err
	}
	if hasFailedSLIResults(results) {
		cmd.SilenceErrors = true
		return errSLIValidationFailed
	}
	return nil
}

func (v *ValidateSLICmd) arguments(cmd *cobra.Command, args []string) error {
	if err := requireFlagsIfFlagIsSet(
		cmd,
		printer.OutputFlagName,
		"jq",
		csv.RecordSeparatorFlag,
		csv.FieldSeparatorFlag,
	); err != nil {
		return err
	}
	options := validateSLIOptions{
		FileCount:  len(v.definitionPaths),
		ArgCount:   len(args),
		HasLast:    cmd.Flags().Changed("last"),
		HasFrom:    cmd.Flags().Changed("from"),
		HasTo:      cmd.Flags().Changed("to"),
		HasSLOFlag: cmd.Flags().Changed("slo"),
	}
	if err := validateSLIOptionsValidation.Validate(options); err != nil {
		return err
	}
	if len(args) == 1 {
		v.sloName = args[0]
	}
	return nil
}

func (v *ValidateSLICmd) resolveTimeRange(cmd *cobra.Command) (dataSourceV1.TimeRange, error) {
	now := v.now()
	options := validateSLITimeRangeOptions{
		Last:    v.last,
		From:    v.from,
		To:      v.to,
		HasLast: cmd.Flags().Changed("last"),
		HasFrom: cmd.Flags().Changed("from"),
		HasTo:   cmd.Flags().Changed("to"),
		Now:     now,
	}
	if err := validateSLITimeRangeValidation.Validate(options); err != nil {
		return dataSourceV1.TimeRange{}, err
	}
	to := v.to
	if !options.HasTo {
		to = now
	}
	from := v.from
	if !options.HasFrom {
		from = to.Add(-v.last)
	}
	return dataSourceV1.TimeRange{
		From: from.UTC(),
		To:   to.UTC(),
	}, nil
}

func (v *ValidateSLICmd) loadSLOs(cmd *cobra.Command) ([]v1alphaSLO.SLO, error) {
	if len(v.definitionPaths) > 0 {
		return v.loadSLOsFromFiles(cmd)
	}
	return v.loadSLOFromAPI(cmd)
}

func (v *ValidateSLICmd) loadSLOsFromFiles(cmd *cobra.Command) ([]v1alphaSLO.SLO, error) {
	objects, err := readObjectsDefinitions(
		cmd.Context(),
		v.validate.client.Config,
		cmd,
		v.definitionPaths,
		newFilesPrompt(false, false, 0),
		v.projectFlagWasSet,
	)
	if err != nil {
		return nil, err
	}
	return v.slosFromObjects(objects)
}

func (v *ValidateSLICmd) loadSLOFromAPI(cmd *cobra.Command) ([]v1alphaSLO.SLO, error) {
	objects, err := v.validate.client.Objects().V1().Get(
		cmd.Context(),
		manifest.KindSLO,
		http.Header{sdk.HeaderProject: []string{v.project}},
		url.Values{objectsV1.QueryKeyName: []string{v.sloName}},
	)
	if err != nil {
		return nil, err
	}
	if len(objects) == 0 {
		return nil, fmt.Errorf("SLO %q was not found in %q Project", v.sloName, v.project)
	}
	return v.slosFromObjects(objects)
}

func (v *ValidateSLICmd) slosFromObjects(objects []manifest.Object) ([]v1alphaSLO.SLO, error) {
	slos := make([]v1alphaSLO.SLO, 0, len(objects))
	for _, obj := range objects {
		if obj.GetKind() != manifest.KindSLO {
			continue
		}
		if v.sloFilter != "" && obj.GetName() != v.sloFilter {
			continue
		}
		slo, err := objectToSLO(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to decode %q SLO: %w", obj.GetName(), err)
		}
		slos = append(slos, slo)
	}
	if len(slos) == 0 {
		switch {
		case v.sloFilter != "":
			return nil, fmt.Errorf("no SLO named %q was found", v.sloFilter)
		case v.sloName != "":
			return nil, fmt.Errorf("no SLO named %q was found", v.sloName)
		default:
			return nil, stderrors.New("no SLO definitions were found")
		}
	}
	return slos, nil
}

func objectToSLO(obj manifest.Object) (v1alphaSLO.SLO, error) {
	var slo v1alphaSLO.SLO
	data, err := json.Marshal(obj)
	if err != nil {
		return slo, err
	}
	if err = json.Unmarshal(data, &slo); err != nil {
		return slo, err
	}
	return slo, nil
}

func (v *ValidateSLICmd) buildCandidates(slos []v1alphaSLO.SLO) ([]validateSLICandidate, error) {
	candidates := make([]validateSLICandidate, 0)
	for _, slo := range slos {
		if slo.Spec.Indicator == nil {
			continue
		}
		metricSource := slo.Spec.Indicator.MetricSource
		if slo.Spec.Indicator.RawMetric != nil && v.objectiveFilter == "" {
			candidates = append(candidates, validateSLICandidate{
				SLO:        slo.Metadata.Name,
				Project:    slo.Metadata.Project,
				Metric:     validateSLIMetricIndicator,
				DataSource: metricSource,
				Query: dataSourceV1.Query{
					RawMetric: &v1alphaSLO.RawMetricSpec{MetricQuery: slo.Spec.Indicator.RawMetric},
				},
			})
		}
		for _, objective := range slo.Spec.Objectives {
			if v.objectiveFilter != "" && objective.Name != v.objectiveFilter {
				continue
			}
			if objective.RawMetric != nil {
				candidates = append(candidates, validateSLICandidate{
					SLO:        slo.Metadata.Name,
					Project:    slo.Metadata.Project,
					Objective:  objective.Name,
					Metric:     validateSLIMetricRaw,
					DataSource: metricSource,
					Query: dataSourceV1.Query{
						RawMetric: objective.RawMetric,
					},
				})
			}
			if objective.CountMetrics != nil {
				candidates = append(candidates, validateSLICandidate{
					SLO:        slo.Metadata.Name,
					Project:    slo.Metadata.Project,
					Objective:  objective.Name,
					Metric:     validateSLIMetricCount,
					DataSource: metricSource,
					Query: dataSourceV1.Query{
						CountMetrics: objective.CountMetrics,
					},
				})
			}
		}
	}
	if v.objectiveFilter != "" && len(candidates) == 0 {
		return nil, fmt.Errorf("no SLI query was found for %q objective", v.objectiveFilter)
	}
	return candidates, nil
}

func (v *ValidateSLICmd) executeCandidates(
	cmd *cobra.Command,
	timeRange dataSourceV1.TimeRange,
	candidates []validateSLICandidate,
) []validateSLIResult {
	results := make([]validateSLIResult, len(candidates))
	group, ctx := errgroup.WithContext(cmd.Context())
	for i := range candidates {
		index := i
		candidate := candidates[i]
		group.Go(func() error {
			results[index] = v.executeCandidate(ctx, timeRange, candidate)
			return nil
		})
	}
	_ = group.Wait()
	return results
}

func (v *ValidateSLICmd) executeCandidate(
	ctx context.Context,
	timeRange dataSourceV1.TimeRange,
	candidate validateSLICandidate,
) validateSLIResult {
	result := validateSLIResult{
		SLO:       candidate.SLO,
		Project:   candidate.Project,
		Objective: candidate.Objective,
		Metric:    candidate.Metric,
		Status:    validateSLIStatusSuccess,
	}
	response, err := v.validate.client.DataSource().V1().Query(ctx, dataSourceV1.QueryRequest{
		DataSource: candidate.DataSource,
		Query:      candidate.Query,
		TimeRange:  timeRange,
	})
	if err != nil {
		result.Status = validateSLIStatusFailed
		result.Errors = apiErrorsFromError(err)
		return result
	}
	result.TimeSeries = toValidateSLITimeSeries(response.TimeSeries)
	return result
}

func (v *ValidateSLICmd) print(cmd *cobra.Command, output validateSLIOutput) error {
	outputFlag := cmd.Flag(printer.OutputFlagName)
	if outputFlag == nil || !outputFlag.Changed {
		printValidateSLIHumanSummary(cmd.OutOrStdout(), output)
		return nil
	}
	if v.printer.OutputFormat() == printer.CSVFormat {
		return v.printer.Print(output.CSVRows())
	}
	return v.printer.Print(output)
}

func printValidateSLIHumanSummary(out io.Writer, output validateSLIOutput) {
	status := "Valid"
	if hasFailedSLIResults(output.Results) {
		status = "Invalid"
	}
	_, _ = fmt.Fprintln(out, status)
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintf(
		out,
		"Validated %d %s for %d %s.\n",
		len(output.Results),
		validateSLIPlural(len(output.Results), "SLI query", "SLI queries"),
		output.SLOCount(),
		validateSLIPlural(output.SLOCount(), "SLO", "SLOs"),
	)
	_, _ = fmt.Fprintf(
		out,
		"Time range: %s - %s\n",
		output.TimeRange.From.Format(flags.TimeLayout),
		output.TimeRange.To.Format(flags.TimeLayout),
	)
	if len(output.Results) == 0 {
		return
	}
	_, _ = fmt.Fprintln(out)
	for i, result := range output.Results {
		if i > 0 {
			_, _ = fmt.Fprintln(out)
		}
		_, _ = fmt.Fprintln(out, result.HumanName())
		if len(result.Errors) > 0 {
			for _, apiErr := range result.Errors {
				for _, line := range strings.Split(validateSLIHumanError(apiErr), "\n") {
					_, _ = fmt.Fprintf(out, "  %s\n", line)
				}
			}
			continue
		}
		if len(result.TimeSeries) == 0 {
			_, _ = fmt.Fprintln(out, "  no time series returned")
			continue
		}
		for _, ts := range result.TimeSeries {
			_, _ = fmt.Fprintf(out, "  %s\n", ts.HumanSummary())
		}
	}
}

type validateSLICandidate struct {
	SLO        string
	Project    string
	Objective  string
	Metric     string
	DataSource v1alphaSLO.MetricSourceSpec
	Query      dataSourceV1.Query
}

type validateSLIOutput struct {
	TimeRange validateSLITimeRange `json:"timeRange" yaml:"timeRange"`
	Results   []validateSLIResult  `json:"results" yaml:"results"`
}

func (o validateSLIOutput) SLOCount() int {
	seen := make(map[string]struct{}, len(o.Results))
	for _, result := range o.Results {
		seen[result.Project+"/"+result.SLO] = struct{}{}
	}
	return len(seen)
}

type validateSLITimeRange struct {
	From time.Time `json:"from" yaml:"from"`
	To   time.Time `json:"to" yaml:"to"`
}

type validateSLIResult struct {
	SLO        string                  `json:"slo" yaml:"slo"`
	Project    string                  `json:"project" yaml:"project"`
	Objective  string                  `json:"objective,omitempty" yaml:"objective,omitempty"`
	Metric     string                  `json:"metric" yaml:"metric"`
	Status     string                  `json:"status" yaml:"status"`
	TimeSeries []validateSLITimeSeries `json:"timeseries,omitempty" yaml:"timeseries,omitempty"`
	Errors     []sdk.APIError          `json:"errors,omitempty" yaml:"errors,omitempty"`
}

func (r validateSLIResult) HumanName() string {
	parts := []string{r.SLO + "/" + r.Project}
	if r.Objective != "" {
		parts = append(parts, r.Objective)
	}
	parts = append(parts, r.Metric)
	return strings.Join(parts, " ")
}

type validateSLITimeSeries struct {
	Name   string             `json:"name" yaml:"name"`
	Values []validateSLIValue `json:"values" yaml:"values"`
}

func (t validateSLITimeSeries) HumanSummary() string {
	if len(t.Values) == 0 {
		return fmt.Sprintf("%s: 0 points", t.Name)
	}
	minValue := t.Values[0].Value
	maxValue := t.Values[0].Value
	for _, value := range t.Values[1:] {
		minValue = min(minValue, value.Value)
		maxValue = max(maxValue, value.Value)
	}
	return fmt.Sprintf(
		"%s: %d %s, min %s, max %s",
		t.Name,
		len(t.Values),
		validateSLIPlural(len(t.Values), "point", "points"),
		validateSLIFormatFloat(minValue),
		validateSLIFormatFloat(maxValue),
	)
}

type validateSLIValue struct {
	Timestamp int64
	Value     float64
}

func (v validateSLIValue) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{v.Timestamp, v.Value})
}

func (v validateSLIValue) MarshalYAML() (any, error) {
	return []any{v.Timestamp, v.Value}, nil
}

type validateSLICSVRow struct {
	SLO          string `json:"slo"`
	Project      string `json:"project"`
	Objective    string `json:"objective"`
	Metric       string `json:"metric"`
	Status       string `json:"status"`
	TimeSeries   string `json:"timeseries"`
	Timestamp    string `json:"timestamp"`
	Value        string `json:"value"`
	ErrorTitle   string `json:"error_title"`
	ErrorCode    string `json:"error_code"`
	ErrorDetails string `json:"error_detail"`
}

func (o validateSLIOutput) CSVRows() []validateSLICSVRow {
	rows := make([]validateSLICSVRow, 0)
	for _, result := range o.Results {
		base := validateSLICSVRow{
			SLO:       result.SLO,
			Project:   result.Project,
			Objective: result.Objective,
			Metric:    result.Metric,
			Status:    result.Status,
		}
		if len(result.Errors) > 0 {
			for _, apiErr := range result.Errors {
				row := base
				row.ErrorTitle = apiErr.Title
				row.ErrorCode = apiErr.Code
				row.ErrorDetails = apiErr.Detail
				rows = append(rows, row)
			}
			continue
		}
		for _, ts := range result.TimeSeries {
			for _, value := range ts.Values {
				row := base
				row.TimeSeries = ts.Name
				row.Timestamp = strconv.FormatInt(value.Timestamp, 10)
				row.Value = strconv.FormatFloat(value.Value, 'f', -1, 64)
				rows = append(rows, row)
			}
		}
		if len(result.TimeSeries) == 0 {
			rows = append(rows, base)
		}
	}
	return rows
}

func apiErrorsFromError(err error) []sdk.APIError {
	var httpErr *sdk.HTTPError
	if stderrors.As(err, &httpErr) && len(httpErr.Errors) > 0 {
		return httpErr.Errors
	}
	var apiErrs sdk.APIErrors
	if stderrors.As(err, &apiErrs) && len(apiErrs.Errors) > 0 {
		return apiErrs.Errors
	}
	return []sdk.APIError{{
		Title:  "SLI validation failed",
		Detail: err.Error(),
	}}
}

func toValidateSLITimeSeries(input []dataSourceV1.TimeSeries) []validateSLITimeSeries {
	output := make([]validateSLITimeSeries, 0, len(input))
	for _, ts := range input {
		valuesCount := min(len(ts.Timestamps), len(ts.Values))
		values := make([]validateSLIValue, 0, valuesCount)
		for i := 0; i < valuesCount; i++ {
			values = append(values, validateSLIValue{
				Timestamp: ts.Timestamps[i],
				Value:     ts.Values[i],
			})
		}
		output = append(output, validateSLITimeSeries{
			Name:   validateSLITimeSeriesName(ts.Measurement),
			Values: values,
		})
	}
	return output
}

func validateSLITimeSeriesName(measurement string) string {
	switch measurement {
	case "raw_metric":
		return "raw"
	case "good_count":
		return "good"
	case "bad_count":
		return "bad"
	case "total_count":
		return "total"
	default:
		return measurement
	}
}

func validateSLIHumanError(apiErr sdk.APIError) string {
	detail := validateSLIHumanErrorDetail(apiErr.Detail)
	if detail == "" {
		return apiErr.Title
	}
	if apiErr.Title == "" {
		return detail
	}
	if strings.Contains(detail, "\n") {
		return apiErr.Title + ":\n" + detail
	}
	return apiErr.Title + ": " + detail
}

func validateSLIHumanErrorDetail(detail string) string {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return ""
	}
	var formatted strings.Builder
	for {
		start := strings.IndexByte(detail, '{')
		if start == -1 {
			formatted.WriteString(detail)
			return formatted.String()
		}
		formatted.WriteString(detail[:start])
		jsonDetail, consumed, ok := validateSLIFormatJSONFromPrefix(detail[start:])
		if !ok {
			formatted.WriteByte(detail[start])
			detail = detail[start+1:]
			continue
		}
		formatted.WriteString(jsonDetail)
		detail = detail[start+consumed:]
	}
}

func validateSLIFormatJSONFromPrefix(input string) (string, int, bool) {
	decoder := json.NewDecoder(strings.NewReader(input))
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return "", 0, false
	}
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, raw, "", "  "); err != nil {
		return "", 0, false
	}
	return formatted.String(), int(decoder.InputOffset()), true
}

func validateSLIPlural(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func validateSLIFormatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func hasFailedSLIResults(results []validateSLIResult) bool {
	for _, result := range results {
		if result.Status == validateSLIStatusFailed {
			return true
		}
	}
	return false
}

type validateSLIOptions struct {
	FileCount  int
	ArgCount   int
	HasLast    bool
	HasFrom    bool
	HasTo      bool
	HasSLOFlag bool
}

var validateSLIOptionsValidation = govy.New[validateSLIOptions](
	govy.For(govy.GetSelf[validateSLIOptions]()).Rules(
		govy.NewRule(func(o validateSLIOptions) error {
			switch {
			case o.ArgCount > 1:
				return stderrors.New("accepts at most one SLO name argument")
			case o.FileCount == 0 && o.ArgCount == 0:
				return stderrors.New("provide an SLO name or use the --file flag")
			case o.FileCount > 0 && o.ArgCount > 0:
				return stderrors.New("the --file flag and SLO name argument are mutually exclusive")
			case o.FileCount == 0 && o.HasSLOFlag:
				return stderrors.New("the --slo flag can be used only with --file")
			case o.HasLast && (o.HasFrom || o.HasTo):
				return stderrors.New("the --last flag cannot be used with --from or --to")
			case o.HasTo && !o.HasFrom:
				return stderrors.New("the --to flag requires --from")
			default:
				return nil
			}
		}),
	),
)

type validateSLITimeRangeOptions struct {
	Last    time.Duration
	From    time.Time
	To      time.Time
	Now     time.Time
	HasLast bool
	HasFrom bool
	HasTo   bool
}

var validateSLITimeRangeValidation = govy.New[validateSLITimeRangeOptions](
	govy.For(govy.GetSelf[validateSLITimeRangeOptions]()).Rules(
		govy.NewRule(func(o validateSLITimeRangeOptions) error {
			if !o.HasFrom {
				if o.Last <= 0 {
					return stderrors.New("the --last flag must be greater than zero")
				}
				if o.Last > maxValidateSLITimeRange {
					return fmt.Errorf("the --last flag cannot exceed %s", maxValidateSLITimeRange)
				}
				return nil
			}
			to := o.To
			if !o.HasTo {
				to = o.Now
			}
			if !o.From.Before(to) {
				return stderrors.New("the --from value must be before --to")
			}
			if to.Sub(o.From) > maxValidateSLITimeRange {
				return fmt.Errorf("the requested time range cannot exceed %s", maxValidateSLITimeRange)
			}
			return nil
		}),
	),
)

type validateSLIPlan struct {
	QueryCount int
}

var validateSLIPlanValidation = govy.New[validateSLIPlan](
	govy.For(govy.GetSelf[validateSLIPlan]()).Rules(
		govy.NewRule(func(p validateSLIPlan) error {
			switch {
			case p.QueryCount == 0:
				return stderrors.New("no SLI queries were found")
			case p.QueryCount > maxValidateSLIQueryCount:
				return fmt.Errorf("too many SLI queries selected: %d, maximum is %d",
					p.QueryCount, maxValidateSLIQueryCount)
			default:
				return nil
			}
		}),
	),
)
