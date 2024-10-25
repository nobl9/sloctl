package flags

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const FlagDryRun = "dry-run"

func NotifyDryRunFlag() {
	_, _ = fmt.Fprintln(os.Stderr, "Running in dry run mode, changes will not be applied.")
}

func RegisterDryRunFlag(cmd *cobra.Command, storeIn *bool) {
	cmd.Flags().BoolVarP(storeIn, FlagDryRun, "", false,
		"Submit server-side request without persisting the configured resources.")
}
