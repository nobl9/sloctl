// Package notifications handles unobtrusive CLI notifications.
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
	"regexp"
	"strconv"
	"strings"
	"time"

	huh "charm.land/huh/v2"
	"github.com/charmbracelet/glamour"
	"github.com/mattn/go-isatty"
	"golang.org/x/term"

	"github.com/nobl9/sloctl/internal/huhform"
	"github.com/nobl9/sloctl/internal/style"
)

const (
	defaultReleaseURL = "https://api.github.com/repos/nobl9/sloctl/releases/latest"
	releaseURLEnv     = "SLOCTL_NOTIFICATIONS_RELEASE_URL"
	optOutEnv         = "SLOCTL_NO_NOTIFICATIONS"
	ciEnv             = "CI"
	checkInterval     = 24 * time.Hour
	checkTimeout      = 750 * time.Millisecond
	maxResponseSize   = 1 << 20
	defaultBoxWidth   = 92
	minBoxWidth       = 48
)

var (
	releaseMetadataPattern = regexp.MustCompile(`\s+\(#\d+\)(?:\s+@\S+)?$`)
	releasePRPattern       = regexp.MustCompile(`#(\d+)`)
)

// Config defines notification runtime dependencies.
type Config struct {
	CurrentVersion string
	Stdin          io.Reader
	Stdout         io.Writer
	Stderr         io.Writer

	ReleaseURL string
	HTTPClient *http.Client
	CachePath  string

	Now    func() time.Time
	Getenv func(string) string
	IsTTY  func() bool
	Lookup func(string) (string, error)

	TerminalWidth  func() int
	RenderMarkdown func(string, int) (string, error)
	Executable     func() (string, error)
	EvalSymlinks   func(string) (string, error)
	RunCommand     func(context.Context, string) error
}

// Result describes what the caller should do after checking notifications.
type Result int

const (
	ResultContinue Result = iota
	ResultExitSuccess
	ResultExitFailure
)

// Notify checks and displays an interactive update prompt when configured to do so.
func Notify(config Config) Result {
	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()
	return newNotifier(config).notify(ctx)
}

type notifier struct {
	currentVersion string
	stdin          io.Reader
	stdout         io.Writer
	stderr         io.Writer
	releaseURL     string
	httpClient     *http.Client
	cachePath      string
	now            func() time.Time
	getenv         func(string) string
	isTTY          func() bool
	lookup         func(string) (string, error)
	terminalWidth  func() int
	renderMarkdown func(string, int) (string, error)
	executable     func() (string, error)
	evalSymlinks   func(string) (string, error)
	runCommand     func(context.Context, string) error
}

type state struct {
	LastCheckedAt       time.Time `json:"lastCheckedAt"`
	LastShownReleaseTag string    `json:"lastShownReleaseTag"`
	LastShownFeatureID  string    `json:"lastShownFeatureID"`
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
}

type feature struct {
	ID    string
	Title string
}

type installChannel string

const (
	installChannelScript   installChannel = "script"
	installChannelHomebrew installChannel = "homebrew"
	installChannelGo       installChannel = "go-install"
)

type updateAction string

const (
	updateActionRunUpgrade           updateAction = "run-upgrade"
	updateActionSkip                 updateAction = "skip"
	updateActionSkipUntilNextVersion updateAction = "skip-until-next-version"
)

