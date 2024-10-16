package flags

import (
	"fmt"
	"time"
)

// type TimeValue struct{ time.Time }
type TimeValue struct{ time.Time }

const (
	timeFormat = time.RFC3339
)

func (t *TimeValue) String() string {
	if t.IsZero() {
		return ""
	}
	return t.Format(timeFormat)
}

func (t *TimeValue) Set(s string) error {
	var err error
	if t.Time, err = time.Parse(timeFormat, s); err != nil {
		return fmt.Errorf("to does not match %s format", timeFormat)
	}
	return nil
}

func (t *TimeValue) Type() string {
	return "time"
}