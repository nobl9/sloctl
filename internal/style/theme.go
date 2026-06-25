// Package style defines shared sloctl terminal styles.
package style

import (
	"os"

	huh "charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

const (
	tealHex      = "#00819E"
	cyanHex      = "#00D5FF"
	mutedHex     = "#BABBBB"
	lightGrayHex = "#E8E9E9"
	darkGrayHex  = "#383939"
	whiteHex     = "#FFFFFF"
	greenHex     = "#0EB46E"
	yellowHex    = "#C8D655"
	pinkHex      = "#DB2779"
	redHex       = "#D42E56"
	noColorEnv   = "NO_COLOR"
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
	if noColor() {
		return plainHuhTheme(t)
	}

	optionText := White
	if !isDark {
		optionText = DarkGray
	}

	t.Focused.Base = t.Focused.Base.BorderForeground(Teal)
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(Cyan)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(Cyan)
	t.Focused.Directory = t.Focused.Directory.Foreground(Cyan)
	t.Focused.Description = t.Focused.Description.Foreground(Muted)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(Red)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(Red)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(Teal)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(Teal)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(Teal)
	t.Focused.Option = t.Focused.Option.Foreground(optionText)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(Teal)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(Green)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(Green)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(optionText)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(White).Background(Pink)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(White).Background(DarkGray)

	t.Focused.TextInput.Cursor.Foreground(Pink)
	t.Focused.TextInput.Placeholder.Foreground(Muted)
	t.Focused.TextInput.Prompt.Foreground(Teal)

	t.Help.ShortKey = t.Help.ShortKey.Foreground(Teal)
	t.Help.FullKey = t.Help.FullKey.Foreground(Teal)
	t.Help.ShortDesc = t.Help.ShortDesc.Foreground(Muted)
	t.Help.FullDesc = t.Help.FullDesc.Foreground(Muted)
	t.Help.ShortSeparator = t.Help.ShortSeparator.Foreground(Muted)
	t.Help.FullSeparator = t.Help.FullSeparator.Foreground(Muted)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NoteTitle = t.Blurred.NoteTitle.Foreground(Muted)
	t.Blurred.Title = t.Blurred.NoteTitle.Foreground(Muted)

	t.Blurred.TextInput.Prompt = t.Blurred.TextInput.Prompt.Foreground(Muted)
	t.Blurred.TextInput.Text = t.Blurred.TextInput.Text.Foreground(optionText)

	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description

	return t
}

// NotificationTitle returns the shared style for notification titles.
func NotificationTitle() lipgloss.Style {
	if noColor() {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(White).Bold(true)
}

// NotificationLink returns the shared style for notification links.
func NotificationLink() lipgloss.Style {
	if noColor() {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(Cyan).Underline(true)
}

// NotificationLabel returns the shared style for notification labels.
func NotificationLabel() lipgloss.Style {
	if noColor() {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(Muted)
}

func noColor() bool {
	return os.Getenv(noColorEnv) != ""
}

func plainHuhTheme(t *huh.Styles) *huh.Styles {
	t.Form.Base = plainStyle(t.Form.Base)
	t.Group.Base = plainStyle(t.Group.Base)
	t.Group.Title = plainStyle(t.Group.Title)
	t.Group.Description = plainStyle(t.Group.Description)
	t.FieldSeparator = plainStyle(t.FieldSeparator)
	t.Focused = plainFieldStyles(t.Focused)
	t.Blurred = plainFieldStyles(t.Blurred)
	t.Help.ShortKey = plainStyle(t.Help.ShortKey)
	t.Help.FullKey = plainStyle(t.Help.FullKey)
	t.Help.ShortDesc = plainStyle(t.Help.ShortDesc)
	t.Help.FullDesc = plainStyle(t.Help.FullDesc)
	t.Help.ShortSeparator = plainStyle(t.Help.ShortSeparator)
	t.Help.FullSeparator = plainStyle(t.Help.FullSeparator)
	return t
}

func plainFieldStyles(s huh.FieldStyles) huh.FieldStyles {
	s.Base = plainStyle(s.Base)
	s.Title = plainStyle(s.Title)
	s.Description = plainStyle(s.Description)
	s.ErrorIndicator = plainStyle(s.ErrorIndicator)
	s.ErrorMessage = plainStyle(s.ErrorMessage)
	s.SelectSelector = plainStyle(s.SelectSelector)
	s.Option = plainStyle(s.Option)
	s.NextIndicator = plainStyle(s.NextIndicator)
	s.PrevIndicator = plainStyle(s.PrevIndicator)
	s.Directory = plainStyle(s.Directory)
	s.File = plainStyle(s.File)
	s.MultiSelectSelector = plainStyle(s.MultiSelectSelector)
	s.SelectedOption = plainStyle(s.SelectedOption)
	s.SelectedPrefix = plainStyle(s.SelectedPrefix)
	s.UnselectedOption = plainStyle(s.UnselectedOption)
	s.UnselectedPrefix = plainStyle(s.UnselectedPrefix)
	s.TextInput.Cursor = plainStyle(s.TextInput.Cursor)
	s.TextInput.CursorText = plainStyle(s.TextInput.CursorText)
	s.TextInput.Placeholder = plainStyle(s.TextInput.Placeholder)
	s.TextInput.Prompt = plainStyle(s.TextInput.Prompt)
	s.TextInput.Text = plainStyle(s.TextInput.Text)
	s.FocusedButton = plainStyle(s.FocusedButton)
	s.BlurredButton = plainStyle(s.BlurredButton)
	s.Card = plainStyle(s.Card)
	s.NoteTitle = plainStyle(s.NoteTitle)
	s.Next = plainStyle(s.Next)
	return s
}

func plainStyle(s lipgloss.Style) lipgloss.Style {
	return s.
		UnsetForeground().
		UnsetBackground().
		UnsetBold().
		UnsetItalic().
		UnsetUnderline().
		UnsetStrikethrough().
		UnsetReverse().
		UnsetBlink().
		UnsetFaint()
}
