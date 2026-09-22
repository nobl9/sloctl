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
		Short: "Validate SLI queries",
		Long:  "Validate the SLI queries defined by existing SLOs or local SLO manifests.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(validate.NewSLICmd(r.GetClient))
	return cmd
}
