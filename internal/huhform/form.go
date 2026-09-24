package huhform

import (
	"fmt"
	"os"
	"strconv"

	huh "charm.land/huh/v2"

	"github.com/nobl9/sloctl/internal/style"
)

// accessibleModeEnv can be set to turn on [huh] accessible mode.
// It can be useful in old terminal emulators (e.g. remote shells).
const accessibleModeEnv = "SLOCTL_ACCESSIBLE_MODE"

// New returns a form configured with sloctl's shared terminal theme.
func New(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).
		WithTheme(huh.ThemeFunc(style.HuhTheme)).
		WithAccessible(AccessibleMode())
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
