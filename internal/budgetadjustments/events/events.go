package events

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nobl9/sloctl/internal/budgetadjustments/sdkclient"
)

func NewRootCmd(clientProvider sdkclient.SdkClientProvider) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Budget adjustments events management",
		Long:  "The 'events' command allows you to manage events related to SLO error budget adjustments",
	}
	cmd.PersistentFlags().BoolP("help", "h", false, fmt.Sprintf("Help for %s.", cmd.Name()))
	cmd.AddCommand(NewGetCmd(clientProvider))
	cmd.AddCommand(NewDeleteCmd(clientProvider))
	cmd.AddCommand(NewUpdateCmd(clientProvider))

	return cmd
}
