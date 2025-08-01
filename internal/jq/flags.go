package jq

import "github.com/spf13/cobra"

func (e *ExpressionRunner) MustRegisterFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(
		&e.config.Expression,
		"jq",
		"q",
		"",
		"jq expression used to filter command results",
	)
}
