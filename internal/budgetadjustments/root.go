package budgetadjustments

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nobl9/sloctl/internal/budgetadjustments/events"
	"github.com/nobl9/sloctl/internal/budgetadjustments/sdkclient"
)

func NewRootCmd(clientProvider sdkclient.SdkClientProvider) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "budgetadjustments",
		Short: "Budget adjustments management",
	}
	cmd.PersistentFlags().BoolP("help", "h", false, fmt.Sprintf("Help for %s.", cmd.Name()))
	cmd.AddCommand(events.NewRootCmd(clientProvider))
	return cmd
}
