package huhform

import (
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// accessibleModeEnv can be set to turn on [huh] accessible mode.
// It can be useful in old terminal emulators (e.g. remote shells).
const accessibleModeEnv = "SLOCTL_ACCESSIBLE_MODE"

var defaultTheme = themeNobl9()

func New(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).
		WithTheme(defaultTheme).
		WithAccessible(getAccessibleEnvValue())
}

func getAccessibleEnvValue() bool {
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

// themeNobl9 returns a new theme based on the Nobl9 color scheme.
func themeNobl9() *huh.Theme {
	t := huh.ThemeBase()

	var (
		black  = lipgloss.Color("#383939")
		green  = lipgloss.Color("#0EB46E")
		yellow = lipgloss.Color("#C8D655")
		pink   = lipgloss.Color("#DB2779")
		blue   = lipgloss.Color("#63D6E5")
		gray   = lipgloss.Color("#989999")
		red    = lipgloss.Color("#D42E56")
		white  = lipgloss.Color("#FFFFFF")
	)

	t.Focused.Base = t.Focused.Base.BorderForeground(gray)
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(blue)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(blue)
	t.Focused.Directory = t.Focused.Directory.Foreground(blue)
	t.Focused.Description = t.Focused.Description.Foreground(gray)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(red)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(red)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(yellow)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(yellow)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(yellow)
	t.Focused.Option = t.Focused.Option.Foreground(white)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(yellow)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(green)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(green)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(white)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(white).Background(pink)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(white).Background(black)

	t.Focused.TextInput.Cursor.Foreground(pink)
	t.Focused.TextInput.Placeholder.Foreground(gray)
	t.Focused.TextInput.Prompt.Foreground(yellow)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NoteTitle = t.Blurred.NoteTitle.Foreground(gray)
	t.Blurred.Title = t.Blurred.NoteTitle.Foreground(gray)

	t.Blurred.TextInput.Prompt = t.Blurred.TextInput.Prompt.Foreground(gray)
	t.Blurred.TextInput.Text = t.Blurred.TextInput.Text.Foreground(white)

	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description

	return t
}
