package jq

import "github.com/spf13/cobra"

func (e *ExpressionRunner) MustRegisterFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(
		&e.config.Expression,
		"jq",
		"",
		"",
		"Query to select values from the response using jq syntax",
	)
}
