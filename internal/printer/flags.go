package printer

import (
	"github.com/spf13/cobra"

	"github.com/nobl9/sloctl/internal/csv"
)

func MustRegisterOutputFormatFlags(
	cmd *cobra.Command,
	outputFormat, fieldSeparator, recordSeparator *string,
) {
	cmd.PersistentFlags().StringVarP(outputFormat, "output", "o", "yaml",
		`Output format: one of yaml|json|csv.`)

	cmd.PersistentFlags().StringVarP(fieldSeparator, csv.FieldSeparatorFlag, "",
		csv.DefaultFieldSeparator, "Field Separator for CSV.")

	cmd.PersistentFlags().StringVarP(recordSeparator, csv.RecordSeparatorFlag, "",
		csv.DefaultRecordSeparator, "Record Separator for CSV.")

	if err := cmd.PersistentFlags().MarkHidden(csv.RecordSeparatorFlag); err != nil {
		panic(err)
	}
}
