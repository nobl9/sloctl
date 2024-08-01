package internal

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nobl9/nobl9-go/manifest"

	"github.com/nobl9/sloctl/internal/csv"
)

const (
	flagFile   = "file"
	flagDryRun = "dry-run"
)

func notifyDryRunFlag() {
	_, _ = fmt.Fprintln(os.Stderr, "Running in dry run mode, changes will not be applied.")
}

func registerFileFlag(cmd *cobra.Command, required bool, storeIn *[]string) {
	cmd.Flags().StringArrayVarP(storeIn, flagFile, "f", []string{},
		"File path, glob pattern or a URL to the configuration in YAML or JSON format."+
			" This option can be used multiple times.")
	if required {
		_ = cmd.MarkFlagRequired(flagFile)
	}
}

func registerDryRunFlag(cmd *cobra.Command, storeIn *bool) {
	cmd.Flags().BoolVarP(storeIn, flagDryRun, "", false,
		"Submit server-side request without persisting the configured resources.")
}

func registerVerboseFlag(cmd *cobra.Command, storeIn *bool) {
	cmd.Flags().BoolVarP(storeIn, "verbose", "v", false,
		"Display verbose information about configuration")
}

func registerAutoConfirmationFlag(cmd *cobra.Command, storeIn *bool) {
	cmd.Flags().BoolVarP(storeIn, "yes", "y", false,
		"Auto confirm files threshold prompt."+
			" Threshold can be changed or disabled in config.toml or via env variables.")
}

func registerOutputFormatFlags(cmd *cobra.Command, outputFormat, fieldSeparator, recordSeparator *string) {
	cmd.PersistentFlags().StringVarP(outputFormat, "output", "o", "yaml",
		`Output format: one of yaml|json|csv.`)

	cmd.PersistentFlags().StringVarP(fieldSeparator, csv.FieldSeparatorFlag, "",
		csv.DefaultFieldSeparator, "Field Separator for CSV.")

	cmd.PersistentFlags().StringVarP(recordSeparator, csv.RecordSeparatorFlag, "",
		csv.DefaultRecordSeparator, "Record Separator for CSV.")
}

var projectFlagSupportingKinds = map[manifest.Kind]struct{}{
	manifest.KindSLO:          {},
	manifest.KindService:      {},
	manifest.KindAgent:        {},
	manifest.KindAlertPolicy:  {},
	manifest.KindAlertSilence: {},
	manifest.KindAlertMethod:  {},
	manifest.KindDirect:       {},
	manifest.KindDataExport:   {},
	manifest.KindRoleBinding:  {},
	manifest.KindAnnotation:   {},
}

func objectKindSupportsProjectFlag(kind manifest.Kind) bool {
	_, ok := projectFlagSupportingKinds[kind]
	return ok
}

func registerProjectFlag(cmd *cobra.Command, storeIn *string) {
	cmd.PersistentFlags().StringVarP(storeIn, "project", "p", "",
		`List the requested object(s) which belong to the specified Project (name).`)
}

func registerAllProjectsFlag(cmd *cobra.Command, storeIn *bool) {
	cmd.PersistentFlags().BoolVarP(storeIn, "all-projects", "A", false,
		`List the requested object(s) across all projects.`)
}

var labelSupportingKinds = map[manifest.Kind]struct{}{
	manifest.KindProject:     {},
	manifest.KindService:     {},
	manifest.KindSLO:         {},
	manifest.KindAlertPolicy: {},
}

func objectKindSupportsLabelsFlag(kind manifest.Kind) bool {
	_, ok := labelSupportingKinds[kind]
	return ok
}

func registerLabelsFlag(cmd *cobra.Command, storeIn *[]string) {
	cmd.PersistentFlags().StringArrayVarP(storeIn, "label", "l", []string{},
		`Filter resource by label. Example: key=value,key2=value2,key2=value3.`)
}
