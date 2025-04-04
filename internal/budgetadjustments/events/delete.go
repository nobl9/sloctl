package events

import (
	"bytes"
	_ "embed"
	"fmt"
	"net/http"

	"github.com/nobl9/go-yaml"
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/nobl9/sloctl/internal/budgetadjustments/sdkclient"
)

type DeleteCmd struct {
	client     *sdk.Client
	filepath   string
	adjustment string
}

//go:embed examples/delete_example.sh
var deleteExample string

func NewDeleteCmd(clientProvider sdkclient.SdkClientProvider) *cobra.Command {
	deleteCmd := &DeleteCmd{}

	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "Delete existing past events.",
		Example: deleteExample,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			deleteCmd.client = clientProvider.GetClient()
		},
		RunE: func(cmd *cobra.Command, args []string) error { return deleteCmd.run(cmd) },
	}

	mustRegisterFileFlag(cmd, &deleteCmd.filepath)
	mustRegisterAdjustmentFlag(cmd, &deleteCmd.adjustment)

	return cmd
}

func (g *DeleteCmd) run(cmd *cobra.Command) error {
	docs, err := getEventsStringsFromFile(g.filepath)
	if err != nil {
		return errors.Wrap(err, "failed to read events form file")
	}

	for _, doc := range docs {
		jsonBytes, err := yaml.YAMLToJSON([]byte(doc))
		if err != nil {
			return errors.Wrap(err, "failed to convert input data to JSON")
		}
		if _, err = DoRequest(
			g.client,
			cmd.Context(),
			http.MethodPost,
			fmt.Sprintf("%s/%s/events/delete", BudgetAdjustmentAPI, g.adjustment),
			nil,
			bytes.NewReader(jsonBytes),
		); err != nil {
			return err
		}
	}

	return nil
}
