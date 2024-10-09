package internal

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/colorstring"
	"github.com/nobl9/go-yaml"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	v1alphaSLO "github.com/nobl9/nobl9-go/manifest/v1alpha/slo"
	"github.com/nobl9/nobl9-go/sdk"
	objectsV1 "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v1"
	sdkModels "github.com/nobl9/nobl9-go/sdk/models"
)

type ReplayCmd struct {
	client      *sdk.Client
	from        TimeValue
	configPaths []string
	sloName     string
	project     string
	deleteAll   bool
}

//go:embed replay_example.sh
var replayExample string

func (r *RootCmd) NewReplayCmd() *cobra.Command {
	replay := &ReplayCmd{}

	cmd := &cobra.Command{
		Use:   "replay",
		Short: "Retrieve historical SLI data and recalculate their SLO error budgets.",
		Long: "Replay pulls in the historical data while your SLO collects new data in real-time. " +
			"The historical and current data are merged, producing an error budget calculated for the entire period. " +
			"Refer to https://docs.nobl9.com/replay for more details on Replay.\n\n" +
			"The 'replay' command allows you to import data for multiple SLOs in bulk. " +
			"Before running the Replays it will verify if the SLOs you've provided are eligible for Replay. " +
			"It will only run a single Replay simultaneously (current limit for concurrent Replays). " +
			"When any Replay fails, it will attempt the import for the next SLO. " +
			"Importing data takes time: Replay for a single SLO may take several minutes up to an hour. " +
			"During that time, the command keeps on running, periodically checking the status of Replay. " +
			"If you cancel the program execution at any time, the current Replay in progress will not be revoked.",
		Example: replayExample,
		Args:    replay.arguments,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			replay.client = r.GetClient()
			if replay.project != "" {
				replay.client.Config.Project = replay.project
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error { return replay.Run(cmd) },
	}

	registerFileFlag(cmd, false, &replay.configPaths)
	cmd.Flags().StringVarP(&replay.project, "project", "p", "",
		`Specifies the Project for the SLOs you want to Replay.`)
	cmd.Flags().Var(&replay.from, "from", "Sets the start of Replay time window.")

	cmd.AddCommand(replay.AddDeleteCommand())

	return cmd
}

func (r *ReplayCmd) Run(cmd *cobra.Command) error {
	if r.client.Config.Project == "*" {
		return errProjectWildcardIsNotAllowed
	}
	replays, err := r.prepareConfigs()
	if err != nil {
		return err
	}
	_, err = r.RunReplays(cmd, replays)
	return err
}

func (r *ReplayCmd) RunReplays(cmd *cobra.Command, replays []ReplayConfig) (failedReplays int, err error) {
	if err = r.verifySLOs(cmd.Context(), replays); err != nil {
		return 0, err
	}

	failedIndexes := make([]int, 0)
	for i, replay := range replays {
		cmd.Println(colorstring.Color(fmt.Sprintf(
			"[cyan][%d/%d][reset] SLO: %s, Project: %s, From: %s, To: %s",
			i+1, len(replays), replay.SLO, replay.Project,
			replay.From.Format(timeLayout), time.Now().In(replay.From.Location()).Format(timeLayout))))

		spinner := NewSpinner("Importing data...")
		spinner.Go()
		err = r.runReplay(cmd.Context(), replay)
		spinner.Stop()

		if err != nil {
			cmd.Println(colorstring.Color("[red]Import failed:[reset] " + err.Error()))
			failedIndexes = append(failedIndexes, i)
			continue
		}
		cmd.Println(colorstring.Color("[green]Import succeeded![reset]"))
	}
	if len(replays) > 0 {
		r.printSummary(cmd, replays, failedIndexes)
	}
	return len(failedIndexes), nil
}

type ReplayConfig struct {
	Project   string                     `json:"project" validate:"required"`
	SLO       string                     `json:"slo" validate:"required"`
	From      time.Time                  `json:"from" validate:"required"`
	SourceSLO *sdkModels.ReplaySourceSLO `json:"sourceSLO,omitempty"`

	metricSource v1alphaSLO.MetricSourceSpec
}

// We can only give an estimate here, since there's no 'to' for Replay.
// We're always sending the duration for Replay, but never specify when it starts.
// The start timestamp of Replay is beyond our control.
// However, it's better to import more than less, that's why we're extending
// the duration here to account for that unknown offset factor.
const startOffsetMinutes = 5

func (r ReplayConfig) ToReplay(timeNow time.Time) sdkModels.Replay {
	windowDuration := timeNow.Sub(r.From)
	return sdkModels.Replay{
		Project: r.Project,
		Slo:     r.SLO,
		Duration: sdkModels.ReplayDuration{
			Unit:  sdkModels.DurationUnitMinute,
			Value: startOffsetMinutes + int(windowDuration.Minutes()),
		},
		SourceSLO: r.SourceSLO,
	}
}

func (r *ReplayCmd) prepareConfigs() ([]ReplayConfig, error) {
	var replays []ReplayConfig
	val := validator.New()

	unique := make(map[string]struct{})
	key := func(s, p string) string { return s + p }
	for _, path := range r.configPaths {
		c, err := r.readConfigFile(path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read Replay config from: %s", path)
		}
		for i := range c {
			if c[i].From.IsZero() {
				c[i].From = r.from.Time
			}
			if len(c[i].Project) == 0 {
				c[i].Project = r.client.Config.Project
			}
			if err = val.Struct(c[i]); err != nil {
				return nil, errors.Wrap(err, "Replay config entry failed validation")
			}
			k := key(c[i].SLO, c[i].Project)
			if _, exists := unique[k]; exists {
				return nil, errors.Errorf(
					"duplicated Replay definition detected for '%s' SLO in '%s' Project",
					c[i].SLO, c[i].Project)
			}
			unique[k] = struct{}{}
		}
		replays = append(replays, c...)
	}

	if len(replays) == 0 {
		replays = append(replays, ReplayConfig{
			Project: r.client.Config.Project,
			SLO:     r.sloName,
			From:    r.from.Time,
		})
	}
	return replays, nil
}

func (r *ReplayCmd) arguments(cmd *cobra.Command, args []string) error {
	if len(r.configPaths) == 0 && len(args) == 0 {
		_ = cmd.Usage()
		return errReplayInvalidOptions
	}
	if len(args) > 1 {
		return errReplayTooManyArgs
	}
	if len(r.configPaths) > 0 && len(args) == 1 {
		return errReplayInvalidOptions
	}
	if len(args) == 1 && r.from.IsZero() {
		return errReplayMissingFromArg
	}
	if len(args) == 1 {
		r.sloName = args[0]
	}
	return nil
}

type TimeValue struct{ time.Time }

const (
	timeLayout       = time.RFC3339
	timeLayoutString = "RFC3339"
)

func (t *TimeValue) String() string {
	if t.IsZero() {
		return ""
	}
	return t.Format(timeLayout)
}

func (t *TimeValue) Set(s string) (err error) {
	t.Time, err = time.Parse(timeLayout, s)
	return
}

func (t *TimeValue) Type() string {
	return "time"
}

func (r *ReplayCmd) readConfigFile(path string) ([]ReplayConfig, error) {
	data, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return nil, err
	}
	var replays []ReplayConfig
	if err = yaml.Unmarshal(data, &replays); err != nil {
		return nil, err
	}
	return replays, nil
}

