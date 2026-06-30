package notifications

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	huh "charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour"
	"golang.org/x/term"

	"github.com/nobl9/sloctl/internal/huhform"
	"github.com/nobl9/sloctl/internal/style"
)

const (
	defaultPromptWidth = 92
	minPromptWidth     = 48
)

type updateAction string

const (
	updateActionRunUpgrade           updateAction = "run-upgrade"
	updateActionSkip                 updateAction = "skip"
	updateActionSkipUntilNextVersion updateAction = "skip-until-next-version"
)

type terminalInfo struct {
	width int
	dark  bool
}

func (n notifier) promptUpdate(
	release githubRelease,
	releaseNotesMarkdown string,
	updateCommand string,
) (updateAction, error) {
	terminal := n.terminalInfo()
	n.printNotification(release, releaseNotesMarkdown, terminal)

	action := defaultUpdateAction(updateCommand)
	form := huhform.NewWithTheme(
		huh.ThemeFunc(func(bool) *huh.Styles {
			return style.HuhTheme(terminal.dark)
		}),
		huh.NewGroup(
			huh.NewSelect[updateAction]().
				Title("Choose update action").
				Options(updateActionOptions(updateCommand)...).
				Value(&action),
		),
	).
		WithInput(n.stdin).
		WithOutput(n.stderr)
	return action, form.Run()
}

func (n notifier) printNotification(
	release githubRelease,
	releaseNotesMarkdown string,
	terminal terminalInfo,
) {
	_, _ = fmt.Fprintln(n.stderr, renderNotification(release, releaseNotesMarkdown, terminal.width, terminal.dark))
	_, _ = fmt.Fprintln(n.stderr)
	separator := style.NotificationSeparator(terminal.dark).Render(strings.Repeat("─", terminal.width))
	_, _ = fmt.Fprintln(n.stderr, separator)
	_, _ = fmt.Fprintln(n.stderr)
}

func (n notifier) terminalInfo() terminalInfo {
	isDark := true
	if style.ColorEnabled() {
		isDark = lipgloss.HasDarkBackground(n.stdin, n.stderr)
	}
	return terminalInfo{
		width: notificationWidth(n.terminalWidth()),
		dark:  isDark,
	}
}

func (n notifier) terminalWidth() int {
	//nolint:gosec // File descriptors are small non-negative integers.
	fd := int(n.stderr.Fd())
	width, _, err := term.GetSize(fd)
	if err != nil {
		return widthFromColumnsEnv()
	}
	return width
}

func updateActionOptions(updateCommand string) []huh.Option[updateAction] {
	if updateCommand == "" {
		return []huh.Option[updateAction]{
			huh.NewOption("Skip", updateActionSkip),
			huh.NewOption("Skip until next version", updateActionSkipUntilNextVersion),
		}
	}
	return []huh.Option[updateAction]{
		huh.NewOption(fmt.Sprintf("Update (runs %s)", updateCommand), updateActionRunUpgrade),
		huh.NewOption("Skip", updateActionSkip),
		huh.NewOption("Skip until next version", updateActionSkipUntilNextVersion),
	}
}

func defaultUpdateAction(updateCommand string) updateAction {
	if updateCommand == "" {
		return updateActionSkip
	}
	return updateActionRunUpgrade
}

func notificationWidth(terminalWidth int) int {
	if terminalWidth <= 0 {
		return defaultPromptWidth
	}
	return min(max(terminalWidth-2, minPromptWidth), defaultPromptWidth)
}

func widthFromColumnsEnv() int {
	width, err := strconv.Atoi(os.Getenv("COLUMNS"))
	if err != nil {
		return defaultPromptWidth
	}
	return width
}

func renderNotification(release githubRelease, releaseNotesMarkdown string, width int, isDark bool) string {
	plainReleaseNotesDisplay := displayReleaseNotesMarkdown(releaseNotesMarkdown, false)
	hasReleaseNotes := plainReleaseNotesDisplay != ""
	rendered := styledPlainNotification(release, plainReleaseNotesDisplay, isDark)
	if hasReleaseNotes {
		releaseNotesDisplay := plainReleaseNotesDisplay
		if style.ColorEnabled() {
			releaseNotesDisplay = displayReleaseNotesMarkdown(releaseNotesMarkdown, true)
		}
		markdown := strings.Join([]string{
			"# " + releaseChangesTitle(release.TagName),
			releaseNotesDisplay,
			fmt.Sprintf("📜 %s", release.HTMLURL),
		}, "\n\n")

		var err error
		rendered, err = renderMarkdownWithGlamour(markdown, width, isDark)
		if err != nil {
			rendered = styledPlainNotification(release, plainReleaseNotesDisplay, isDark)
		}
		rendered = trimTrailingLineSpace(rendered)
	}
	return strings.TrimSpace(rendered)
}

func renderMarkdownWithGlamour(markdown string, width int, isDark bool) (string, error) {
	styleConfig := style.NotificationMarkdownStyle(isDark)
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(styleConfig),
		glamour.WithWordWrap(width),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return "", err
	}
	return renderer.Render(markdown)
}

func styledPlainNotification(release githubRelease, releaseNotesDisplay string, isDark bool) string {
	titleStyle := style.NotificationTitle(isDark)
	linkStyle := style.NotificationLink(isDark)
	labelStyle := style.NotificationLabel(isDark)
	hasReleaseNotes := releaseNotesDisplay != ""

	parts := []string{
		titleStyle.Render(notificationTitle(hasReleaseNotes, release.TagName)),
	}
	if hasReleaseNotes {
		parts = append(parts, releaseNotesDisplay)
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

func notificationTitle(hasReleaseNotes bool, releaseTag string) string {
	if !hasReleaseNotes {
		return newVersionTitle(releaseTag)
	}
	return releaseChangesTitle(releaseTag)
}

func newVersionTitle(releaseTag string) string {
	return fmt.Sprintf("New sloctl version %s is available!", releaseTag)
}

func releaseChangesTitle(releaseTag string) string {
	return fmt.Sprintf("Changes in version %s", releaseTag)
}
