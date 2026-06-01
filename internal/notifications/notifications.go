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

	"github.com/charmbracelet/glamour"
	"github.com/mattn/go-isatty"
	"github.com/nobl9/sloctl/internal/style"
	"golang.org/x/term"
)

const (
	defaultReleaseURL = "https://api.github.com/repos/nobl9/sloctl/releases/latest"
	optOutEnv         = "SLOCTL_NO_NOTIFICATIONS"
	ciEnv             = "CI"
	checkInterval     = 24 * time.Hour
	checkTimeout      = 750 * time.Millisecond
	maxResponseSize   = 1 << 20
	defaultBoxWidth   = 92
	minBoxWidth       = 48
	boxPadding        = 2
)

var (
	releaseMetadataPattern = regexp.MustCompile(`\s+\(#\d+\)(?:\s+@\S+)?$`)
	releasePRPattern       = regexp.MustCompile(`#(\d+)`)
)

// Config defines notification runtime dependencies.
type Config struct {
	CurrentVersion string
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
}

// Notify checks and displays an unobtrusive notification when configured to do so.
func Notify(config Config) {
	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()
	newNotifier(config).notify(ctx)
}

type notifier struct {
	currentVersion string
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

func newNotifier(config Config) notifier {
	stderr := io.Writer(os.Stderr)
	if config.Stderr != nil {
		stderr = config.Stderr
	}
	releaseURL := config.ReleaseURL
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
	getenv := config.Getenv
	if getenv == nil {
		getenv = os.Getenv
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
	return notifier{
		currentVersion: strings.TrimSpace(config.CurrentVersion),
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
	}
}

func (n notifier) notify(ctx context.Context) {
	if !n.canNotify() {
		return
	}
	currentState := n.readState()
	now := n.now()
	if !currentState.LastCheckedAt.IsZero() && now.Sub(currentState.LastCheckedAt) < checkInterval {
		return
	}

	release, err := n.fetchLatestRelease(ctx)
	currentState.LastCheckedAt = now
	if err != nil {
		n.saveState(currentState)
		return
	}
	if isCurrentRelease(n.currentVersion, release.TagName) {
		n.saveState(currentState)
		return
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
		return
	}

	_, _ = fmt.Fprintln(n.stderr, n.renderNotification(release, releaseNotesMarkdown))
	currentState.LastShownReleaseTag = release.TagName
	currentState.LastShownFeatureID = releaseNoteID
	n.saveState(currentState)
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

func (n notifier) renderNotification(release githubRelease, releaseNotesMarkdown string) string {
	width := notificationWidth(n.terminalWidth())
	border := style.NotificationBorder()
	contentWidth := width - boxPadding - border.GetLeftSize() - border.GetRightSize()
	updateCommand := n.updateCommand()
	formattedUpdateCommand := formatShellPipeline(updateCommand, contentWidth-boxPadding*2)
	parts := []string{
		notificationTitleMarkdown(releaseNotesMarkdown, release.TagName),
	}
	if releaseNotesMarkdown != "" {
		parts = append(parts, strings.TrimSpace(releaseNotesMarkdown))
	}
	parts = append(parts, fmt.Sprintf("📜 %s", release.HTMLURL))
	if updateCommand != "" {
		parts = append(parts, fmt.Sprintf("Update sloctl with:\n\n```shell\n%s\n```", formattedUpdateCommand))
	}
	markdown := strings.Join(parts, "\n\n")

	rendered := styledPlainNotification(release, releaseNotesMarkdown, formattedUpdateCommand)
	if releaseNotesMarkdown != "" {
		var err error
		rendered, err = n.renderMarkdown(markdown, contentWidth)
		if err != nil {
			rendered = styledPlainNotification(release, releaseNotesMarkdown, formattedUpdateCommand)
		}
	}
	rendered = strings.TrimSpace(rendered)

	return style.NotificationBox(contentWidth).Render(rendered)
}

func renderMarkdownWithGlamour(markdown string, width int) (string, error) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return "", err
	}
	return renderer.Render(markdown)
}

func styledPlainNotification(release githubRelease, releaseNotesMarkdown, updateCommand string) string {
	titleStyle := style.NotificationTitle()
	linkStyle := style.NotificationLink()
	labelStyle := style.NotificationLabel()
	commandStyle := style.NotificationCommand()

	parts := []string{
		titleStyle.Render(notificationTitle(releaseNotesMarkdown, release.TagName)),
	}
	if releaseNotesMarkdown != "" {
		parts = append(parts, stripMarkdownHeading(releaseNotesMarkdown))
	}
	parts = append(parts, labelStyle.Render("📜")+" "+linkStyle.Render(release.HTMLURL))
	if updateCommand != "" {
		parts = append(
			parts,
			labelStyle.Render("Update sloctl with:")+"\n"+commandStyle.Render(updateCommand),
		)
	}
	return strings.Join(parts, "\n\n")
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

func formatShellPipeline(command string, maxLineWidth int) string {
	if len(command) <= maxLineWidth {
		return command
	}
	left, right, ok := strings.Cut(command, " | ")
	if !ok {
		return command
	}
	pipeline := left + " \\\n| " + right
	if maxLineLen(pipeline) <= maxLineWidth {
		return pipeline
	}
	if url, ok := strings.CutPrefix(left, "curl -fsSL "); ok {
		curlPipeline := "curl -fsSL \\\n" + url + " | " + right
		if maxLineLen(curlPipeline) <= maxLineWidth {
			return curlPipeline
		}
		return "curl -fsSL \\\n  " + url + " \\\n  | " + right
	}
	return pipeline
}

func maxLineLen(text string) int {
	maxLen := 0
	for _, line := range strings.Split(text, "\n") {
		maxLen = max(maxLen, len(line))
	}
	return maxLen
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
