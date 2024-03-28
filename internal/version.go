package internal

import (
	_ "embed"
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
// BuildVersion defaults to VERSION file contents.
// This is neccessary since we don't have control over build flags when installed through `go install`.
var (
	BuildGitRevision string
	BuildGitBranch   string
	//go:embed VERSION
	BuildVersion string
)