// averageReplayDuration is used to calculate when running bulk Replay to calculate time offset for each SLO.
const averageReplayDuration = 20 * time.Minute

func (r *ReplayCmd) verifySLOs(ctx context.Context, replays []ReplayConfig) error {
	sloNames := make([]string, 0, len(replays))
	for _, r := range replays {
		sloNames = append(sloNames, r.SLO)
		if r.SourceSLO != nil {
			// Add source SLOs to the list of SLOs to check for existence and permissions.
			sloNames = append(sloNames, r.SourceSLO.Slo)
		}
	}
	if r.client.Config.Project == "" {
		r.client.Config.Project = sdk.ProjectsWildcard
	}

	// Find non-existent or RBAC protected SLOs.
	// We're also filling the Data Source spec here for ReplayConfig.
	data, err := r.doRequest(
		ctx,
		http.MethodGet,
		endpointGetSLO,
		"*",
		url.Values{objectsV1.QueryKeyName: sloNames},
		nil)
	if err != nil {
		return err
	}
	var slos []v1alphaSLO.SLO
	if err = json.Unmarshal(data, &slos); err != nil {
		return err
	}
	missingSLOs := make([]string, 0)
	compositeSLOs := make([]string, 0)
outer:
	for i := range replays {
		for j := range slos {
			if replays[i].SLO == slos[j].Metadata.Name && replays[i].Project == slos[j].Metadata.Project {
				if slos[j].Spec.HasCompositeObjectives() {
					compositeSLOs = append(compositeSLOs,
						fmt.Sprintf("Replay is unavailable for composite SLOs: '%s' SLO in '%s' Project",
							slos[j].Metadata.Name,
							slos[j].Metadata.Project))
					continue outer
				}
				replays[i].metricSource = slos[j].Spec.Indicator.MetricSource
				continue outer
			}
		}
		missingSLOs = append(
			missingSLOs,
			fmt.Sprintf("'%s' SLO in '%s' Project", replays[i].SLO, replays[i].Project))
	}
	if len(missingSLOs) > 0 {
		return errors.Errorf("Some of the SLOs marked for Replay were not found or"+
			" you don't have permissions to view them: \n - %s", strings.Join(missingSLOs, "\n - "))
	}
	if len(compositeSLOs) > 0 {
		return errors.Errorf("The following SLOs are composite and not eligible for Replay: \n - %s",
			strings.Join(compositeSLOs, "\n - "))
	}

	// Check Replay availability.
	notAvailable := make([]string, 0)
	mu := sync.Mutex{}
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(10)
	for i := range replays {
		eg.Go(func() error {
			c := replays[i]
			timeNow := time.Now()
			tt := c.ToReplay(timeNow)
			offset := i * int(averageReplayDuration.Minutes())
			expectedDuration := offset + tt.Duration.Value
			av, err := r.getReplayAvailability(ctx, c, tt.Duration.Unit, expectedDuration)
			if err != nil {
				return errors.Wrapf(err,
					"failed to check Replay availability for '%s' SLO in '%s' Project", c.SLO, c.Project)
			}
			if !av.Available {
				mu.Lock()
				notAvailable = append(notAvailable,
					fmt.Sprintf("['%s' SLO in '%s' Project] %s",
						c.SLO, c.Project, r.replayUnavailabilityReasonExplanation(
							av.Reason,
							c,
							time.Duration(expectedDuration)*time.Minute,
							time.Duration(offset)*time.Minute,
							timeNow)))
				mu.Unlock()
			}
			return nil
		})
	}
	if err = eg.Wait(); err != nil {
		return err
	}
	if len(notAvailable) > 0 {
		return errors.Errorf("The following SLOs are not available for Replay: \n - %s",
			strings.Join(notAvailable, "\n - "))
	}
	return nil
}