func newNotifier(config Config) notifier {
	stdin := io.Reader(os.Stdin)
	if config.Stdin != nil {
		stdin = config.Stdin
	}
	stdout := io.Writer(os.Stdout)
	if config.Stdout != nil {
		stdout = config.Stdout
	}
	stderr := io.Writer(os.Stderr)
	if config.Stderr != nil {
		stderr = config.Stderr
	}
	getenv := config.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	releaseURL := strings.TrimSpace(config.ReleaseURL)
	if releaseURL == "" {
		releaseURL = strings.TrimSpace(getenv(releaseURLEnv))
	}
	if releaseURL == "" {
		releaseURL = defaultReleaseURL
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: checkTimeout}
	}
	cachePath := config.CachePath
	if cachePath == "" {
		cachePath = defaultCachePath()
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	isTTY := config.IsTTY
	if isTTY == nil {
		isTTY = func() bool {
			stderrFile, ok := stderr.(*os.File)
			if !ok {
				return false
			}
			return isatty.IsTerminal(stderrFile.Fd())
		}
	}
	terminalWidth := config.TerminalWidth
	if terminalWidth == nil {
		terminalWidth = func() int {
			stderrFile, ok := stderr.(*os.File)
			if !ok {
				return defaultBoxWidth
			}
			fd, err := strconv.Atoi(strconv.FormatUint(uint64(stderrFile.Fd()), 10))
			if err != nil {
				return widthFromColumnsEnv(getenv)
			}
			width, _, err := term.GetSize(fd)
			if err != nil {
				return widthFromColumnsEnv(getenv)
			}
			return width
		}
	}
	renderMarkdown := config.RenderMarkdown
	if renderMarkdown == nil {
		renderMarkdown = renderMarkdownWithGlamour
	}
	lookup := config.Lookup
	if lookup == nil {
		lookup = exec.LookPath
	}
	executable := config.Executable
	if executable == nil {
		executable = os.Executable
	}
	evalSymlinks := config.EvalSymlinks
	if evalSymlinks == nil {
		evalSymlinks = filepath.EvalSymlinks
	}
	runCommand := config.RunCommand
	if runCommand == nil {
		runCommand = func(ctx context.Context, command string) error {
			cmd := exec.CommandContext(ctx, "sh", "-c", command)
			cmd.Stdin = stdin
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			return cmd.Run()
		}
	}
	return notifier{
		currentVersion: strings.TrimSpace(config.CurrentVersion),
		stdin:          stdin,
		stdout:         stdout,
		stderr:         stderr,
		releaseURL:     releaseURL,
		httpClient:     httpClient,
		cachePath:      cachePath,
		now:            now,
		getenv:         getenv,
		isTTY:          isTTY,
		lookup:         lookup,
		terminalWidth:  terminalWidth,
		renderMarkdown: renderMarkdown,
		executable:     executable,
		evalSymlinks:   evalSymlinks,
		runCommand:     runCommand,
	}
}

func (n notifier) notify(ctx context.Context) Result {
	if !n.canNotify() {
		return ResultContinue
	}
	currentState := n.readState()
	now := n.now()
	if !currentState.LastCheckedAt.IsZero() && now.Sub(currentState.LastCheckedAt) < checkInterval {
		return ResultContinue
	}

	release, err := n.fetchLatestRelease(ctx)
	currentState.LastCheckedAt = now
	if err != nil {
		n.saveState(currentState)
		return ResultContinue
	}
	if isCurrentRelease(n.currentVersion, release.TagName) {
		n.saveState(currentState)
		return ResultContinue
	}
	releaseNotesMarkdown, _ := releaseNotesSection(release.Body)
	var releaseNoteID string
	if note, ok := firstReleaseNote(releaseNotesMarkdown); ok {
		releaseNoteID = note.ID
	} else {
		releaseNotesMarkdown = ""
	}
	if alreadyShown(currentState, release.TagName, releaseNoteID) {
		n.saveState(currentState)
		return ResultContinue
	}

	updateCommand := n.updateCommand()
	action, err := n.promptUpdate(release, releaseNotesMarkdown, updateCommand)
	if err != nil {
		n.saveState(currentState)
		return ResultContinue
	}
	if action == updateActionSkipUntilNextVersion {
		currentState.LastShownReleaseTag = release.TagName
		currentState.LastShownFeatureID = releaseNoteID
	}
	n.saveState(currentState)
	if action != updateActionRunUpgrade {
		return ResultContinue
	}
	if updateCommand == "" {
		return ResultContinue
	}
	if err := n.runCommand(context.Background(), updateCommand); err != nil {
		_, _ = fmt.Fprintf(n.stderr, "failed to update sloctl: %v\n", err)
		return ResultExitFailure
	}
	return ResultExitSuccess
}

func (n notifier) canNotify() bool {
	return n.isTTY() &&
		n.cachePath != "" &&
		n.getenv(ciEnv) == "" &&
		n.getenv(optOutEnv) == "" &&
		!isDevelopmentVersion(n.currentVersion)
}

func (n notifier) fetchLatestRelease(ctx context.Context) (githubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, n.releaseURL, nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "sloctl")

	resp, err := n.httpClient.Do(req)
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

