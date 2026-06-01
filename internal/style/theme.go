// Package style defines shared sloctl terminal styles.
package style

import (
	huh "charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

const (
	tealHex      = "#00819E"
	cyanHex      = "#63D6E5"
	mutedHex     = "#BABBBB"
	lightGrayHex = "#E8E9E9"
	darkGrayHex  = "#383939"
	whiteHex     = "#FFFFFF"
	greenHex     = "#0EB46E"
	yellowHex    = "#C8D655"
	pinkHex      = "#DB2779"
	redHex       = "#D42E56"
)

var (
	// Teal is the primary sloctl terminal accent color.
	Teal = lipgloss.Color(tealHex)
	// Cyan is the secondary sloctl terminal accent color.
	Cyan = lipgloss.Color(cyanHex)
	// Muted is used for secondary terminal text.
	Muted = lipgloss.Color(mutedHex)
	// LightGray is used for emphasized neutral terminal text.
	LightGray = lipgloss.Color(lightGrayHex)
	// DarkGray is used for low-emphasis backgrounds.
	DarkGray = lipgloss.Color(darkGrayHex)
	// White is used for primary terminal text.
	White = lipgloss.Color(whiteHex)
	// Green is used for successful or selected terminal states.
	Green = lipgloss.Color(greenHex)
	// Yellow is used for active terminal indicators.
	Yellow = lipgloss.Color(yellowHex)
	// Pink is used for focused terminal actions.
	Pink = lipgloss.Color(pinkHex)
	// Red is used for terminal error states.
	Red = lipgloss.Color(redHex)
)

// HuhTheme returns the shared Nobl9 terminal theme for interactive forms.
func HuhTheme(isDark bool) *huh.Styles {
	t := huh.ThemeBase(isDark)

	t.Focused.Base = t.Focused.Base.BorderForeground(Teal)
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(Cyan)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(Cyan)
	t.Focused.Directory = t.Focused.Directory.Foreground(Cyan)
	t.Focused.Description = t.Focused.Description.Foreground(Muted)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(Red)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(Red)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(Yellow)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(Yellow)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(Yellow)
	t.Focused.Option = t.Focused.Option.Foreground(White)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(Yellow)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(Green)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(Green)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(White)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(White).Background(Pink)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(White).Background(DarkGray)

	t.Focused.TextInput.Cursor.Foreground(Pink)
	t.Focused.TextInput.Placeholder.Foreground(Muted)
	t.Focused.TextInput.Prompt.Foreground(Yellow)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NoteTitle = t.Blurred.NoteTitle.Foreground(Muted)
	t.Blurred.Title = t.Blurred.NoteTitle.Foreground(Muted)

	t.Blurred.TextInput.Prompt = t.Blurred.TextInput.Prompt.Foreground(Muted)
	t.Blurred.TextInput.Text = t.Blurred.TextInput.Text.Foreground(White)

	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description

	return t
}

// NotificationBorder returns the shared border for notification containers.
func NotificationBorder() lipgloss.Border {
	return lipgloss.RoundedBorder()
}

// NotificationBox returns the shared style for notification containers.
func NotificationBox(contentWidth int) lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(NotificationBorder()).
		BorderForeground(Teal).
		Padding(0, 1).
		Width(contentWidth)
}

// NotificationTitle returns the shared style for notification titles.
func NotificationTitle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(White)
}

// NotificationLink returns the shared style for notification links.
func NotificationLink() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Cyan).Underline(true)
}

// NotificationLabel returns the shared style for notification labels.
func NotificationLabel() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Muted)
}

// NotificationCommand returns the shared style for notification commands.
func NotificationCommand() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(LightGray).Italic(true)
}
