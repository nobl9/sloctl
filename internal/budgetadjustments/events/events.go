package events

import (
	_ "embed"
	"fmt"

	"github.com/nobl9/nobl9-go/sdk"
	"github.com/spf13/cobra"
)

func NewRootCmd(client *sdk.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Budget adjustments events managment",
	}
	cmd.PersistentFlags().BoolP("help", "h", false, fmt.Sprintf("Help for %s.", cmd.Name()))
	cmd.AddCommand(NewGetCmd(client))
	cmd.AddCommand(NewUpdateCmd(client))
	return cmd
}
