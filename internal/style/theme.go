// Package style defines shared sloctl terminal styles.
package style

import (
	"image/color"
	"os"

	huh "charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour/ansi"
	glamourstyles "github.com/charmbracelet/glamour/styles"
)

const (
	blue900Hex   = "#01465C"
	blue800Hex   = "#00819E"
	blue600Hex   = "#00BAD3"
	blue200Hex   = "#E5F9FD"
	gray850Hex   = "#676868"
	mutedHex     = "#BABBBB"
	lightGrayHex = "#E8E9E9"
	darkGrayHex  = "#383939"
	blackHex     = "#000000"
	whiteHex     = "#FFFFFF"
	pinkHex      = "#DB2779"
	redHex       = "#D42E56"
	noColorEnv   = "NO_COLOR"
)

var (
	darkGray        = lipgloss.Color(darkGrayHex)
	white           = lipgloss.Color(whiteHex)
	pink            = lipgloss.Color(pinkHex)
	red             = lipgloss.Color(redHex)
	darkModePalette = terminalPalette{
		accentHex:             blue600Hex,
		headingHex:            blue600Hex,
		textHex:               lightGrayHex,
		mutedHex:              mutedHex,
		strongHex:             whiteHex,
		optionHex:             whiteHex,
		selectedForegroundHex: whiteHex,
		selectedBackgroundHex: blue900Hex,
		codeForegroundHex:     whiteHex,
		codeBackgroundHex:     blue900Hex,
	}
	lightModePalette = terminalPalette{
		accentHex:             blue800Hex,
		headingHex:            blue800Hex,
		textHex:               gray850Hex,
		mutedHex:              gray850Hex,
		strongHex:             blackHex,
		optionHex:             blackHex,
		selectedForegroundHex: whiteHex,
		selectedBackgroundHex: blue800Hex,
		codeForegroundHex:     blue800Hex,
		codeBackgroundHex:     blue200Hex,
	}
)

type terminalPalette struct {
	accentHex             string
	headingHex            string
	textHex               string
	mutedHex              string
	strongHex             string
	optionHex             string
	selectedForegroundHex string
	selectedBackgroundHex string
	codeForegroundHex     string
	codeBackgroundHex     string
}

func terminalPaletteFor(isDark bool) terminalPalette {
	if isDark {
		return darkModePalette
	}
	return lightModePalette
}

func (p terminalPalette) accent() color.Color {
	return lipgloss.Color(p.accentHex)
}

func (p terminalPalette) muted() color.Color {
	return lipgloss.Color(p.mutedHex)
}

func (p terminalPalette) strong() color.Color {
	return lipgloss.Color(p.strongHex)
}

func (p terminalPalette) option() color.Color {
	return lipgloss.Color(p.optionHex)
}

func (p terminalPalette) selectedForeground() color.Color {
	return lipgloss.Color(p.selectedForegroundHex)
}

func (p terminalPalette) selectedBackground() color.Color {
	return lipgloss.Color(p.selectedBackgroundHex)
}

// HuhTheme returns the shared Nobl9 terminal theme for interactive forms.
func HuhTheme(isDark bool) *huh.Styles {
	t := huh.ThemeBase(isDark)
	if !ColorEnabled() {
		return plainHuhTheme(t)
	}

	p := terminalPaletteFor(isDark)
	t.Focused.Base = t.Focused.Base.BorderForeground(p.accent())
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(p.accent())
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(p.accent())
	t.Focused.Directory = t.Focused.Directory.Foreground(p.accent())
	t.Focused.Description = t.Focused.Description.Foreground(p.muted())
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(red)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(red)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(p.accent())
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(p.accent())
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(p.accent())
	t.Focused.Option = t.Focused.Option.Foreground(p.option()).UnsetFaint()
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(p.accent())
	t.Focused.SelectedOption = t.Focused.SelectedOption.
		Foreground(p.selectedForeground()).
		Background(p.selectedBackground()).
		UnsetFaint().
		Bold(true)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(p.accent())
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(p.option()).UnsetFaint()
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(p.option()).UnsetFaint()
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(white).Background(pink)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(white).Background(darkGray)

	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(pink)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(p.muted())
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(p.accent())

	t.Help.ShortKey = t.Help.ShortKey.Foreground(p.accent())
	t.Help.FullKey = t.Help.FullKey.Foreground(p.accent())
	t.Help.ShortDesc = t.Help.ShortDesc.Foreground(p.muted())
	t.Help.FullDesc = t.Help.FullDesc.Foreground(p.muted())
	t.Help.ShortSeparator = t.Help.ShortSeparator.Foreground(p.muted())
	t.Help.FullSeparator = t.Help.FullSeparator.Foreground(p.muted())

	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NoteTitle = t.Blurred.NoteTitle.Foreground(p.muted())
	t.Blurred.Title = t.Blurred.Title.Foreground(p.muted())

	t.Blurred.TextInput.Prompt = t.Blurred.TextInput.Prompt.Foreground(p.muted())
	t.Blurred.TextInput.Text = t.Blurred.TextInput.Text.Foreground(p.strong())

	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description

	return t
}

