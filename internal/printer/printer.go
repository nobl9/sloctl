// Package printer provides utilities for printing standard structures from api in convenient formats
package printer

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/nobl9/go-yaml"
	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"

	"github.com/nobl9/sloctl/internal/csv"
)

// Format represents supported printing outputs
type Format string

// All supported output formats by a Printer
const (
	YAMLFormat Format = "yaml"
	JSONFormat Format = "json"
	CSVFormat  Format = "csv"
)

// Printer represents generic printer for cli
type Printer interface {
	Print(interface{}) error
}

// New returns an instance of a proper printer based on format parameter
func New(out io.Writer, format Format, fieldSeparator, recordSeparator string) (Printer, error) {
	switch format {
	case JSONFormat:
		return &jsonPrinter{Out: out}, nil
	case YAMLFormat:
		return &yamlPrinter{Out: out}, nil
	case CSVFormat:
		return &csvPrinter{Out: out, fieldSeparator: fieldSeparator, recordSeparator: recordSeparator}, nil
	default:
		return nil, fmt.Errorf("unknown output format %q", format)
	}
}

type jsonPrinter struct {
	Out io.Writer
}

func (p *jsonPrinter) Print(content interface{}) error {
	switch v := content.(type) {
	case []manifest.Object:
		return sdk.PrintObjects(v, p.Out, manifest.ObjectFormatJSON)
	default:
		b, err := json.MarshalIndent(content, "", "  ")
		if err != nil {
			return err
		}
		_, err = p.Out.Write(b)
		return err
	}
}

type yamlPrinter struct {
	Out io.Writer
}

func (p *yamlPrinter) Print(content interface{}) error {
	switch v := content.(type) {
	case []manifest.Object:
		return sdk.PrintObjects(v, p.Out, manifest.ObjectFormatYAML)
	default:
		b, err := yaml.Marshal(content)
		if err != nil {
			return err
		}
		_, err = p.Out.Write(b)
		return err
	}
}

type csvPrinter struct {
	Out             io.Writer
	fieldSeparator  string
	recordSeparator string
}

func (p *csvPrinter) Print(content interface{}) error {
	b, err := csv.Marshal(content, p.fieldSeparator, p.recordSeparator)
	if err != nil {
		return err
	}
	_, err = p.Out.Write(b)
	return err
}
