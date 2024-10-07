package budgetadjustments

import (
	_ "embed"
	"fmt"

	"github.com/nobl9/nobl9-go/sdk"
	"github.com/spf13/cobra"
)

func NewRootCmd(client *sdk.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "budgetadjustments",
		Short: "Manage budgetadjustment.",
	}
	cmd.PersistentFlags().BoolP("help", "h", false, fmt.Sprintf("Help for %s.", cmd.Name()))
	cmd.AddCommand(NewGetCmd(client))
	return cmd
}
