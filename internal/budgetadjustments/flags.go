package budgetadjustments

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/nobl9/sloctl/internal/csv"
)

const (
	flagAdjustment = "adjustment-name"
	flagFrom       = "from"
	flagTo         = "to"
	flagProject    = "project"
	flagSloName    = "slo-name"
)

func registerOutputFormatFlags(
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

func registerAdjustmentFlag(cmd *cobra.Command, storeIn *string) {
	cmd.Flags().StringVar(storeIn, flagAdjustment, "", "Name of the Adjustment.")
	if err := cmd.MarkFlagRequired(flagAdjustment); err != nil {
		panic(err)
	}
}

func registerProjectFlag(cmd *cobra.Command, storeIn *string) {
	cmd.Flags().StringVarP(storeIn, flagProject, "", "",
		"Name of the project. Required when sloName is defined.")
}

func registerSloNameFlag(cmd *cobra.Command, storeIn *string) {
	cmd.Flags().StringVarP(storeIn, flagSloName, "", "",
		"Name of the SLO. Required when sloName is defined.")
}

type TimeValue struct{ time.Time }

const (
	timeLayout       = time.RFC3339
	timeLayoutString = "RFC3339"
)

func (t *TimeValue) String() string {
	if t.IsZero() {
		return ""
	}
	return t.Format(timeLayout)
}

func (t *TimeValue) Set(s string) (err error) {
	t.Time, err = time.Parse(timeLayout, s)
	return
}

func (t *TimeValue) Type() string {
	return "time"
}

func registerFromFlag(
	cmd *cobra.Command,
	storeIn *TimeValue,
) {
	cmd.Flags().
		Var(storeIn, flagFrom, "Specifies the start date and time for the data range (in UTC).")
	if err := cmd.MarkFlagRequired(flagFrom); err != nil {
		panic(err)
	}
}

func registerToFlag(
	cmd *cobra.Command,
	storeIn *TimeValue,
) {
	cmd.Flags().Var(storeIn, flagTo, "Specifies the end date and time for the data range (in UTC).")
	if err := cmd.MarkFlagRequired(flagTo); err != nil {
		panic(err)
	}
}
