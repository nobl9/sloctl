package sloctl

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nobl9/n9/internal/sloctl/csv"
)

const (
	flagFile   = "file"
	flagDryRun = "dry-run"
)

func NotifyDryRunFlag() {
	_, _ = fmt.Fprintln(os.Stderr, "Running in dry run mode, changes will not be applied.")
}

func RegisterFileFlag(cmd *cobra.Command, required bool, storeIn *[]string) {
	cmd.Flags().StringArrayVarP(storeIn, flagFile, "f", []string{},
		"File path, glob pattern or a URL to the configuration in YAML or JSON format."+
			" This option can be used multiple times.")
	if required {
		_ = cmd.MarkFlagRequired(flagFile)
	}
}

func RegisterDryRunFlag(cmd *cobra.Command, storeIn *bool) {
	cmd.Flags().BoolVarP(storeIn, flagDryRun, "", false,
		"Submit server-side request without persisting the configured resources.")
}

func RegisterVerboseFlag(cmd *cobra.Command, storeIn *bool) {
	cmd.Flags().BoolVarP(storeIn, "verbose", "v", false,
		"Display verbose information about configuration")
}

func RegisterAutoConfirmationFlag(cmd *cobra.Command, storeIn *bool) {
	cmd.Flags().BoolVarP(storeIn, "yes", "y", false,
		"Auto confirm files threshold prompt."+
			" Threshold can be changed or disabled in config.toml or via env variables.")
}

func RegisterOutputFormatFlags(cmd *cobra.Command, outputFormat, fieldSeparator, recordSeparator *string) {
	cmd.PersistentFlags().StringVarP(outputFormat, "output", "o", "yaml",
		`Output format: one of yaml|json|csv.`)

	cmd.PersistentFlags().StringVarP(fieldSeparator, csv.FieldSeparatorFlag, "",
		csv.DefaultFieldSeparator, "Field Separator for CSV.")

	cmd.PersistentFlags().StringVarP(recordSeparator, csv.RecordSeparatorFlag, "",
		csv.DefaultRecordSeparator, "Record Separator for CSV.")
}
