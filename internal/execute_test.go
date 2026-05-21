package internal

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestExecuteRunsNotifierAfterSuccessfulCommand(t *testing.T) {
	events := make([]string, 0, 2)
	cmd := &cobra.Command{
		Use: "test",
		Run: func(*cobra.Command, []string) {
			events = append(events, "command")
		},
	}

	exitCode := execute(cmd, func() {
		events = append(events, "notification")
	})

	assert.Zero(t, exitCode)
	assert.Equal(t, []string{"command", "notification"}, events)
}

func TestExecuteSkipsNotifierAfterFailedCommand(t *testing.T) {
	notificationCalled := false
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(*cobra.Command, []string) error {
			return errors.New("command failed")
		},
	}

	exitCode := execute(cmd, func() {
		notificationCalled = true
	})

	assert.Equal(t, 1, exitCode)
	assert.False(t, notificationCalled)
}
