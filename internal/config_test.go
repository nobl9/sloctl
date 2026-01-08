package internal

import (
	"bytes"
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed test_data/config.toml
var expectedConfig string

func TestConfigAddContext(t *testing.T) {
	t.Skip("the test is currently not working due to how teatest works with huh")

	tmpDir := t.TempDir()
	configFilePath := filepath.Join(tmpDir, "config.toml")
	config := sdk.FileConfig{
		ContextlessConfig: sdk.ContextlessConfig{DefaultContext: "default"},
		Contexts: map[string]sdk.ContextConfig{
			"default": {
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
		},
	}
	err := config.Save(configFilePath)
	require.NoError(t, err)

	runHuhFormFunc = func(_ context.Context, form *huh.Form) error {
		tm := teatest.NewTestModel(t, form, teatest.WithInitialTermSize(800, 500))

		waitForTeaText(t, tm, "Provide context name")
		// Context name:
		tm.Type("new-context")
		tm.Send(teaKeyEnter())
		// Client ID:
		tm.Type("client-id")
		tm.Send(teaKeyEnter())
		// Client Secret:
		tm.Type("super-secret")
		tm.Send(teaKeyEnter())
		// Platform instance choice (default):
		tm.Send(teaKeyEnter())

		waitForTeaText(t, tm, "Provide default project")
		// Project choice (default):
		tm.Send(teaKeyEnter())
		// Set as default context? (No - default):
		tm.Send(teaKeyEnter())

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
		return nil
	}

	cmd := new(ConfigCmd)
	err = cmd.loadFileConfig(configFilePath)
	require.NoError(t, err)

	err = cmd.AddContextCommand().RunE(&cobra.Command{}, nil)
	require.NoError(t, err)

	data, err := os.ReadFile(configFilePath)
	require.NoError(t, err)

	assert.Equal(t, expectedConfig, string(data))
}

func TestMaskField(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{in: "", out: ""},
		{in: "asd", out: "***"},
		{in: "foo-ba", out: "***"},
		{in: "foo-bar", out: "fo***ar"},
		{in: "super-secret-long-string", out: "su***ng"},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			out := maskField(test.in)
			assert.Equal(t, test.out, out)
		})
	}
}

func teaKeyEnter() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEnter}
}

func waitForTeaText(t *testing.T, tm *teatest.TestModel, text string) {
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte(text))
	}, teatest.WithCheckInterval(time.Millisecond*100), teatest.WithDuration(time.Second*3))
}
