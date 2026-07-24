package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_executeRootCommand_ClientTimeout(t *testing.T) {
	t.Parallel()

	clientTimeoutError := fmt.Errorf(
		"failed to execute request: %w",
		&url.Error{
			Op:  "Get",
			URL: "https://example.com",
			Err: fmt.Errorf(
				"%w (Client.Timeout exceeded while awaiting headers)",
				context.DeadlineExceeded,
			),
		},
	)
	cmd := &cobra.Command{
		Use:          "sloctl",
		SilenceUsage: true,
		RunE: func(*cobra.Command, []string) error {
			return clientTimeoutError
		},
	}
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	err := executeRootCommand(cmd)

	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Same(t, clientTimeoutError, err)
	const expectedHint = "Hint: The request exceeded sloctl's client-side timeout. " +
		`Increase the active context's "timeout" setting or set SLOCTL_TIMEOUT ` +
		"(for example, SLOCTL_TIMEOUT=2m).\n"
	assert.Equal(t, "Error: "+clientTimeoutError.Error()+"\n"+expectedHint, stderr.String())
}

func Test_executeRootCommand_NonClientTimeoutErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]error{
		"caller deadline": &url.Error{
			Op:  "Get",
			URL: "https://example.com",
			Err: context.DeadlineExceeded,
		},
		"context cancellation": context.Canceled,
		"gateway timeout response": errors.New(
			"server returned 504 Gateway Timeout",
		),
		"unrelated URL timeout": &url.Error{
			Op:  "Get",
			URL: "https://example.com",
			Err: errors.New("dial tcp: i/o timeout"),
		},
		"client timeout marker without deadline": &url.Error{
			Op:  "Get",
			URL: "https://example.com",
			Err: errors.New("Client.Timeout exceeded"),
		},
	}
	for name, commandError := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{
				Use:          "sloctl",
				SilenceUsage: true,
				RunE: func(*cobra.Command, []string) error {
					return commandError
				},
			}
			var stderr bytes.Buffer
			cmd.SetErr(&stderr)

			err := executeRootCommand(cmd)

			require.Error(t, err)
			assert.Equal(t, "Error: "+commandError.Error()+"\n", stderr.String())
		})
	}
}