func (n notifier) promptUpdate(
	release githubRelease,
	releaseNotesMarkdown string,
	updateCommand string,
) (updateAction, error) {
	_, _ = fmt.Fprintln(n.stderr, n.renderNotification(release, releaseNotesMarkdown))
	action := updateActionSkip
	form := huhform.New(huh.NewGroup(
		huh.NewSelect[updateAction]().
			Title("Choose update action").
			Options(updateActionOptions(updateCommand)...).
			Value(&action),
	)).
		WithInput(n.stdin).
		WithOutput(n.stderr)
	return action, form.Run()
}

func updateActionOptions(updateCommand string) []huh.Option[updateAction] {
	options := []huh.Option[updateAction]{
		huh.NewOption("Skip", updateActionSkip),
		huh.NewOption("Skip until next version", updateActionSkipUntilNextVersion),
	}
	if updateCommand == "" {
		return options
	}
	return append(
		[]huh.Option[updateAction]{
			huh.NewOption(fmt.Sprintf("Update (runs %s)", updateCommand), updateActionRunUpgrade),
		},
		options...,
	)
}

func (n notifier) renderNotification(release githubRelease, releaseNotesMarkdown string) string {
	width := notificationWidth(n.terminalWidth())
	parts := []string{
		notificationTitleMarkdown(releaseNotesMarkdown, release.TagName),
	}
	if releaseNotesMarkdown != "" {
		parts = append(parts, strings.TrimSpace(releaseNotesMarkdown))
	}
	parts = append(parts, fmt.Sprintf("📜 %s", release.HTMLURL))
	markdown := strings.Join(parts, "\n\n")

	rendered := styledPlainNotification(release, releaseNotesMarkdown)
	if releaseNotesMarkdown != "" {
		var err error
		rendered, err = n.renderMarkdown(markdown, width)
		if err != nil {
			rendered = styledPlainNotification(release, releaseNotesMarkdown)
		}
		rendered = trimTrailingLineSpace(rendered)
	}
	return strings.TrimSpace(rendered)
}

func renderMarkdownWithGlamour(markdown string, width int) (string, error) {
	styleName := "dark"
	if os.Getenv("NO_COLOR") != "" {
		styleName = "ascii"
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(styleName),
		glamour.WithWordWrap(width),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return "", err
	}
	return renderer.Render(markdown)
}

func styledPlainNotification(release githubRelease, releaseNotesMarkdown string) string {
	titleStyle := style.NotificationTitle()
	linkStyle := style.NotificationLink()
	labelStyle := style.NotificationLabel()

	parts := []string{
		titleStyle.Render(notificationTitle(releaseNotesMarkdown, release.TagName)),
	}
	if releaseNotesMarkdown != "" {
		parts = append(parts, stripMarkdownHeading(releaseNotesMarkdown))
	}
	parts = append(parts, labelStyle.Render("📜")+" "+linkStyle.Render(release.HTMLURL))
	return strings.Join(parts, "\n\n")
}

func trimTrailingLineSpace(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}

func notificationTitleMarkdown(releaseNotesMarkdown, releaseTag string) string {
	if releaseNotesMarkdown == "" {
		return notificationTitle(releaseNotesMarkdown, releaseTag)
	}
	return "### " + notificationTitle(releaseNotesMarkdown, releaseTag)
}

func notificationTitle(releaseNotesMarkdown, releaseTag string) string {
	if releaseNotesMarkdown == "" {
		return fmt.Sprintf("New sloctl version %s is available!", releaseTag)
	}
	return fmt.Sprintf("New sloctl updates in %s!", releaseTag)
}

func (n notifier) updateCommand() string {
	switch n.installChannel() {
	case installChannelHomebrew:
		return "brew upgrade sloctl"
	case installChannelGo:
		return "go install github.com/nobl9/sloctl/cmd/sloctl@latest"
	case installChannelScript:
		return n.scriptUpdateCommand()
	default:
		return ""
	}
}

func (n notifier) installChannel() installChannel {
	executablePath, err := n.executable()
	if err != nil {
		return installChannelScript
	}
	resolvedPath, err := n.evalSymlinks(executablePath)
	if err != nil {
		resolvedPath = executablePath
	}
	if isHomebrewExecutable(resolvedPath) {
		return installChannelHomebrew
	}
	if isGoInstallExecutable(resolvedPath, n.getenv) {
		return installChannelGo
	}
	return installChannelScript
}

