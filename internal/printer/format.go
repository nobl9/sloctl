package printer

import "fmt"

// All supported output formats by [Printer].
const (
	YAMLFormat Format = "yaml"
	JSONFormat Format = "json"
	CSVFormat  Format = "csv"
	TOMLFormat Format = "toml"
)

// ObjectsSupportedFormats lists [Format] which supports encoding [manifest.Object].
var ObjectsSupportedFormats = []Format{
	YAMLFormat,
	JSONFormat,
	CSVFormat,
}

// Format represents supported printing outputs.
type Format string

func (f *Format) String() string {
	return string(*f)
}

// Set implements [pflag.Value] interface.
func (f *Format) Set(value string) error {
	switch value {
	case "yaml", "json", "csv", "toml":
		*f = Format(value)
		return nil
	default:
		return errInvalidFormat(value)
	}
}

func (f *Format) Type() string {
	return "format"
}

func errInvalidFormat[T ~string](value T) error {
	return fmt.Errorf("invalid value for Format: %s", value)
}
