package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_executeRootCommand_TimeoutHint(t *testing.T) {
	t.Parallel()

	callerDeadline, cancel := context.WithDeadline(t.Context(), time.Time{})
	t.Cleanup(cancel)

	configuredTimeoutError := fmt.Errorf(
		"failed to execute request: %w",
		&url.Error{
			Op:  "Get",
			URL: "https://example.com",
			Err: context.DeadlineExceeded,
		},
	)
	tests := map[string]struct {
		ctx          context.Context
		commandError error
		expectHint   bool
	}{
		"configured client timeout": {
			ctx:          t.Context(),
			commandError: configuredTimeoutError,
			expectHint:   true,
		},
		"caller deadline": {
			ctx:          callerDeadline,
			commandError: configuredTimeoutError,
		},
		"bare deadline": {
			ctx:          t.Context(),
			commandError: context.DeadlineExceeded,
		},
		"independent joined errors": {
			ctx: t.Context(),
			commandError: errors.Join(
				context.DeadlineExceeded,
				&url.Error{
					Op:  "Get",
					URL: "https://example.com",
					Err: errors.New("connection reset"),
				},
			),
		},
		"unrelated URL error": {
			ctx: t.Context(),
			commandError: &url.Error{
				Op:  "Get",
				URL: "https://example.com",
				Err: errors.New("connection reset"),
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{
				Use:          "sloctl",
				SilenceUsage: true,
				RunE: func(*cobra.Command, []string) error {
					return test.commandError
				},
			}
			cmd.SetContext(test.ctx)
			var stderr bytes.Buffer
			cmd.SetErr(&stderr)

			err := executeRootCommand(cmd)

			require.ErrorIs(t, err, test.commandError)
			if test.expectHint {
				assert.Contains(t, stderr.String(), clientTimeoutConfigurationHint)
			} else {
				assert.NotContains(t, stderr.String(), clientTimeoutConfigurationHint)
			}
		})
	}
}
