//go:build unit_test

package internal

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobl9/nobl9-go/sdk"
)

//go:embed test_data/definitions_test/project.yaml
var testProject []byte

//go:embed test_data/definitions_test/service.yaml
var testService []byte

func TestReadResourceDefinitions(t *testing.T) {
	for name, test := range map[string]struct {
		Paths               []string
		In                  io.Reader
		Prompt              filesPrompt
		PromptResponse      string
		ExpectedDefinitions int
		PromptDisplayed     bool
		ExpectedError       error
	}{
		"read a single file": {
			Paths:               []string{"test_data/definitions_test/project.yaml"},
			In:                  bytes.NewBuffer(testProject),
			ExpectedDefinitions: 1,
		},
		"read from stdin via ' ' source": {
			Paths:               []string{""},
			In:                  bytes.NewBuffer(testService),
			ExpectedDefinitions: 1,
		},
		"read from stdin via '-' source": {
			Paths:               []string{"-"},
			In:                  bytes.NewBuffer(testService),
			ExpectedDefinitions: 1,
		},
		"read from stdin and a file": {
			Paths:               []string{"test_data/definitions_test/project.yaml", ""},
			In:                  bytes.NewBuffer(testService),
			ExpectedDefinitions: 2,
		},
		"don't display prompt if threshold is not reached": {
			Paths:               []string{"test_data/definitions_test"},
			ExpectedDefinitions: 2,
			Prompt:              filesPrompt{Enabled: true, Threshold: 10},
			PromptDisplayed:     false,
		},
		"don't display prompt if threshold is exceeded, but prompt is disabled": {
			Paths:               []string{"test_data/definitions_test"},
			ExpectedDefinitions: 2,
			Prompt:              filesPrompt{Enabled: false, Threshold: 1},
			PromptDisplayed:     false,
		},
		"don't display prompt if threshold is exceeded, but auto confirm is set": {
			Paths:               []string{"test_data/definitions_test"},
			ExpectedDefinitions: 2,
			Prompt:              filesPrompt{Enabled: true, AutoConfirm: true, Threshold: 1},
			PromptDisplayed:     false,
		},
		"don't display prompt if multiple definitions.SourceTypeFile sources are provided exceeding threshold": {
			Paths: []string{
				"test_data/definitions_test/project.yaml",
				"test_data/definitions_test/service.yaml",
			},
			ExpectedDefinitions: 2,
			Prompt:              filesPrompt{Enabled: true, Threshold: 1},
			PromptDisplayed:     false,
		},
		"display prompt when threshold is exceeded (variant 1)": {
			Paths:               []string{"test_data/definitions_test"},
			ExpectedDefinitions: 2,
			Prompt:              filesPrompt{Enabled: true, Threshold: 1},
			PromptResponse:      "y\n",
			PromptDisplayed:     true,
		},
		"display prompt when threshold is exceeded (variant 2)": {
			Paths:               []string{"test_data/definitions_test"},
			ExpectedDefinitions: 2,
			Prompt:              filesPrompt{Enabled: true, Threshold: 1},
			PromptResponse:      "yes\n",
			PromptDisplayed:     true,
		},
		"display prompt when threshold is exceeded (variant 3)": {
			Paths:               []string{"test_data/definitions_test"},
			ExpectedDefinitions: 2,
			Prompt:              filesPrompt{Enabled: true, Threshold: 1},
			PromptResponse:      "Y\n",
			PromptDisplayed:     true,
		},
		"display prompt when threshold is exceeded (variant 4)": {
			Paths:               []string{"test_data/definitions_test"},
			ExpectedDefinitions: 2,
			Prompt:              filesPrompt{Enabled: true, Threshold: 1},
			PromptResponse:      "YES\n",
			PromptDisplayed:     true,
		},
		"display prompt when threshold is exceeded for definitions.SourceTypeGlobPattern": {
			Paths:               []string{"test_data/definitions_test/**"},
			ExpectedDefinitions: 2,
			Prompt:              filesPrompt{Enabled: true, Threshold: 1},
			PromptResponse:      "yes\n",
			PromptDisplayed:     true,
		},
		"abort process when prompt is not confirmed": {
			Paths:               []string{"test_data/definitions_test"},
			ExpectedDefinitions: 2,
			Prompt:              filesPrompt{Enabled: true, Threshold: 1},
			PromptResponse:      "no\n",
			PromptDisplayed:     true,
			ExpectedError:       errOperationAborted,
		},
		"abort process when empty new line is provided": {
			Paths:               []string{"test_data/definitions_test"},
			ExpectedDefinitions: 2,
			Prompt:              filesPrompt{Enabled: true, Threshold: 1},
			PromptResponse:      "\n",
			PromptDisplayed:     true,
			ExpectedError:       errOperationAborted,
		},
		"return error if project override does not match the object's project": {
			Paths:               []string{"test_data/definitions_test"},
			ExpectedDefinitions: 2,
			Prompt:              filesPrompt{Enabled: true, Threshold: 1},
			PromptResponse:      "\n",
			PromptDisplayed:     true,
			ExpectedError:       errOperationAborted,
		},
	} {
		t.Run(name, func(t *testing.T) {
			cmd := new(cobra.Command)
			out := new(bytes.Buffer)
			cmd.SetOut(out)
			cmd.SetIn(test.In)

			reader := &mockReader{Reader: strings.NewReader(test.PromptResponse)}
			test.Prompt.ReadFrom = reader

			d, err := readObjectsDefinitions(cmd.Context(), &sdk.Config{}, cmd, test.Paths, test.Prompt, false)

			if test.PromptDisplayed {
				assert.Equal(t, test.PromptDisplayed, reader.WasRead)
				// If any of the test files would contain more than one definition, this would have to corrected.
				assert.Equal(t,
					fmt.Sprintf(filesPromptPattern, test.Prompt.Threshold, test.ExpectedDefinitions),
					out.String())
			}
			if test.ExpectedError == nil {
				require.NoError(t, err)
				assert.Len(t, d, test.ExpectedDefinitions)
			} else {
				require.Error(t, err)
				assert.Equal(t, test.ExpectedError, err)
			}
		})
	}
}

type mockReader struct {
	WasRead bool
	Reader  *strings.Reader
}

func (m *mockReader) Read(p []byte) (n int, err error) {
	m.WasRead = true
	return m.Reader.Read(p)
}
