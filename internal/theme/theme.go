package theme

import "github.com/charmbracelet/huh"

var defaultTheme = huh.ThemeBase16()

// GetDefault returns a default [huh] theme which should be used by all prompts.
func GetDefault() *huh.Theme {
	return defaultTheme
}
