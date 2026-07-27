package notifications

import (
	"errors"
	"io"
	"os"
	"testing"

	huh "charm.land/huh/v2"
	"github.com/charmbracelet/x/ansi"
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
	plainOutput := ansi.Strip(string(output))
	assert.Contains(t, plainOutput, "New sloctl version v1.2.3 is available!")
	assert.Contains(t, plainOutput, "https://github.com/nobl9/sloctl/releases/tag/v1.2.3")
	assert.NotContains(t, plainOutput, "Choose update action")
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

func Test_isReleaseNewer(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		currentVersion string
		releaseTag     string
		expected       bool
	}{
		"newer patch": {
			currentVersion: "1.2.3",
			releaseTag:     "v1.2.4",
			expected:       true,
		},
		"stable after prerelease": {
			currentVersion: "v1.2.3-rc.1",
			releaseTag:     "v1.2.3",
			expected:       true,
		},
		"older release": {
			currentVersion: "v1.2.3",
			releaseTag:     "v1.2.2",
		},
		"same version with optional prefix": {
			currentVersion: "1.2.3",
			releaseTag:     "v1.2.3",
		},
		"equivalent build metadata": {
			currentVersion: "v1.2.3+local",
			releaseTag:     "v1.2.3+release",
		},
		"invalid current version": {
			currentVersion: "unknown",
			releaseTag:     "v1.2.3",
		},
		"invalid release tag": {
			currentVersion: "v1.2.3",
			releaseTag:     "latest",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, isReleaseNewer(tt.currentVersion, tt.releaseTag))
		})
	}
}

func Test_isReleaseNotesHeading(t *testing.T) {
	t.Parallel()
	tests := map[string]bool{
		"## 🚀 Features":              true,
		"## 🐞 Bug Fixes":             true,
		"## ⚠️ Breaking Changes":     true,
		"## 💻 Fixed Vulnerabilities": true,
		"## Maintenance":             false,
		"## Prefixes":                false,
		"### Features":               false,
	}
	for heading, expected := range tests {
		t.Run(heading, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, expected, isReleaseNotesHeading(heading))
		})
	}
}

func Test_isInstallScriptExecutable(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		path     string
		expected bool
	}{
		"default Unix install": {
			path:     "/usr/local/bin/sloctl",
			expected: true,
		},
		"default MinGW install": {
			path:     `C:\msys64\usr\local\bin\sloctl.exe`,
			expected: true,
		},
		"manual Unix install": {
			path: "/opt/bin/sloctl",
		},
		"system package install": {
			path: "/usr/bin/sloctl",
		},
		"similar suffix outside MinGW": {
			path: "/opt/usr/local/bin/sloctl",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, isInstallScriptExecutable(tt.path))
		})
	}
}

func Test_notifier_handlePromptError(t *testing.T) {
	tests := map[string]struct {
		promptErr      error
		expectedResult Result
		expectedOutput string
	}{
		"user abort": {
			promptErr:      huh.ErrUserAborted,
			expectedResult: ResultInterrupted,
		},
		"wrapped user abort": {
			promptErr:      errors.Join(errors.New("render form"), huh.ErrUserAborted),
			expectedResult: ResultInterrupted,
		},
		"prompt failure": {
			promptErr:      errors.New("render form"),
			expectedResult: ResultContinue,
			expectedOutput: "failed to read update choice: render form; continuing with the requested command\n",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			reader, writer, err := os.Pipe()
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = reader.Close()
				_ = writer.Close()
			})

			n := notifier{stderr: writer}
			result := n.handlePromptError(tt.promptErr)
			require.NoError(t, writer.Close())
			output, err := io.ReadAll(reader)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedResult, result)
			assert.Equal(t, tt.expectedOutput, string(output))
		})
	}
}
