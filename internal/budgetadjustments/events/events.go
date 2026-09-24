package events

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nobl9/sloctl/internal/budgetadjustments/sdkclient"
)

func NewRootCmd(clientProvider sdkclient.SdkClientProvider) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Get, update, or delete budget adjustment events",
		Long: `Manage event occurrences associated with budget adjustments. These
commands do not modify budget adjustment definitions.`,
	}
	cmd.PersistentFlags().BoolP("help", "h", false, fmt.Sprintf("Help for %s.", cmd.Name()))
	cmd.AddCommand(NewGetCmd(clientProvider))
	cmd.AddCommand(NewDeleteCmd(clientProvider))
	cmd.AddCommand(NewUpdateCmd(clientProvider))

	return cmd
}
