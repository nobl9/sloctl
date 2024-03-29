package internal

import (
	_ "embed"
	"fmt"
	"strings"

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

// BuildVersion defaults to VERSION file contents.
// This is necessary since we don't have control over build flags when installed through `go install`.
//
//go:embed VERSION
var embeddedBuildVersion string

// Set during build time.
var (
	BuildGitRevision string
	BuildGitBranch   string
	BuildVersion     string
)

func getBuildVersion() string {
	version := BuildVersion
	if version == "" {
		version = embeddedBuildVersion
	}
	return strings.TrimSpace(version)
}
