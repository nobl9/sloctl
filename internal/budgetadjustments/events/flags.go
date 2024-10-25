package events

import (
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

func MustRegisterFileFlag(cmd *cobra.Command, storeIn *string) {
	cmd.Flags().StringVarP(storeIn, FlagFile, "f", "",
		"File path, glob pattern or a URL to the configuration in YAML or JSON format.")
	if err := cmd.MarkFlagRequired(FlagFile); err != nil {
		panic(err)
	}
}

func MustRegisterAdjustmentFlag(cmd *cobra.Command, storeIn *string) {
	cmd.Flags().StringVar(storeIn, FlagAdjustment, "", "Name of the Adjustment.")
	if err := cmd.MarkFlagRequired(FlagAdjustment); err != nil {
		panic(err)
	}
}

func RegisterProjectFlag(cmd *cobra.Command, storeIn *string) {
	cmd.Flags().StringVarP(storeIn, FlagSloProject, "", "",
		"Name of the project. Required when sloName is defined.")
}

func RegisterSloNameFlag(cmd *cobra.Command, storeIn *string) {
	cmd.Flags().StringVarP(storeIn, FlagSloName, "", "",
		"Name of the SLO. Required when sloName is defined.")
}

func MustRegisterFromFlag(
	cmd *cobra.Command,
	storeIn *flags.TimeValue,
) {
	cmd.Flags().
		Var(storeIn, FlagFrom, "Specifies the start date and time for the data range (in UTC).")
	if err := cmd.MarkFlagRequired(FlagFrom); err != nil {
		panic(err)
	}
}

func MustRegisterToFlag(
	cmd *cobra.Command,
	storeIn *flags.TimeValue,
) {
	cmd.Flags().Var(storeIn, FlagTo, "Specifies the end date and time for the data range (in UTC).")
	if err := cmd.MarkFlagRequired(FlagTo); err != nil {
		panic(err)
	}
}
