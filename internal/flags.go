package internal

import (
	"fmt"
	"os"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/spf13/cobra"
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
	// While Alert itself is not Project scoped per-se,
	// it does support Project filtering.
	manifest.KindAlert: {},
}

func objectKindSupportsProjectFlag(kind manifest.Kind) bool {
	_, ok := projectFlagSupportingKinds[kind]
	return ok
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

var sloSupportingKinds = map[manifest.Kind]struct{}{
	manifest.KindBudgetAdjustment: {},
}

func objectKindSupportsProjectSloFlag(kind manifest.Kind) bool {
	_, ok := sloSupportingKinds[kind]
	return ok
}

func registerProjectSloFlag(cmd *cobra.Command, storeSloIn, storeProjectIn *string) {
	cmd.Flags().StringVarP(storeSloIn, "slo", "", "",
		`Filter resource by SLO name. Example: my-sample-slo-name`)
	cmd.Flags().StringVarP(storeProjectIn, "project", "p", "",
		`Filter resource by SLO Project name. Example: my-sample-project-name`)
	cmd.MarkFlagsRequiredTogether("slo", "project")
}

func registerLabelsFlag(cmd *cobra.Command, storeIn *[]string) {
	cmd.Flags().StringArrayVarP(storeIn, "label", "l", []string{},
		`Filter resource by label. Example: key=value,key2=value2,key2=value3.`)
}
