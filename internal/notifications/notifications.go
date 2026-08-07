// Package notifications displays release notices in eligible interactive sessions.
// It offers update actions for recognized Homebrew and Go installations.
package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
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

// Notify checks for a newer release in eligible interactive sessions and reports
// whether the caller should continue or exit. Checks are best-effort and cached;
// recognized Homebrew and Go installations may offer an interactive update action.
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
	LastCheckedAt time.Time `json:"lastCheckedAt"`
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
	lastCheckAge := now.Sub(currentState.LastCheckedAt)
	if !currentState.LastCheckedAt.IsZero() && lastCheckAge >= 0 && lastCheckAge < checkInterval {
		return ResultContinue
	}

	release, err := n.fetchLatestReleaseWithTimeout()
	currentState.LastCheckedAt = now
	if err != nil {
		_ = n.saveState(currentState)
		return ResultContinue
	}
	if !isReleaseNewer(n.currentVersion, release.TagName) {
		_ = n.saveState(currentState)
		return ResultContinue
	}
	if n.hasSkippedRelease(release.TagName) {
		_ = n.saveState(currentState)
		return ResultContinue
	}

	releaseNotesMarkdown := extractReleaseNotesMarkdown(release.Body)
	updateCommand := detectUpdateCommand()
	action, err := n.promptUpdate(
		release,
		releaseNotesMarkdown,
		updateCommand,
		isUpdateFormSupported(
			runtime.GOOS,
			os.Getenv("MSYSTEM"),
			isatty.IsCygwinTerminal(n.stderr.Fd()),
		),
	)
	if err != nil {
		result := n.handlePromptError(err)
		if result != ResultInterrupted {
			_ = n.saveState(currentState)
		}
		return result
	}
	_ = n.saveState(currentState)
	if action == updateActionSkipUntilNextVersion {
		if err := n.saveSkippedRelease(release.TagName); err != nil {
			_, _ = fmt.Fprintf(
				n.stderr,
				"failed to save update preference; the notification may be shown again: %v\n",
				err,
			)
		}
	}
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

func isUpdateFormSupported(goOS, msysEnvironment string, isCygwinTerminal bool) bool {
	if goOS != "windows" {
		return true
	}
	if !isCygwinTerminal {
		return false
	}
	return !strings.EqualFold(strings.TrimSpace(msysEnvironment), "MSYS")
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

func (n notifier) saveState(currentState state) error {
	if err := os.MkdirAll(filepath.Dir(n.cachePath), 0o700); err != nil {
		return fmt.Errorf("create notification cache directory: %w", err)
	}
	data, err := json.MarshalIndent(currentState, "", "  ")
	if err != nil {
		return fmt.Errorf("encode notification cache: %w", err)
	}
	if err := writeFileAtomically(n.cachePath, data); err != nil {
		return fmt.Errorf("write notification cache: %w", err)
	}
	return nil
}

func (n notifier) hasSkippedRelease(releaseTag string) bool {
	_, err := os.Stat(n.skippedReleasePath(releaseTag))
	return err == nil
}

func (n notifier) saveSkippedRelease(releaseTag string) error {
	if err := os.MkdirAll(filepath.Dir(n.cachePath), 0o700); err != nil {
		return fmt.Errorf("create notification cache directory: %w", err)
	}
	if err := os.WriteFile(n.skippedReleasePath(releaseTag), nil, 0o600); err != nil {
		return fmt.Errorf("write update preference: %w", err)
	}
	return nil
}

func (n notifier) skippedReleasePath(releaseTag string) string {
	return filepath.Join(filepath.Dir(n.cachePath), "skip-"+releaseTag)
}

func writeFileAtomically(path string, data []byte) error {
	temporaryFile, err := os.CreateTemp(filepath.Dir(path), ".notifications-*")
	if err != nil {
		return err
	}
	temporaryPath := temporaryFile.Name()
	defer func() { _ = os.Remove(temporaryPath) }()

	if _, err = temporaryFile.Write(data); err != nil {
		_ = temporaryFile.Close()
		return err
	}
	if err = temporaryFile.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
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
