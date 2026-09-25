package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
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

func TestRootHelpDoesNotRepeatCommandHelpHint(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)

	require.NoError(t, cmd.Help())
	assert.NotContains(t, output.String(), "Run `sloctl <command> --help`")
	assert.Contains(t, output.String(), `Use "sloctl [command] --help"`)
}

func TestRenderHelpMarkdownPreservesCodeLines(t *testing.T) {
	t.Parallel()

	const command = `sloctl completion bash > "$(brew --prefix)/etc/bash_completion.d/sloctl"`
	tests := map[string]struct {
		markdown string
		want     string
	}{
		"tilde fence": {
			markdown: "Before.\n\n~~~text\n" + command + "\n~~~\n\nAfter.",
			want:     "Before.\n\n" + command + "\n\nAfter.",
		},
		"backtick fence": {
			markdown: "Before.\n\n```text\n" + command + "\n```\n\nAfter.",
			want:     "Before.\n\n" + command + "\n\nAfter.",
		},
		"shorter fence inside code": {
			markdown: "~~~~text\n~~~\n" + command + "\n~~~~",
			want:     "~~~\n" + command,
		},
		"unclosed fence": {
			markdown: "~~~text\n" + command,
			want:     command,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rendered, err := renderHelpMarkdown(test.markdown, 24)

			require.NoError(t, err)
			lines := strings.Split(ansi.Strip(rendered), "\n")
			for i := range lines {
				lines[i] = strings.TrimRight(lines[i], " ")
			}
			assert.Equal(t, test.want, strings.TrimSpace(strings.Join(lines, "\n")))
		})
	}
}
