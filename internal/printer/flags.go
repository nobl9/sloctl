package printer

import (
	"github.com/spf13/cobra"

	"github.com/nobl9/sloctl/internal/csv"
)

func (o *Printer) MustRegisterFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().VarP(
		&o.config.OutputFormat,
		"output",
		"o",
		`Output format: one of yaml|json|csv.`,
	)

	cmd.PersistentFlags().StringVarP(
		&o.config.CSVFieldSeparator,
		csv.FieldSeparatorFlag,
		"",
		csv.DefaultFieldSeparator,
		"Field Separator for CSV.",
	)
	if err := cmd.PersistentFlags().MarkHidden(csv.FieldSeparatorFlag); err != nil {
		panic(err)
	}

	cmd.PersistentFlags().StringVarP(
		&o.config.CSVRecordSeparator,
		csv.RecordSeparatorFlag,
		"",
		csv.DefaultRecordSeparator,
		"Record Separator for CSV.",
	)
	if err := cmd.PersistentFlags().MarkHidden(csv.RecordSeparatorFlag); err != nil {
		panic(err)
	}
}
