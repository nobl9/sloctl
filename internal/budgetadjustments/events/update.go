package events

import (
	"bytes"
	_ "embed"
	"encoding/json"
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
		Short:   "Update existing events with new values. Values for eventStart and eventEnd are required.",
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
	data, err := readFile(g.filepath)
	if err != nil {
		return errors.Wrap(err, "failed to read input data")
	}
	var yamlData []UpdateEvent
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return errors.Wrap(err, "failed to load input data")
	}
	jsonData, err := json.Marshal(yamlData)
	if err != nil {
		return errors.Wrap(err, "failed to load input data")
	}

	_, err = DoRequest(
		g.client,
		cmd.Context(),
		http.MethodPut,
		fmt.Sprintf("%s/%s/events/update", BudgetAdjustmentAPI, g.adjustment),
		nil,
		bytes.NewReader(jsonData),
	)
	if err != nil {
		return err
	}

	return nil
}
