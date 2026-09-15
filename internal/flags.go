package internal

import (
	"fmt"
	"os"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/spf13/cobra"
)

const (
	flagFile    = "file"
	flagDryRun  = "dry-run"
	flagVerbose = "verbose"
)

// FlagDescriptionMarkdownAnnotation stores the Markdown description exported for a flag.
const FlagDescriptionMarkdownAnnotation = "sloctl.nobl9.com/description-markdown"

func notifyDryRunFlag() {
	_, _ = fmt.Fprintln(os.Stderr, "Running in dry run mode, changes will not be applied.")
}

func registerFileFlag(cmd *cobra.Command, required bool, storeIn *[]string) {
	usage := "Path, directory, URL, glob pattern, or '-' for YAML or JSON from standard input. " +
		"Repeat this flag to use multiple sources."
	cmd.Flags().StringArrayVarP(storeIn, flagFile, "f", []string{}, usage)
	setFlagDescriptions(cmd, flagFile, usage,
		"Path, directory, URL, glob pattern, or `-` for YAML or JSON from standard input. "+
			"Repeat this flag to use multiple sources.")
	if required {
		_ = cmd.MarkFlagRequired(flagFile)
	}
}

func setFlagDescriptions(cmd *cobra.Command, name, usage, descriptionMarkdown string) {
	flag := cmd.Flag(name)
	flag.Usage = usage
	if flag.Annotations == nil {
		flag.Annotations = make(map[string][]string)
	}
	flag.Annotations[FlagDescriptionMarkdownAnnotation] = []string{descriptionMarkdown}
}

func registerDryRunFlag(cmd *cobra.Command, storeIn *bool) {
	cmd.Flags().BoolVarP(storeIn, flagDryRun, "", false,
		"Send the request without persisting changes.")
}

func registerVerboseFlag(cmd *cobra.Command, storeIn *bool) {
	cmd.Flags().BoolVarP(storeIn, flagVerbose, "v", false,
		"Display verbose information about configuration")
}

func registerAutoConfirmationFlag(cmd *cobra.Command, storeIn *bool) {
	usage := "Skip the file-count confirmation prompt. By default, the prompt appears when a directory or glob " +
		"resolves to more than 23 files. Configure filesPromptEnabled and filesPromptThreshold in the [sloctl] " +
		"section of config.toml, or set SLOCTL_FILES_PROMPT_ENABLED and SLOCTL_FILES_PROMPT_THRESHOLD."
	cmd.Flags().BoolVarP(storeIn, "yes", "y", false, usage)
	setFlagDescriptions(cmd, "yes", usage,
		"Skip the file-count confirmation prompt. By default, the prompt appears when a directory or glob "+
			"resolves to more than 23 files. Configure `filesPromptEnabled` and `filesPromptThreshold` in the "+
			"`[sloctl]` section of `config.toml`, or set `SLOCTL_FILES_PROMPT_ENABLED` and `SLOCTL_FILES_PROMPT_THRESHOLD`.")
}

func registerProjectFlag(cmd *cobra.Command, storeIn *string) {
	cmd.Flags().StringVarP(storeIn, "project", "p", "",
		"Select resources from this project instead of the configured default project.")
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

func registerLabelsFlag(cmd *cobra.Command, storeIn *[]string) {
	cmd.Flags().StringArrayVarP(storeIn, "label", "l", []string{},
		"Filter by label. Repeat the flag or separate labels with commas, "+
			"for example: team=platform,env=prod.")
}

func registerSLOServiceFlag(cmd *cobra.Command, storeIn *[]string) {
	cmd.Flags().StringArrayVarP(storeIn, "service", "s", nil,
		"Filter SLOs by service name. Repeat to select multiple services.")
}

// requireFlagsIfFlagIsSet validates that the provided deps are only set if the "parent" flag is set.
// This one way dependency is not supported natively by cobra and requires custom verification.
func requireFlagsIfFlagIsSet(cmd *cobra.Command, flag string, deps ...string) error {
	if commandFlagChanged(cmd, flag) {
		return nil
	}
	for _, d := range deps {
		if commandFlagChanged(cmd, d) {
			return fmt.Errorf("--%s flag can only be set if --%s flag is also provided", d, flag)
		}
	}
	return nil
}

func commandFlagChanged(cmd *cobra.Command, name string) bool {
	flag := cmd.Flag(name)
	return flag != nil && flag.Changed
}
