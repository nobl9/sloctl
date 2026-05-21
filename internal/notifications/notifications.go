// Package notifications handles unobtrusive CLI notifications.
package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour"
	"github.com/mattn/go-isatty"
	"golang.org/x/term"
)

const (
	defaultReleaseURL = "https://api.github.com/repos/nobl9/sloctl/releases/latest"
	optOutEnv         = "SLOCTL_NO_NOTIFICATIONS"
	ciEnv             = "CI"
	checkInterval     = 24 * time.Hour
	checkTimeout      = 750 * time.Millisecond
	maxResponseSize   = 1 << 20
	defaultBoxWidth   = 88
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

	TerminalWidth  func() int
	RenderMarkdown func(string, int) (string, error)
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
	terminalWidth  func() int
	renderMarkdown func(string, int) (string, error)
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
	return notifier{
		currentVersion: strings.TrimSpace(config.CurrentVersion),
		stderr:         stderr,
		releaseURL:     releaseURL,
		httpClient:     httpClient,
		cachePath:      cachePath,
		now:            now,
		getenv:         getenv,
		isTTY:          isTTY,
		terminalWidth:  terminalWidth,
		renderMarkdown: renderMarkdown,
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
	featuresMarkdown, ok := featuresSection(release.Body)
	if !ok {
		n.saveState(currentState)
		return
	}
	nextFeature, ok := firstFeature(featuresMarkdown)
	if !ok {
		n.saveState(currentState)
		return
	}
	if alreadyShown(currentState, release.TagName, nextFeature.ID) {
		n.saveState(currentState)
		return
	}

	_, _ = fmt.Fprintln(n.stderr, n.renderNotification(release, featuresMarkdown))
	currentState.LastShownReleaseTag = release.TagName
	currentState.LastShownFeatureID = nextFeature.ID
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

func (n notifier) renderNotification(release githubRelease, featuresMarkdown string) string {
	width := notificationWidth(n.terminalWidth())
	border := lipgloss.RoundedBorder()
	contentWidth := width - boxPadding - border.GetLeftSize() - border.GetRightSize()
	markdown := strings.Join([]string{
		fmt.Sprintf("### New sloctl features in %s", release.TagName),
		strings.TrimSpace(featuresMarkdown),
		fmt.Sprintf("[View release](%s)", release.HTMLURL),
	}, "\n\n")

	rendered, err := n.renderMarkdown(markdown, contentWidth)
	if err != nil {
		rendered = plainNotification(release, featuresMarkdown)
	}
	rendered = strings.TrimSpace(rendered)

	return lipgloss.NewStyle().
		BorderStyle(border).
		BorderForeground(lipgloss.Color("#0EB46E")).
		Padding(0, 1).
		Width(contentWidth).
		Render(rendered)
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

func plainNotification(release githubRelease, featuresMarkdown string) string {
	return strings.Join([]string{
		fmt.Sprintf("New sloctl features in %s", release.TagName),
		stripMarkdownHeading(featuresMarkdown),
		release.HTMLURL,
	}, "\n\n")
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

func featuresSection(body string) (string, bool) {
	var lines []string
	inFeaturesSection := false
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if isLevel2Heading(trimmed) {
			if inFeaturesSection {
				break
			}
			inFeaturesSection = strings.Contains(trimmed, "Features")
		}
		if inFeaturesSection {
			lines = append(lines, line)
		}
	}
	section := strings.TrimSpace(strings.Join(lines, "\n"))
	return section, section != ""
}

func firstFeature(body string) (feature, bool) {
	inFeaturesSection := false
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if isLevel2Heading(trimmed) {
			inFeaturesSection = strings.Contains(trimmed, "Features")
			continue
		}
		if !inFeaturesSection || !strings.HasPrefix(trimmed, "- ") {
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

func parseFeature(raw string) feature {
	title := strings.TrimSpace(raw)
	title = strings.TrimPrefix(title, "feat:")
	title = strings.TrimPrefix(title, "Feat:")
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
