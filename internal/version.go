package internal

import (
	_ "embed"
	"fmt"
	"runtime/debug"
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

// Set during build time.
var (
	BuildGitRevision string
	BuildGitBranch   string
	BuildVersion     string
)

func getBuildVersion() string {
	version := BuildVersion
	if version == "" {
		version = getRuntimeVersion()
	}
	return strings.TrimSpace(version)
}

func getRuntimeVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "(devel)" {
		return "0.0.0"
	}
	return strings.TrimPrefix(info.Main.Version, "v")
}
