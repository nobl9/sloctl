package form

import (
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/huh"
)

const accessibleModeEnv = "SLOCTL_ACCESSIBLE_MODE"

var defaultTheme = huh.ThemeBase16()

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
