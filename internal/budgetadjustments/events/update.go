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

type UpdateCmd struct {
	client     *sdk.Client
	filepath   string
	adjustment string
}

//go:embed examples/update_example.sh
var updateExample string

func NewUpdateCmd(clientProvider sdkclient.SdkClientProvider) *cobra.Command {
	update := &UpdateCmd{}

	cmd := &cobra.Command{
		Use:     "update",
		Short:   "Update existing past events with new values. Values for eventStart and eventEnd are required.",
		Example: updateExample,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			update.client = clientProvider.GetClient()
		},
		RunE: func(cmd *cobra.Command, args []string) error { return update.run(cmd) },
	}

	mustRegisterFileFlag(cmd, &update.filepath)
	mustRegisterAdjustmentFlag(cmd, &update.adjustment)

	return cmd
}

func (g *UpdateCmd) run(cmd *cobra.Command) error {
	docs, err := getEventsStringsFromFile(g.filepath)
	if err != nil {
		return errors.Wrap(err, "failed to read events form file")
	}

	for _, doc := range docs {
		jsonBytes, err := yaml.YAMLToJSON([]byte(doc))
		if err != nil {
			return errors.Wrap(err, "failed to convert input data to JSON")
		}
		_, err = DoRequest(
			g.client,
			cmd.Context(),
			http.MethodPut,
			fmt.Sprintf("%s/%s/events/update", BudgetAdjustmentAPI, g.adjustment),
			nil,
			bytes.NewReader(jsonBytes),
		)
		if err != nil {
			return err
		}
	}

	return nil
}
