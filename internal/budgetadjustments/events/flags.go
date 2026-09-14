package events

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/nobl9/sloctl/internal/flags"
)

const (
	FlagFile       = "file"
	FlagAdjustment = "adjustment-name"
	FlagFrom       = "from"
	FlagTo         = "to"
	FlagSloProject = "slo-project"
	FlagSloName    = "slo-name"
)

func mustRegisterFileFlag(cmd *cobra.Command, storeIn *string) {
	cmd.Flags().StringVarP(storeIn, FlagFile, "f", "",
		"Path to a YAML event definition file, or - to read from standard input.")
	if err := cmd.MarkFlagRequired(FlagFile); err != nil {
		panic(err)
	}
}

func mustRegisterAdjustmentFlag(cmd *cobra.Command, storeIn *string) {
	cmd.Flags().StringVar(storeIn, FlagAdjustment, "", "Name of the budget adjustment.")
	if err := cmd.MarkFlagRequired(FlagAdjustment); err != nil {
		panic(err)
	}
}

func registerProjectFlag(cmd *cobra.Command, storeIn *string) {
	cmd.Flags().StringVarP(storeIn, FlagSloProject, "", "",
		"Project of the SLO to filter by. Must be used with --slo-name.")
}

func registerSloNameFlag(cmd *cobra.Command, storeIn *string) {
	cmd.Flags().StringVarP(storeIn, FlagSloName, "", "",
		"SLO name to filter by. Must be used with --slo-project.")
}

func mustRegisterFromFlag(cmd *cobra.Command, storeIn *time.Time) {
	flags.RegisterTimeVar(
		cmd,
		storeIn,
		FlagFrom,
		"Start of the query range in RFC3339 format.",
	)
	if err := cmd.MarkFlagRequired(FlagFrom); err != nil {
		panic(err)
	}
}

func mustRegisterToFlag(cmd *cobra.Command, storeIn *time.Time) {
	flags.RegisterTimeVar(
		cmd,
		storeIn,
		FlagTo,
		"End of the query range in RFC3339 format.",
	)
	if err := cmd.MarkFlagRequired(FlagTo); err != nil {
		panic(err)
	}
}
