package budgetadjustments

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

	"github.com/nobl9/sloctl/internal/printer"
)

type GetCmd struct {
	client           *sdk.Client
	outputFormat     string
	fieldSeparator   string
	recordSeparator  string
	out              io.Writer
	adjustment       string
	from, to         TimeValue
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
var example string

func NewGetCmd(client *sdk.Client) *cobra.Command {
	get := &GetCmd{out: os.Stdout}

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Return a list of events for given Adjustment with related SLOs.",
		Long: "This endpoint only return past and ongoing events (events that are already started)." +
			"Please see Editing budget adjustments." +
			"Maximum 500 events can be returned." +
			"Optional filtering for specific SLO (only one). If SLO is defined we will return only events for that SLO and the result will also include other SLOs that this events have. Sorted by eventStart.",
		Example: example,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			get.client = client
			project, _ := cmd.Flags().GetString(flagProject)
			sloName, _ := cmd.Flags().GetString(flagSloName)
			if project != "" {
				cmd.MarkFlagRequired(flagSloName)
			}
			if sloName != "" {
				cmd.MarkFlagRequired(flagProject)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error { return get.run(cmd) },
	}

	registerOutputFormatFlags(cmd, &get.outputFormat, &get.fieldSeparator, &get.recordSeparator)
	registerAdjustmentFlag(cmd, &get.adjustment)
	registerProjectFlag(cmd, &get.project)
	registerSloNameFlag(cmd, &get.sloName)
	registerFromFlag(cmd, &get.from)
	registerToFlag(cmd, &get.to)

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

	resBody, err := doRequest(
		g.client,
		cmd.Context(),
		http.MethodGet,
		fmt.Sprintf("%s/%s/events", budgetAdjustmentAPI, g.adjustment),
		values,
	)
	if err != nil {
		return errors.Wrap(err, "failed to get")
	}

	var events []Event
	if err := json.Unmarshal(resBody, &events); err != nil {
		return errors.Wrap(err, "failed parse response")
	}

	g.printObjects(events)

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