const replayStatusCheckInterval = 30 * time.Second

func (r *ReplayCmd) runReplay(ctx context.Context, config ReplayConfig) error {
	_, err := r.doRequest(ctx, http.MethodPost, endpointReplayPost, config.Project, nil, config.ToReplay(time.Now()))
	if err != nil {
		return errors.Wrap(err, "failed to start new Replay")
	}
	ticker := time.NewTicker(replayStatusCheckInterval)
	for {
		select {
		case <-ticker.C:
			status, err := r.getReplayStatus(ctx, config)
			if err != nil {
				return errors.Wrap(err, "failed to get for Replay status")
			}
			switch status {
			case sdkModels.ReplayStatusFailed:
				return errors.New("Replay has failed")
			case sdkModels.ReplayStatusCompleted:
				return nil
			default:
				continue
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (r *ReplayCmd) getReplayAvailability(
	ctx context.Context,
	config ReplayConfig,
	durationUnit string,
	durationValue int,
) (availability sdkModels.ReplayAvailability, err error) {
	values := url.Values{
		"dataSource":        {config.metricSource.Name},
		"dataSourceKind":    {config.metricSource.Kind.String()},
		"dataSourceProject": {config.metricSource.Project},
		"durationUnit":      {durationUnit},
		"durationValue":     {strconv.Itoa(durationValue)},
	}
	data, err := r.doRequest(ctx, http.MethodGet, endpointReplayGetAvailability, config.Project, values, nil)
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, &availability); err != nil {
		return
	}
	return
}

func (r *ReplayCmd) getReplayStatus(
	ctx context.Context,
	config ReplayConfig,
) (string, error) {
	data, err := r.doRequest(
		ctx,
		http.MethodGet,
		fmt.Sprintf(endpointReplayGetStatus, config.SLO),
		config.Project,
		nil,
		nil)
	if err != nil {
		return "", err
	}
	var ws sdkModels.ReplayWithStatus
	if err = json.Unmarshal(data, &ws); err != nil {
		return "", err
	}
	return ws.Status.Status, nil
}

const (
	endpointReplayPost            = "/timetravel"
	endpointReplayDelete          = "/timetravel"
	endpointReplayGetStatus       = "/timetravel/%s"
	endpointReplayGetAvailability = "/internal/timemachine/availability"
	endpointGetSLO                = "/get/slo"
)

func (r *ReplayCmd) doRequest(
	ctx context.Context,
	method, endpoint, project string,
	values url.Values,
	payload interface{},
) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(payload); err != nil {
			return nil, err
		}
		body = buf
	}
	header := http.Header{sdk.HeaderProject: []string{project}}
	req, err := r.client.CreateRequest(ctx, method, endpoint, header, values, body)
	if err != nil {
		return nil, err
	}
	resp, err := r.client.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("bad response (status: %d): %s", resp.StatusCode, string(data))
	}
	return io.ReadAll(resp.Body)
}

