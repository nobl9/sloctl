// Package notifications shows release notifications and prompts for updates on supported terminals.
package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	huh "charm.land/huh/v2"
	"github.com/mattn/go-isatty"
	"golang.org/x/mod/semver"
)

// latestReleaseURL can be replaced at link time for deterministic notification tests.
var latestReleaseURL = "https://api.github.com/repos/nobl9/sloctl/releases/latest"

const (
	optOutEnv       = "SLOCTL_NO_NOTIFICATIONS"
	ciEnv           = "CI"
	checkInterval   = 24 * time.Hour
	checkTimeout    = 750 * time.Millisecond
	maxResponseSize = 1 << 20
)

// Result describes what the caller should do after running the notification flow.
type Result int

const (
	// ResultContinue allows the requested sloctl command to run.
	ResultContinue Result = iota
	// ResultExitSuccess exits after a successful update.
	ResultExitSuccess
	// ResultExitFailure exits after a failed update.
	ResultExitFailure
	// ResultInterrupted exits after the user cancels the update prompt.
	ResultInterrupted
)

// Notify checks for a newer release in eligible interactive sessions.
// It may display an update prompt when the terminal supports one.
func Notify(currentVersion string) Result {
	return newNotifier(currentVersion).notify()
}

type notifier struct {
	currentVersion string
	stdin          *os.File
	stdout         *os.File
	stderr         *os.File
	releaseURL     string
	cachePath      string
}

type state struct {
	LastCheckedAt       time.Time `json:"lastCheckedAt"`
	LastShownReleaseTag string    `json:"lastShownReleaseTag"`
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
}

func newNotifier(currentVersion string) notifier {
	return notifier{
		currentVersion: strings.TrimSpace(currentVersion),
		stdin:          os.Stdin,
		stdout:         os.Stdout,
		stderr:         os.Stderr,
		releaseURL:     latestReleaseURL,
		cachePath:      defaultCachePath(),
	}
}

func (n notifier) notify() Result {
	if !n.canNotify() {
		return ResultContinue
	}
	currentState := n.readState()
	now := time.Now()
	if !currentState.LastCheckedAt.IsZero() && now.Sub(currentState.LastCheckedAt) < checkInterval {
		return ResultContinue
	}

	release, err := n.fetchLatestReleaseWithTimeout()
	currentState.LastCheckedAt = now
	if err != nil {
		n.saveState(currentState)
		return ResultContinue
	}
	if !isReleaseNewer(n.currentVersion, release.TagName) {
		n.saveState(currentState)
		return ResultContinue
	}
	if currentState.LastShownReleaseTag == release.TagName {
		n.saveState(currentState)
		return ResultContinue
	}

	releaseNotesMarkdown := extractReleaseNotesMarkdown(release.Body)
	updateCommand := detectUpdateCommand()
	action, err := n.promptUpdate(
		release,
		releaseNotesMarkdown,
		updateCommand.display,
		isUpdateFormSupported(runtime.GOOS, systemName),
	)
	if err != nil {
		result := n.handlePromptError(err)
		if result != ResultInterrupted {
			n.saveState(currentState)
		}
		return result
	}
	if action == updateActionSkipUntilNextVersion {
		currentState.LastShownReleaseTag = release.TagName
	}
	n.saveState(currentState)
	if action != updateActionRunUpgrade {
		return ResultContinue
	}
	if !updateCommand.available() {
		return ResultContinue
	}
	if err := n.runCommand(updateCommand); err != nil {
		_, _ = fmt.Fprintf(n.stderr, "failed to update sloctl: %v\n", err)
		return ResultExitFailure
	}
	return ResultExitSuccess
}

func (n notifier) fetchLatestReleaseWithTimeout() (githubRelease, error) {
	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()
	return n.fetchLatestRelease(ctx)
}

func (n notifier) canNotify() bool {
	return isTerminal(n.stdin) &&
		isTerminal(n.stderr) &&
		n.cachePath != "" &&
		os.Getenv(ciEnv) == "" &&
		os.Getenv(optOutEnv) == "" &&
		!isDevelopmentVersion(n.currentVersion)
}

func isTerminal(file *os.File) bool {
	fd := file.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

func (n notifier) handlePromptError(err error) Result {
	if errors.Is(err, huh.ErrUserAborted) {
		return ResultInterrupted
	}
	_, _ = fmt.Fprintf(
		n.stderr,
		"failed to read update choice: %v; continuing with the requested command\n",
		err,
	)
	return ResultContinue
}

func isUpdateFormSupported(goOS string, readSystemName func() (string, error)) bool {
	if goOS != "windows" {
		return true
	}
	systemName, err := readSystemName()
	if err != nil {
		return false
	}
	systemName = strings.ToLower(strings.TrimSpace(systemName))
	return strings.HasPrefix(systemName, "mingw") || strings.HasPrefix(systemName, "cygwin")
}

func systemName() (string, error) {
	output, err := exec.Command("uname").Output()
	return string(output), err
}

func (n notifier) fetchLatestRelease(ctx context.Context) (githubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, n.releaseURL, nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "sloctl")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return githubRelease{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return githubRelease{}, fmt.Errorf("github release request returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err = json.NewDecoder(io.LimitReader(resp.Body, maxResponseSize)).Decode(&release); err != nil {
		return githubRelease{}, err
	}
	if release.TagName == "" || release.HTMLURL == "" {
		return githubRelease{}, fmt.Errorf("github release response is missing required fields")
	}
	return release, nil
}

func (n notifier) readState() state {
	data, err := os.ReadFile(n.cachePath)
	if err != nil {
		return state{}
	}
	var currentState state
	if err = json.Unmarshal(data, &currentState); err != nil {
		return state{}
	}
	return currentState
}

func (n notifier) saveState(currentState state) {
	if err := os.MkdirAll(filepath.Dir(n.cachePath), 0o700); err != nil {
		return
	}
	data, err := json.MarshalIndent(currentState, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(n.cachePath, data, 0o600)
}

func defaultCachePath() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(cacheDir, "nobl9", "sloctl", "notifications.json")
}

func isDevelopmentVersion(version string) bool {
	version = strings.TrimSpace(version)
	return version == "" ||
		version == "0.0.0" ||
		strings.HasSuffix(version, "-test") ||
		strings.Contains(version, "devel")
}

func isReleaseNewer(currentVersion, releaseTag string) bool {
	currentVersion = semanticVersion(currentVersion)
	releaseTag = semanticVersion(releaseTag)
	return semver.IsValid(currentVersion) &&
		semver.IsValid(releaseTag) &&
		semver.Compare(releaseTag, currentVersion) > 0
}

func semanticVersion(version string) string {
	version = strings.TrimSpace(version)
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}