func (n notifier) scriptUpdateCommand() string {
	const scriptURL = "https://raw.githubusercontent.com/nobl9/sloctl/main/install.bash"
	if _, err := n.lookup("curl"); err == nil {
		return "curl -fsSL " + scriptURL + " | bash"
	}
	if _, err := n.lookup("wget"); err == nil {
		return "wget -O - -q " + scriptURL + " | bash"
	}
	return ""
}

func isHomebrewExecutable(path string) bool {
	return strings.Contains(filepath.ToSlash(path), "/Cellar/sloctl/")
}

func isGoInstallExecutable(path string, getenv func(string) string) bool {
	path = filepath.Clean(path)
	for _, binDir := range goBinDirs(getenv) {
		if samePath(path, filepath.Join(binDir, "sloctl")) ||
			samePath(path, filepath.Join(binDir, "sloctl.exe")) {
			return true
		}
	}
	return false
}

func goBinDirs(getenv func(string) string) []string {
	if goBin := strings.TrimSpace(getenv("GOBIN")); goBin != "" {
		return []string{goBin}
	}
	goPath := strings.TrimSpace(getenv("GOPATH"))
	if goPath == "" {
		homeDir := strings.TrimSpace(getenv("HOME"))
		if homeDir == "" {
			homeDir = strings.TrimSpace(getenv("USERPROFILE"))
		}
		if homeDir == "" {
			return nil
		}
		goPath = filepath.Join(homeDir, "go")
	}
	var binDirs []string
	for _, path := range filepath.SplitList(goPath) {
		if path != "" {
			binDirs = append(binDirs, filepath.Join(path, "bin"))
		}
	}
	return binDirs
}

func samePath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

func notificationWidth(terminalWidth int) int {
	if terminalWidth <= 0 {
		return defaultBoxWidth
	}
	return min(max(terminalWidth-2, minBoxWidth), defaultBoxWidth)
}

func widthFromColumnsEnv(getenv func(string) string) int {
	width, err := strconv.Atoi(getenv("COLUMNS"))
	if err != nil {
		return defaultBoxWidth
	}
	return width
}

func releaseNotesSection(body string) (string, bool) {
	var section []string
	inReleaseNotesSection := false
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if isLevel2Heading(trimmed) {
			if inReleaseNotesSection {
				if _, ok := firstReleaseNote(strings.Join(section, "\n")); ok {
					break
				}
			}
			inReleaseNotesSection = isReleaseNotesHeading(trimmed)
			section = nil
		}
		if inReleaseNotesSection {
			section = append(section, line)
		}
	}
	markdown := strings.TrimSpace(strings.Join(section, "\n"))
	if _, ok := firstReleaseNote(markdown); !ok {
		return "", false
	}
	return markdown, true
}

func firstReleaseNote(body string) (feature, bool) {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		return parseFeature(trimmed[2:]), true
	}
	return feature{}, false
}

func stripMarkdownHeading(markdown string) string {
	lines := strings.Split(markdown, "\n")
	if len(lines) == 0 {
		return markdown
	}
	if strings.HasPrefix(strings.TrimSpace(lines[0]), "##") {
		return strings.TrimSpace(strings.Join(lines[1:], "\n"))
	}
	return strings.TrimSpace(markdown)
}

func isLevel2Heading(line string) bool {
	return strings.HasPrefix(line, "## ") || line == "##"
}

func isReleaseNotesHeading(line string) bool {
	heading := strings.ToLower(line)
	return strings.Contains(heading, "features") || strings.Contains(heading, "fixes")
}

func parseFeature(raw string) feature {
	title := strings.TrimSpace(raw)
	title = strings.TrimPrefix(title, "feat:")
	title = strings.TrimPrefix(title, "Feat:")
	title = strings.TrimPrefix(title, "fix:")
	title = strings.TrimPrefix(title, "Fix:")
	title = strings.TrimSpace(title)
	id := title
	if matches := releasePRPattern.FindStringSubmatch(raw); len(matches) == 2 {
		id = matches[1]
	}
	title = releaseMetadataPattern.ReplaceAllString(title, "")
	return feature{
		ID:    id,
		Title: strings.TrimSpace(title),
	}
}

func alreadyShown(currentState state, releaseTag, featureID string) bool {
	return currentState.LastShownReleaseTag == releaseTag && currentState.LastShownFeatureID == featureID
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
