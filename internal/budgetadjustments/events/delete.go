package events

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/nobl9/nobl9-go/sdk"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/nobl9/sloctl/internal/budgetadjustments/sdkclient"
	"github.com/nobl9/sloctl/internal/flags"
)

type DeleteCmd struct {
	client          *sdk.Client
	filepath        string
	dryRun          bool
	outputFormat    string
	fieldSeparator  string
	recordSeparator string
	out             io.Writer
	adjustment      string
}

//go:embed examples/delete_example.sh
var deleteExample string

func NewDeleteCmd(clientProvider sdkclient.SdkClientProvider) *cobra.Command {
	deleteCmd := &DeleteCmd{out: os.Stdout}

	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "Delete existing events with new values. Values for eventStart and eventEnd are required.",
		Example: deleteExample,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			deleteCmd.client = clientProvider.GetClient()
			if deleteCmd.dryRun {
				flags.NotifyDryRunFlag()
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error { return deleteCmd.run(cmd) },
	}

	MustRegisterFileFlag(cmd, &deleteCmd.filepath)
	flags.RegisterDryRunFlag(cmd, &deleteCmd.dryRun)
	MustRegisterAdjustmentFlag(cmd, &deleteCmd.adjustment)

	return cmd
}

func (g *DeleteCmd) run(cmd *cobra.Command) error {
	data, err := read(g.filepath)
	if err != nil {
		return errors.Wrap(err, "failed to read input data")
	}
	var yamlData []Event
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return errors.Wrap(err, "failed to load input data")
	}
	jsonData, err := json.Marshal(yamlData)
	if err != nil {
		return errors.Wrap(err, "failed to convert input data to JSON")
	}

	if g.dryRun {
		return nil
	}

	if _, err = DoRequest(
		g.client,
		cmd.Context(),
		http.MethodPost,
		fmt.Sprintf("%s/%s/events/delete", BudgetAdjustmentAPI, g.adjustment),
		nil,
		bytes.NewReader(jsonData),
	); err != nil {
		return err
	}

	return nil
}
