// Package style defines shared sloctl terminal styles.
package style

import (
	"os"

	huh "charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

const (
	pinkHex = "#DB2779"
	tealHex = "#00819E"
)

var (
	darkGray  = lipgloss.Color("#383939")
	white     = lipgloss.Color("#FFFFFF")
	pink      = lipgloss.Color(pinkHex)
	red       = lipgloss.Color("#D42E56")
	teal      = lipgloss.Color(tealHex)
	cyan      = lipgloss.Color("#00BAD3")
	gray      = lipgloss.Color("#676868")
	lightGray = lipgloss.Color("#BABBBB")
	darkTeal  = lipgloss.Color("#01465C")
)

// HuhTheme returns the shared Nobl9 terminal theme for interactive forms.
func HuhTheme(isDark bool) *huh.Styles {
	t := huh.ThemeBase(isDark)
	if os.Getenv("NO_COLOR") != "" {
		return plainHuhTheme(t)
	}

	lightDark := lipgloss.LightDark(isDark)
	accent := lightDark(teal, cyan)
	muted := lightDark(gray, lightGray)
	// Keep ordinary text readable even when background detection is unavailable.
	text := lipgloss.NoColor{}
	selectedBackground := lightDark(teal, darkTeal)
	t.Focused.Base = t.Focused.Base.BorderForeground(accent)
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(accent)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(accent)
	t.Focused.Directory = t.Focused.Directory.Foreground(accent)
	t.Focused.Description = t.Focused.Description.Foreground(muted)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(red)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(red)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(accent)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(accent)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(accent)
	t.Focused.Option = t.Focused.Option.Foreground(text).UnsetFaint()
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(accent)
	t.Focused.SelectedOption = t.Focused.SelectedOption.
		Foreground(white).
		Background(selectedBackground).
		UnsetFaint().
		Bold(true)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(accent)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(text).UnsetFaint()
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(text).UnsetFaint()
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(white).Background(pink)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(white).Background(darkGray)

	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(pink)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(muted)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(accent)

	t.Help.ShortKey = t.Help.ShortKey.Foreground(accent)
	t.Help.FullKey = t.Help.FullKey.Foreground(accent)
	t.Help.ShortDesc = t.Help.ShortDesc.Foreground(muted)
	t.Help.FullDesc = t.Help.FullDesc.Foreground(muted)
	t.Help.ShortSeparator = t.Help.ShortSeparator.Foreground(muted)
	t.Help.FullSeparator = t.Help.FullSeparator.Foreground(muted)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NoteTitle = t.Blurred.NoteTitle.Foreground(muted)
	t.Blurred.Title = t.Blurred.Title.Foreground(muted)

	t.Blurred.TextInput.Prompt = t.Blurred.TextInput.Prompt.Foreground(muted)
	t.Blurred.TextInput.Text = t.Blurred.TextInput.Text.Foreground(text)

	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description

	return t
}

func plainHuhTheme(t *huh.Styles) *huh.Styles {
	// ThemeBase applies colors only to buttons, placeholders, and help text.
	for _, s := range []*lipgloss.Style{
		&t.Focused.FocusedButton,
		&t.Focused.BlurredButton,
		&t.Focused.TextInput.Placeholder,
		&t.Blurred.FocusedButton,
		&t.Blurred.BlurredButton,
		&t.Blurred.TextInput.Placeholder,
		&t.Help.ShortKey,
		&t.Help.ShortDesc,
		&t.Help.ShortSeparator,
		&t.Help.FullKey,
		&t.Help.FullDesc,
		&t.Help.FullSeparator,
	} {
		*s = s.UnsetForeground().UnsetBackground()
	}
	return t
}