// NotificationTitle returns the shared style for notification titles.
func NotificationTitle(isDark bool) lipgloss.Style {
	if !ColorEnabled() {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(terminalPaletteFor(isDark).strong()).Bold(true)
}

// NotificationLink returns the shared style for notification links.
func NotificationLink(isDark bool) lipgloss.Style {
	if !ColorEnabled() {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(terminalPaletteFor(isDark).accent()).Underline(true)
}

// NotificationLabel returns the shared style for notification labels.
func NotificationLabel(isDark bool) lipgloss.Style {
	if !ColorEnabled() {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(terminalPaletteFor(isDark).muted())
}

// NotificationSeparator returns the shared style for notification dividers.
func NotificationSeparator(isDark bool) lipgloss.Style {
	if !ColorEnabled() {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(terminalPaletteFor(isDark).muted())
}

// ColorEnabled reports whether terminal styles should emit ANSI color.
func ColorEnabled() bool {
	return os.Getenv(noColorEnv) == ""
}

// NotificationMarkdownStyle returns the shared Glamour style for notifications.
func NotificationMarkdownStyle(isDark bool) ansi.StyleConfig {
	if !ColorEnabled() {
		return notificationASCIIMarkdownStyle()
	}
	p := terminalPaletteFor(isDark)
	styleConfig := glamourstyles.LightStyleConfig
	if isDark {
		styleConfig = glamourstyles.DarkStyleConfig
	}
	styleConfig.Document.Margin = nil
	styleConfig.Document.Color = new(p.textHex)
	styleConfig.Paragraph.Color = new(p.textHex)
	styleConfig.BlockQuote.Color = new(p.textHex)
	for _, heading := range []*ansi.StyleBlock{
		&styleConfig.Heading,
		&styleConfig.H2,
		&styleConfig.H3,
		&styleConfig.H4,
		&styleConfig.H5,
		&styleConfig.H6,
	} {
		setMarkdownHeading(heading, p.headingHex)
	}
	setMarkdownHeading(&styleConfig.H1, p.selectedForegroundHex)
	styleConfig.H1.BackgroundColor = new(p.selectedBackgroundHex)
	styleConfig.Strong.Color = new(p.strongHex)
	styleConfig.Strong.Bold = new(true)
	styleConfig.Item.Color = new(p.textHex)
	styleConfig.Enumeration.Color = new(p.textHex)
	styleConfig.Link.Color = new(p.accentHex)
	styleConfig.Link.Underline = new(true)
	styleConfig.LinkText.Color = new(p.accentHex)
	styleConfig.LinkText.Underline = new(true)
	styleConfig.HorizontalRule.Color = new(p.mutedHex)
	styleConfig.Code.Color = new(p.codeForegroundHex)
	styleConfig.Code.BackgroundColor = new(p.codeBackgroundHex)
	return styleConfig
}

func notificationASCIIMarkdownStyle() ansi.StyleConfig {
	styleConfig := glamourstyles.ASCIIStyleConfig
	styleConfig.Document.Margin = nil
	return styleConfig
}

func setMarkdownHeading(heading *ansi.StyleBlock, colorHex string) {
	heading.Color = new(colorHex)
	heading.Bold = new(true)
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
