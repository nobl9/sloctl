// Sloctl provides a single binary command-line tool to interact with N9 application.
// It is heavily inspired by kubectl and follows its user experience and conventions.
// It uses the .config.toml configuration file and looks for it in $HOME/.config/nobl9.
// Path to this file can be overwritten by passing flag --config CONFIG_FILE_PATH
// example configuration file can be found in this repository samples/config.toml.
package main

import "github.com/nobl9/sloctl/internal"

func main() {
	internal.Execute()
}
