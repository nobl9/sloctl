package flags

import (
	"fmt"
	"time"
)

type TimeValue struct{ time.Time }

const (
	TimeLayout       = time.RFC3339
	TimeLayoutString = "RFC3339"
)

func (t *TimeValue) String() string {
	if t.IsZero() {
		return ""
	}
	return t.Format(TimeLayout)
}

func (t *TimeValue) Set(s string) error {
	var err error
	if t.Time, err = time.Parse(TimeLayout, s); err != nil {
		return fmt.Errorf("date does not match '%s' layout (%s)", TimeLayout, TimeLayoutString)
	}
	return nil
}

func (t *TimeValue) Type() string {
	return "time"
}
