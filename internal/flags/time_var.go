package flags

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

const (
	TimeLayout     = time.RFC3339
	TimeLayoutName = "RFC3339"
)

// TimeVar is a custom [pflag.Value] implementation for [time.Time]
// that provides a better formatted error message.
type TimeVar struct {
	value *time.Time
}

func (t *TimeVar) String() string {
	if t.value == nil || t.value.IsZero() {
		return ""
	}
	return t.value.Format(TimeLayout)
}

func (t *TimeVar) Set(s string) error {
	parsed, err := time.Parse(TimeLayout, s)
	if err != nil {
		return fmt.Errorf(
			"invalid time format, expected RFC3339 layout " +
				"(e.g. '2006-01-02T15:04:05Z' or '2006-01-02T08:04:05-07:00')",
		)
	}
	*t.value = parsed
	return nil
}

func (t *TimeVar) Type() string {
	return "time"
}

func RegisterTimeVar(cmd *cobra.Command, storeIn *time.Time, name, usage string) {
	cmd.Flags().Var(&TimeVar{value: storeIn}, name, usage)
}
