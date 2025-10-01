package form

import (
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/huh"
)

const accesibleModeEnv = "SLOCTL_ACCESSIBLE_MODE"

var defaultTheme = huh.ThemeBase16()

func New(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).
		WithTheme(defaultTheme).
		WithAccessible(getAccesibleEnvValue())
}

func getAccesibleEnvValue() bool {
	v, ok := os.LookupEnv(accesibleModeEnv)
	if !ok {
		return false
	}
	accesible, err := strconv.ParseBool(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid value: %q for %q environment variable. Error: %v", v, accesibleModeEnv, err)
	}
	return accesible
}
