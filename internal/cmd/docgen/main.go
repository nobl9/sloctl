package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nobl9/sloctl/internal"
)

const defaultOutputPath = "docs/sloctl-command-reference.json"

type commandOptions struct {
	outputPath string
}

func main() {
	if err := newCommand().Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newCommand() *cobra.Command {
	options := commandOptions{}
	cmd := &cobra.Command{
		Use:           "docgen",
		Short:         "Generate structured sloctl command-reference data.",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(*cobra.Command, []string) error {
			return writeCommandReference(internal.NewRootCmd(), options.outputPath)
		},
	}
	cmd.Flags().StringVar(&options.outputPath, "output", defaultOutputPath, "Output JSON file path.")
	return cmd
}
