package notifications

import (
	"regexp"
	"strings"
)

var releaseMetadataPattern = regexp.MustCompile(`\s+\(#\d+\)(?:\s+@\S+)?$`)

func releaseHighlights(body string) []string {
	var sections []string
	var section []string
	var includeSection, inNestedSection, hasNote bool
	appendSection := func() {
		if includeSection && hasNote {
			sections = append(sections, strings.TrimSpace(strings.Join(section, "\n")))
		}
	}
	for _, line := range strings.Split(body, "\n") {
		level := markdownHeadingLevel(line)
		if level == 2 {
			appendSection()
			includeSection = isReleaseNotesHeading(line)
			inNestedSection, hasNote = false, false
			section = nil
		}
		if !includeSection {
			continue
		}
		if level > 2 {
			inNestedSection = true
		}
		if !inNestedSection && strings.HasPrefix(line, "- ") {
			if title := parseReleaseNote(line[2:]); title != "" {
				line = "- " + title
				hasNote = true
			}
		}
		section = append(section, line)
	}
	appendSection()
	return sections
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

func isReleaseNotesHeading(line string) bool {
	line = strings.TrimSpace(line)
	if markdownHeadingLevel(line) != 2 {
		return false
	}
	heading := strings.ToLower(strings.TrimSpace(line[2:]))
	for _, suffix := range [...]string{
		"features",
		"fixes",
		"breaking",
		"breaking changes",
		"vulnerabilities",
	} {
		if heading == suffix || strings.HasSuffix(heading, " "+suffix) {
			return true
		}
	}
	return false
}

func parseReleaseNote(raw string) string {
	title := strings.TrimSpace(raw)
	for _, prefix := range [...]string{"feat:", "fix:", "breaking:", "sec:"} {
		if strings.HasPrefix(strings.ToLower(title), prefix) {
			title = strings.TrimSpace(title[len(prefix):])
			break
		}
	}
	return strings.TrimSpace(releaseMetadataPattern.ReplaceAllString(title, ""))
}
