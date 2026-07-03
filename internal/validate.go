package internal

import (
	"github.com/spf13/cobra"

	"github.com/nobl9/nobl9-go/sdk"
)

type ValidateCmd struct {
	client *sdk.Client
}

func (r *RootCmd) NewValidateCmd() *cobra.Command {
	validate := &ValidateCmd{}
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate Nobl9 resources.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(validate.NewSLICmd(r.GetClient))
	return cmd
}
