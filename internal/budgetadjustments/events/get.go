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

	"github.com/nobl9/sloctl/internal/budgetadjustments/sdkclient"
	"github.com/nobl9/sloctl/internal/flags"
	"github.com/nobl9/sloctl/internal/printer"
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
	SLOs       []SLO     `json:"slos"`
}

//go:embed examples/get_example.sh
var getExample string

func NewGetCmd(clientProvider sdkclient.SdkClientProvider) *cobra.Command {
	get := &GetCmd{out: os.Stdout}

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Return a list of events for given Adjustment with related SLOs",
		Long: "Returns a list of events for the specified adjustment along with related **SLO**. " +
			"This command returns past and ongoing events (events that have already started). " +
			"The events 'get' command can return a maximum of 250 events. " +
			"You can optionally filter for a specific SLO (only one). " +
			"If an SLO is defined, only events for that SLO will be returned, " +
			"but the results will also include other SLOs associated with those events. " +
			"The results are sorted by event start time.",
		Example: getExample,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			get.client = clientProvider.GetClient()
			project, _ := cmd.Flags().GetString(FlagSloProject)
			sloName, _ := cmd.Flags().GetString(FlagSloName)
			if project != "" {
				if err := cmd.MarkFlagRequired(FlagSloName); err != nil {
					panic(err)
				}
			}
			if sloName != "" {
				if err := cmd.MarkFlagRequired(FlagSloProject); err != nil {
					panic(err)
				}
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error { return get.run(cmd) },
	}

	printer.MustRegisterOutputFormatFlags(
		cmd,
		&get.outputFormat,
		&get.fieldSeparator,
		&get.recordSeparator,
	)
	MustRegisterAdjustmentFlag(cmd, &get.adjustment)
	RegisterProjectFlag(cmd, &get.project)
	RegisterSloNameFlag(cmd, &get.sloName)
	MustRegisterFromFlag(cmd, &get.from)
	MustRegisterToFlag(cmd, &get.to)

	return cmd
}

func (g *GetCmd) run(cmd *cobra.Command) error {
	values := url.Values{"from": {g.from.String()}, "to": {g.to.String()}}
	if g.sloName != "" {
		values.Add("sloName", g.sloName)
	}
	if g.project != "" {
		values.Add("sloProject", g.project)
	}

	resBody, err := DoRequest(
		g.client,
		cmd.Context(),
		http.MethodGet,
		fmt.Sprintf("%s/%s/events", BudgetAdjustmentAPI, g.adjustment),
		values,
		nil,
	)
	if err != nil {
		return err
	}

	var events []Event
	if err := json.Unmarshal(resBody, &events); err != nil {
		return errors.Wrap(err, "failed to parse response")
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
