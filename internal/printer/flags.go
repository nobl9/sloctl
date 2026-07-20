package printer

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nobl9/sloctl/internal/csv"
)

const OutputFlagName = "output"

// MustRegisterFlags registers flags related to printing structured data.
func (o *Printer) MustRegisterFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().VarP(
		&o.config.OutputFormat,
		OutputFlagName,
		"o",
		fmt.Sprintf(`Output format: one of %s.`, strings.Join(toStringSlice(o.config.SupportedFromats), "|")),
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

	o.jq.MustRegisterFlags(cmd)
}

func toStringSlice[T ~string](s []T) []string {
	result := make([]string, len(s))
	for i := range s {
		result[i] = string(s[i])
	}
	return result
}
