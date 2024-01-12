package internal

import (
	"fmt"

	"github.com/spf13/cobra"
)

const versionCmdName = "version"

// NewVersionCmd returns cobra command version with all flags for it.
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   versionCmdName,
		Short: "Print the sloctl version",
		Run: func(*cobra.Command, []string) {
			fmt.Println(getUserAgent())
		},
	}
}

// Set during build time.
var (
	BuildGitRevision string
	BuildGitBranch   string
	BuildVersion     string
)
