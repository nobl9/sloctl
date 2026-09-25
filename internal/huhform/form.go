package huhform

import (
	"fmt"
	"os"
	"strconv"

	tea "charm.land/bubbletea/v2"
	huh "charm.land/huh/v2"

	"github.com/nobl9/sloctl/internal/style"
)

// accessibleModeEnv can be set to turn on [huh] accessible mode.
// It can be useful in old terminal emulators (e.g. remote shells).
const accessibleModeEnv = "SLOCTL_ACCESSIBLE_MODE"

// New returns a form configured with sloctl's shared terminal theme.
func New(groups ...*huh.Group) *huh.Form {
	accessible := AccessibleMode()
	isDark := true
	form := huh.NewForm(groups...).
		WithTheme(huh.ThemeFunc(func(bool) *huh.Styles { return style.HuhTheme(isDark) })).
		WithAccessible(accessible)
	if accessible || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return form
	}
	requested := false
	// Huh handles background responses but does not request them. Keep the query
	// in Bubble Tea's event loop so keyboard input and cancellation are retained.
	// WithProgramOptions replaces Huh's defaults, including its stderr output.
	return form.WithProgramOptions(tea.WithOutput(os.Stderr), tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
		if background, ok := msg.(tea.BackgroundColorMsg); ok {
			isDark = background.IsDark()
		}
		if requested {
			return msg
		}
		requested = true
		return tea.Batch(tea.RequestBackgroundColor, func() tea.Msg { return msg })()
	}))
}

// AccessibleMode reports whether SLOCTL_ACCESSIBLE_MODE enables plain-text prompts.
func AccessibleMode() bool {
	v, ok := os.LookupEnv(accessibleModeEnv)
	if !ok {
		return false
	}
	accessible, err := strconv.ParseBool(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid value: %q for %q environment variable. Error: %v", v, accessibleModeEnv, err)
	}
	return accessible
}
