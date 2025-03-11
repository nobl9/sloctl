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
			if bi, ok := debug.ReadBuildInfo(); ok {
				fmt.Printf("Path: %s\n", bi.Path)
				fmt.Printf("GoVersion: %s\n", bi.GoVersion)
				fmt.Printf("Version: %s\n", bi.Main.Version)
				fmt.Printf("Version: %s\n", bi.Main.Sum)
				for i, setting := range bi.Settings {
					fmt.Printf("Setting [%d]: %s = %s\n", i, setting.Key, setting.Value)
				}
			} else {
				fmt.Println("No build info available")
			}
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
