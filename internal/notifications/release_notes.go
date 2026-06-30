package notifications

import (
	"regexp"
	"strings"
)

var releaseMetadataPattern = regexp.MustCompile(`\s+\(#\d+\)(?:\s+@\S+)?$`)

func extractReleaseNotesMarkdown(body string) string {
	var sections []string
	var section []string
	inReleaseNotesSection := false
	appendSection := func() {
		markdown := strings.TrimSpace(strings.Join(section, "\n"))
		if inReleaseNotesSection && hasTopLevelReleaseNote(markdown) {
			sections = append(sections, markdown)
		}
	}

	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if markdownHeadingLevel(trimmed) == 2 {
			appendSection()
			inReleaseNotesSection = isReleaseNotesHeading(trimmed)
			section = nil
		}
		if inReleaseNotesSection {
			section = append(section, line)
		}
	}
	appendSection()
	return strings.TrimSpace(strings.Join(sections, "\n\n"))
}

func hasTopLevelReleaseNote(markdown string) bool {
	inNestedSection := false
	for _, line := range strings.Split(markdown, "\n") {
		inNestedSection = updateNestedSectionState(line, inNestedSection)
		if isTopLevelReleaseNoteLine(line, inNestedSection) {
			return true
		}
	}
	return false
}

func updateNestedSectionState(line string, current bool) bool {
	level := markdownHeadingLevel(line)
	switch {
	case level == 2:
		return false
	case level > 2:
		return true
	default:
		return current
	}
}

func isTopLevelReleaseNoteLine(line string, inNestedSection bool) bool {
	return !inNestedSection && strings.HasPrefix(line, "- ")
}

func markdownHeadingLevel(line string) int {
	line = strings.TrimSpace(line)
	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	if level == 0 || level > 6 {
		return 0
	}
	if len(line) == level || line[level] == ' ' {
		return level
	}
	return 0
}

func displayReleaseNotesMarkdown(markdown string, highlightTitles bool) string {
	lines := strings.Split(markdown, "\n")
	inNestedSection := false
	for i, line := range lines {
		inNestedSection = updateNestedSectionState(line, inNestedSection)
		if !isTopLevelReleaseNoteLine(line, inNestedSection) {
			continue
		}
		title := parseReleaseNote(line[2:])
		if title == "" {
			continue
		}
		if highlightTitles {
			title = "**" + title + "**"
		}
		lines[i] = "- " + title
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func isReleaseNotesHeading(line string) bool {
	heading := strings.ToLower(line)
	return strings.Contains(heading, "features") ||
		strings.Contains(heading, "fixes") ||
		strings.Contains(heading, "breaking")
}

func parseReleaseNote(raw string) string {
	title := trimReleaseNotePrefix(strings.TrimSpace(raw))
	title = releaseMetadataPattern.ReplaceAllString(title, "")
	return strings.TrimSpace(title)
}

func trimReleaseNotePrefix(title string) string {
	lowerTitle := strings.ToLower(title)
	for _, prefix := range [...]string{
		"feat:",
		"fix:",
		"breaking:",
	} {
		if strings.HasPrefix(lowerTitle, prefix) {
			return strings.TrimSpace(title[len(prefix):])
		}
	}
	return title
}
