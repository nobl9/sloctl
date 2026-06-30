// Package notifications checks for new sloctl releases and prompts for updates.
package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
)

const (
	defaultReleaseURL = "https://api.github.com/repos/nobl9/sloctl/releases/latest"
	releaseURLEnv     = "SLOCTL_NOTIFICATIONS_RELEASE_URL"
	optOutEnv         = "SLOCTL_NO_NOTIFICATIONS"
	ciEnv             = "CI"
	checkInterval     = 24 * time.Hour
	checkTimeout      = 750 * time.Millisecond
	maxResponseSize   = 1 << 20
)

// Result describes what the caller should do after checking notifications.
type Result int

const (
	ResultContinue Result = iota
	ResultExitSuccess
	ResultExitFailure
)

// Notify checks and displays an interactive update prompt when configured to do so.
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
	releaseURL := strings.TrimSpace(os.Getenv(releaseURLEnv))
	if releaseURL == "" {
		releaseURL = defaultReleaseURL
	}
	return notifier{
		currentVersion: strings.TrimSpace(currentVersion),
		stdin:          os.Stdin,
		stdout:         os.Stdout,
		stderr:         os.Stderr,
		releaseURL:     releaseURL,
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
	if isCurrentRelease(n.currentVersion, release.TagName) {
		n.saveState(currentState)
		return ResultContinue
	}
	if currentState.LastShownReleaseTag == release.TagName {
		n.saveState(currentState)
		return ResultContinue
	}

	releaseNotesMarkdown := extractReleaseNotesMarkdown(release.Body)
	updateCommand := n.updateCommand()
	action, err := n.promptUpdate(release, releaseNotesMarkdown, updateCommand)
	if err != nil {
		n.saveState(currentState)
		return ResultContinue
	}
	if action == updateActionSkipUntilNextVersion {
		currentState.LastShownReleaseTag = release.TagName
	}
	n.saveState(currentState)
	if action != updateActionRunUpgrade {
		return ResultContinue
	}
	if updateCommand == "" {
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
	return isatty.IsTerminal(n.stdin.Fd()) &&
		isatty.IsTerminal(n.stderr.Fd()) &&
		n.cachePath != "" &&
		os.Getenv(ciEnv) == "" &&
		os.Getenv(optOutEnv) == "" &&
		!isDevelopmentVersion(n.currentVersion)
}

func (n notifier) runCommand(command string) error {
	//nolint:gosec // The command is assembled from fixed sloctl update templates.
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdin = n.stdin
	cmd.Stdout = n.stdout
	cmd.Stderr = n.stderr
	return cmd.Run()
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

func isCurrentRelease(currentVersion, releaseTag string) bool {
	return normalizeVersion(currentVersion) == normalizeVersion(releaseTag)
}

func normalizeVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}
