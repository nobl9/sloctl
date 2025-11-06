package flags

import (
	"time"

	"github.com/spf13/cobra"
)

const (
	TimeLayout     = time.RFC3339
	TimeLayoutName = "RFC3339"
)

func RegisterTimeVar(cmd *cobra.Command, storeIn *time.Time, name, usage string) {
	cmd.Flags().TimeVar(storeIn, name, time.Time{}, []string{TimeLayout}, usage)
}
