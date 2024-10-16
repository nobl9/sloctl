package events

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/nobl9/nobl9-go/sdk"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/nobl9/sloctl/internal/budgetadjustments/flags"
	"github.com/nobl9/sloctl/internal/budgetadjustments/request"
	"github.com/nobl9/sloctl/internal/printer"
	"github.com/nobl9/sloctl/internal/sdkclient"
)

type GetCmd struct {
	client           *sdk.Client
	outputFormat     string
	fieldSeparator   string
	recordSeparator  string
	out              io.Writer
	adjustment       string
	from, to         flags.TimeValue
	project, sloName string
}

type SLO struct {
	Project string `json:"project" validate:"required"`
	Name    string `json:"name"    validate:"required"`
}

type Event struct {
	EventStart time.Time `json:"eventStart"`
	EventEnd   time.Time `json:"eventEnd"`
	Slos       []SLO     `json:"slos"`
}

//go:embed examples/get_example.sh
var getExample string

func NewGetCmd(clientProvider sdkclient.SdkClientProvider) *cobra.Command {
	get := &GetCmd{out: os.Stdout}

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Return a list of events for given Adjustment with related SLOs",
		Long: "Returns a list of events for the specified adjustment along with related **SLO**." +
			"This command returns past and ongoing events (events that have already started)." +
			"The events 'get' command can return a maximum of 250 events. You can optionally filter for a specific SLO (only one)." +
			"If an SLO is defined, only events for that SLO will be returned, but the results will also include other SLOs associated with those events." +
			"The results are sorted by event start time.",
		Example: getExample,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			get.client = clientProvider.GetClient()
			project, _ := cmd.Flags().GetString(flags.FlagSloProject)
			sloName, _ := cmd.Flags().GetString(flags.FlagSloName)
			if project != "" {
				if err := cmd.MarkFlagRequired(flags.FlagSloName); err != nil {
					panic(err)
				}
			}
			if sloName != "" {
				if err := cmd.MarkFlagRequired(flags.FlagSloProject); err != nil {
					panic(err)
				}
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error { return get.run(cmd) },
	}

	flags.MustRegisterOutputFormatFlags(
		cmd,
		&get.outputFormat,
		&get.fieldSeparator,
		&get.recordSeparator,
	)
	flags.MustRegisterAdjustmentFlag(cmd, &get.adjustment)
	flags.RegisterProjectFlag(cmd, &get.project)
	flags.RegisterSloNameFlag(cmd, &get.sloName)
	flags.MustRegisterFromFlag(cmd, &get.from)
	flags.MustRegisterToFlag(cmd, &get.to)

	return cmd
}

func (g *GetCmd) run(cmd *cobra.Command) error {
	values := url.Values{"from": {g.from.String()}, "to": {g.to.String()}}
	if g.sloName != "" {
		values.Add("sloName", g.sloName)
	}
	if g.project != "" {
		values.Add("project", g.project)
	}

	resBody, err := request.DoRequest(
		g.client,
		cmd.Context(),
		http.MethodGet,
		fmt.Sprintf("%s/%s/events", request.BudgetAdjustmentAPI, g.adjustment),
		values,
	)
	if err != nil {
		return errors.Wrap(err, "failed to get events")
	}

	var events []Event
	if err := json.Unmarshal(resBody, &events); err != nil {
		return errors.Wrap(err, "failed parse response")
	}

	if err := g.printObjects(events); err != nil {
		return errors.Wrap(err, "failed to print objects")
	}

	return nil
}

func (g *GetCmd) printObjects(objects interface{}) error {
	p, err := printer.New(
		g.out,
		printer.Format(g.outputFormat),
		g.fieldSeparator,
		g.recordSeparator,
	)
	if err != nil {
		return err
	}
	if err = p.Print(objects); err != nil {
		return err
	}
	return nil
}
