package notifications

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotifier_promptUpdate_WithoutForm(t *testing.T) {
	t.Parallel()
	stdin, err := os.CreateTemp(t.TempDir(), "stdin")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stdin.Close()) })
	stderr, err := os.CreateTemp(t.TempDir(), "stderr")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stderr.Close()) })

	n := notifier{stdin: stdin, stderr: stderr}
	action, err := n.promptUpdate(
		githubRelease{
			TagName: "v1.2.3",
			HTMLURL: "https://github.com/nobl9/sloctl/releases/tag/v1.2.3",
		},
		"",
		"sloctl update",
		false,
	)
	require.NoError(t, err)
	assert.Equal(t, updateActionSkip, action)

	_, err = stderr.Seek(0, io.SeekStart)
	require.NoError(t, err)
	output, err := io.ReadAll(stderr)
	require.NoError(t, err)
	assert.Contains(t, string(output), "New sloctl version v1.2.3 is available!")
	assert.Contains(t, string(output), "https://github.com/nobl9/sloctl/releases/tag/v1.2.3")
	assert.NotContains(t, string(output), "Choose update action")
}

func Test_isUpdateFormSupported(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		goOS                string
		systemName          string
		systemNameErr       error
		expectedIsSupported bool
	}{
		"Linux": {
			goOS:                "linux",
			systemNameErr:       errors.New("uname unavailable"),
			expectedIsSupported: true,
		},
		"Windows MinGW": {
			goOS:                "windows",
			systemName:          "MINGW64_NT-10.0-26100\n",
			expectedIsSupported: true,
		},
		"Windows Cygwin": {
			goOS:                "windows",
			systemName:          "CYGWIN_NT-10.0-26100\n",
			expectedIsSupported: true,
		},
		"Windows MSYS": {
			goOS:                "windows",
			systemName:          "MSYS_NT-10.0-26100\n",
			expectedIsSupported: false,
		},
		"Windows native shell": {
			goOS:                "windows",
			systemName:          "Windows_NT\n",
			expectedIsSupported: false,
		},
		"Windows without uname": {
			goOS:                "windows",
			systemNameErr:       errors.New("uname unavailable"),
			expectedIsSupported: false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			readSystemName := func() (string, error) {
				return tt.systemName, tt.systemNameErr
			}
			assert.Equal(t, tt.expectedIsSupported, isUpdateFormSupported(tt.goOS, readSystemName))
		})
	}
}
