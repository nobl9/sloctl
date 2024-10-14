package budgetadjustments

import (
	_ "embed"
	"fmt"

	"github.com/nobl9/nobl9-go/sdk"
	"github.com/spf13/cobra"

	"github.com/nobl9/sloctl/internal/budgetadjustments/events"
)

func NewRootCmd(client *sdk.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "budgetadjustments",
		Short: "Budget adjustments managment",
	}
	cmd.PersistentFlags().BoolP("help", "h", false, fmt.Sprintf("Help for %s.", cmd.Name()))
	cmd.AddCommand(events.NewRootCmd(client))
	return cmd
}
