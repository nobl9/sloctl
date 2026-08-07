package notifications

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	huh "charm.land/huh/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotifier_promptUpdate_WithoutForm(t *testing.T) {
	t.Setenv("SLOCTL_ACCESSIBLE_MODE", "1")
	const goUpdateCommand = "go install github.com/nobl9/sloctl/cmd/sloctl@latest"
	tests := map[string]struct {
		updateCommand    updateCommand
		showUpdateForm   bool
		expectedGuidance string
	}{
		"detected updater": {
			updateCommand: updateCommand{
				display:    goUpdateCommand,
				executable: "go",
			},
			expectedGuidance: "Update with: " + goUpdateCommand,
		},
		"installation guide": {
			showUpdateForm:   true,
			expectedGuidance: "Installation options: " + installationGuideURL,
		},
		"incomplete updater": {
			updateCommand:    updateCommand{display: goUpdateCommand},
			showUpdateForm:   true,
			expectedGuidance: "Installation options: " + installationGuideURL,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
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
				tt.updateCommand,
				tt.showUpdateForm,
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
			assert.Contains(t, plainOutput, tt.expectedGuidance)
			assert.NotContains(t, plainOutput, "Choose update action")
		})
	}
}

func Test_isGoInstallExecutable_ConfiguredGoEnvironment(t *testing.T) {
	goExecutable, err := exec.LookPath("go")
	require.NoError(t, err)
	t.Setenv("GOENV", "off")

	t.Run("GOBIN", func(t *testing.T) {
		goBin := t.TempDir()
		t.Setenv("GOBIN", goBin)
		t.Setenv("GOPATH", t.TempDir())

		executablePath := writeTestSloctlExecutable(t, goBin)
		assert.True(t, isGoInstallExecutable(executablePath, goExecutable))
	})

	t.Run("GOPATH", func(t *testing.T) {
		goPath := t.TempDir()
		t.Setenv("GOBIN", "")
		t.Setenv("GOPATH", goPath)

		executablePath := writeTestSloctlExecutable(t, filepath.Join(goPath, "bin"))
		assert.True(t, isGoInstallExecutable(executablePath, goExecutable))
	})
}

func Test_isGoInstallExecutable_UsesFirstGOPATHEntry(t *testing.T) {
	goExecutable, err := exec.LookPath("go")
	require.NoError(t, err)
	t.Setenv("GOENV", "off")
	t.Setenv("GOBIN", "")
	firstGoPath := t.TempDir()
	secondGoPath := t.TempDir()
	t.Setenv("GOPATH", strings.Join([]string{firstGoPath, secondGoPath}, string(os.PathListSeparator)))

	secondExecutable := writeTestSloctlExecutable(t, filepath.Join(secondGoPath, "bin"))
	assert.False(t, isGoInstallExecutable(secondExecutable, goExecutable))

	firstExecutable := writeTestSloctlExecutable(t, filepath.Join(firstGoPath, "bin"))
	assert.True(t, isGoInstallExecutable(firstExecutable, goExecutable))
}

func Test_isGoInstallExecutable_UsesFileIdentity(t *testing.T) {
	goExecutable, err := exec.LookPath("go")
	require.NoError(t, err)
	t.Setenv("GOENV", "off")
	t.Setenv("GOPATH", t.TempDir())

	if runtime.GOOS == "windows" {
		goBin := t.TempDir()
		t.Setenv("GOBIN", goBin)
		executablePath := writeTestSloctlExecutable(t, goBin)
		assert.True(t, isGoInstallExecutable(strings.ToUpper(executablePath), goExecutable))
		return
	}

	realGoBin := t.TempDir()
	goBin := filepath.Join(t.TempDir(), "bin")
	require.NoError(t, os.Symlink(realGoBin, goBin))
	t.Setenv("GOBIN", goBin)
	executablePath := writeTestSloctlExecutable(t, realGoBin)
	assert.True(t, isGoInstallExecutable(executablePath, goExecutable))
}

func Test_isUpdateFormSupported(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		goOS                string
		msysEnvironment     string
		isCygwinTerminal    bool
		expectedIsSupported bool
	}{
		"Linux": {
			goOS:                "linux",
			expectedIsSupported: true,
		},
		"Windows MinGW": {
			goOS:                "windows",
			msysEnvironment:     "MINGW64",
			isCygwinTerminal:    true,
			expectedIsSupported: true,
		},
		"Windows Cygwin": {
			goOS:                "windows",
			isCygwinTerminal:    true,
			expectedIsSupported: true,
		},
		"Windows MSYS": {
			goOS:             "windows",
			msysEnvironment:  "MSYS",
			isCygwinTerminal: true,
		},
		"Windows native shell": {
			goOS: "windows",
		},
		"Windows native shell launched from MinGW": {
			goOS:            "windows",
			msysEnvironment: "MINGW64",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(
				t,
				tt.expectedIsSupported,
				isUpdateFormSupported(tt.goOS, tt.msysEnvironment, tt.isCygwinTerminal),
			)
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

func TestNotifier_StateWritesDoNotEraseSkippedRelease(t *testing.T) {
	n := notifier{cachePath: filepath.Join(t.TempDir(), "notifications.json")}

	require.NoError(t, n.saveSkippedRelease("v1.2.3"))
	require.NoError(t, n.saveState(state{LastCheckedAt: time.Now()}))
	require.NoError(t, n.saveState(state{LastCheckedAt: time.Now().Add(time.Minute)}))

	assert.True(t, n.hasSkippedRelease("v1.2.3"))
	assert.False(t, n.hasSkippedRelease("v1.2.4"))
	assert.False(t, n.readState().LastCheckedAt.IsZero())
}

func writeTestSloctlExecutable(t *testing.T, directory string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(directory, 0o700))
	name := "sloctl"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(directory, name)
	require.NoError(t, os.WriteFile(path, []byte("test"), 0o600))
	return path
}
