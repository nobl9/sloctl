// Package internal implements user facing commands for sloctl.
package internal

import (
	"fmt"
	"os"
	"runtime"
	"sync"

	v1alphaParser "github.com/nobl9/nobl9-go/manifest/v1alpha/parser"
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/spf13/cobra"

	"github.com/nobl9/sloctl/internal/budgetadjustments"
)

const programName = "sloctl"

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return NewRootCmd().Execute()
}

type globalFlags struct {
	ConfigFile   string
	Context      string
	NoConfigFile bool
}

// NewRootCmd returns the base command when called without any subcommands
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   programName,
		Short: "Create, get and delete SLO definitions from command line easily.",
		Long: `All available commands for execution are listed below.
Use this tool to work with Nobl9 YAML configuration files (including, but not limited to SLOs).
More detailed help is available for each command.`,
		SilenceUsage: true,
	}

	root := RootCmd{}
	rootCmd.PersistentFlags().BoolP("help", "h", false, fmt.Sprintf("Help for %s.", rootCmd.Name()))
	rootCmd.PersistentFlags().StringVar(&root.Flags.ConfigFile, "config", "", "Config file path.")
	rootCmd.PersistentFlags().StringVarP(&root.Flags.Context, "context", "c", "",
		`Overrides the default context for the duration of the selected command.`)
	rootCmd.PersistentFlags().BoolVarP(&root.Flags.NoConfigFile, "no-config-file", "", false,
		`Don't create config.toml, operate only on env variables.`)

	rootCmd.AddCommand(root.NewApplyCmd())
	rootCmd.AddCommand(root.NewDeleteCmd())
	rootCmd.AddCommand(root.NewGetCmd())
	rootCmd.AddCommand(NewVersionCmd())
	rootCmd.AddCommand(root.NewConfigCmd())
	rootCmd.AddCommand(root.NewReplayCmd())
	rootCmd.AddCommand(root.NewAwsIamIds())
	rootCmd.AddCommand(budgetadjustments.NewRootCmd(&root))
	rootCmd.AddCommand(root.NewConvertCmd())
	rootCmd.AddCommand(root.NewMoveCmd())
	rootCmd.AddCommand(root.NewMCPCmd())
	rootCmd.AddCommand(root.NewReviewCmd())
	rootCmd.AddCommand(root.NewRecipesCmd())

	return rootCmd
}

type RootCmd struct {
	Client *sdk.Client
	Flags  globalFlags
	once   sync.Once
}

func (r *RootCmd) GetClient() *sdk.Client {
	r.once.Do(func() {
		if err := r.setupClient(); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	})
	return r.Client
}

const sdkEnvPrefix = "SLOCTL_"

// setupClient reads in config file, ENV variables if set and sets up an API client.
func (r *RootCmd) setupClient() error {
	options := []sdk.ConfigOption{sdk.ConfigOptionEnvPrefix(sdkEnvPrefix)}
	if r.Flags.NoConfigFile {
		options = append(options, sdk.ConfigOptionNoConfigFile())
	}
	if r.Flags.ConfigFile != "" {
		options = append(options, sdk.ConfigOptionFilePath(r.Flags.ConfigFile))
	}
	if r.Flags.Context != "" {
		options = append(options, sdk.ConfigOptionUseContext(r.Flags.Context))
	}
	conf, err := sdk.ReadConfig(options...)
	if err != nil {
		return err
	}
	r.Client, err = sdk.NewClient(conf)
	if err != nil {
		return err
	}
	r.Client.SetUserAgent(getUserAgent())
	// Use generic object representation instead of concrete models for sloctl to be version agnostic.
	v1alphaParser.UseGenericObjects = true
	// Decode JSON numbers into [json.Number] in order to properly handle integers and floats.
	// If we don't use this option, all numbers will be converted to floats, including integers.
	v1alphaParser.UseJSONNumber = true
	return nil
}

func getUserAgent() string {
	return fmt.Sprintf("%s/%s-%s-%s (%s %s %s)",
		programName, getBuildVersion(), BuildGitBranch, BuildGitRevision,
		runtime.GOOS, runtime.GOARCH, runtime.Version(),
	)
}
