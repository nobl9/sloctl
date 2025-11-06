package events

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/nobl9/nobl9-go/sdk"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/nobl9/sloctl/internal/budgetadjustments/sdkclient"
	"github.com/nobl9/sloctl/internal/printer"
)

type GetCmd struct {
	client           *sdk.Client
	printer          *printer.Printer
	adjustment       string
	from, to         time.Time
	project, sloName string
}

//go:embed examples/get_example.sh
var getExample string

func NewGetCmd(clientProvider sdkclient.SdkClientProvider) *cobra.Command {
	get := &GetCmd{
		printer: printer.NewPrinter(printer.Config{}),
	}

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

	get.printer.MustRegisterFlags(cmd)
	mustRegisterAdjustmentFlag(cmd, &get.adjustment)
	registerProjectFlag(cmd, &get.project)
	registerSloNameFlag(cmd, &get.sloName)
	mustRegisterFromFlag(cmd, &get.from)
	mustRegisterToFlag(cmd, &get.to)

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

	if err := g.printer.Print(events); err != nil {
		return errors.Wrap(err, "failed to print objects")
	}

	return nil
}