func (r *ReplayCmd) replayUnavailabilityReasonExplanation(
	reason string,
	replay ReplayConfig,
	expectedDuration, replayOffset time.Duration,
	timeNow time.Time,
) string {
	switch reason {
	case sdkModels.ReplayIntegrationDoesNotSupportReplay:
		return fmt.Sprintf("%s Data Source does not support Replay yet", replay.metricSource.Kind)
	case sdkModels.ReplayAgentVersionDoesNotSupportReplay:
		return fmt.Sprintf("Update your '%s' Agent in '%s' Project"+
			" version to the latest to use Replay for this Data Source.",
			replay.metricSource.Name, replay.metricSource.Project)
	case sdkModels.ReplayMaxHistoricalDataRetrievalTooLow:
		var offsetNotice string
		if replayOffset > 0 {
			offsetNotice = fmt.Sprintf(
				" + %s (offset for each next replay run in bulk is increased by an average of %s)",
				replayOffset, averageReplayDuration)
		}
		return fmt.Sprintf(
			"Value configured for spec.historicalDataRetrieval.maxDuration.value"+
				" for '%s' Data Source in '%s' Project is lower than the duration you're trying to run Replay for."+
				" The calculated duration is: %s, calculated from: %s (time.Now) - %s (from)"+
				" + %dm (start offset to ensure Replay covers the desired time window) %s."+
				" Edit the Data Source and run Replay once again.",
			replay.metricSource.Name, replay.metricSource.Project, expectedDuration.String(),
			timeNow.Format(timeLayout), r.from.Format(timeLayout), startOffsetMinutes, offsetNotice)
	case sdkModels.ReplayConcurrentReplayRunsLimitExhausted:
		return "You've exceeded the limit of concurrent Replay runs. Wait until the current Replay(s) are done."
	case sdkModels.ReplayUnknownAgentVersion:
		return "Your Agent isn't connected to the Data Source. Deploy the Agent and run Replay once again."
	default:
		return reason
	}
}

func (r *ReplayCmd) printSummary(cmd *cobra.Command, replays []ReplayConfig, failedIndexes []int) {
	if len(failedIndexes) == 0 {
		cmd.Printf("\nSuccessfully imported data for all %d SLOs.\n", len(replays))
	} else {
		failedDetails := make([]string, 0, len(failedIndexes))
		for _, i := range failedIndexes {
			fr, _ := json.Marshal(replays[i])
			failedDetails = append(failedDetails, string(fr))
		}
		cmd.Printf("\nSuccessfully imported data for %d and failed for %d SLOs:\n - %s\n",
			len(replays)-len(failedIndexes), len(failedIndexes), strings.Join(failedDetails, "\n - "))
	}
}
