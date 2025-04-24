// Package printer provides utilities for printing standard structures from api in convenient formats.
package printer

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/spf13/cobra"

	"github.com/nobl9/sloctl/internal/csv"
)

type Config struct {
	Output             io.Writer
	OutputFormat       Format
	CSVFieldSeparator  string
	CSVRecordSeparator string
}

func NewPrinter(config Config) *Printer {
	if config.Output == nil {
		config.Output = os.Stdout
	}
	if config.OutputFormat == "" {
		config.OutputFormat = YAMLFormat
	}
	return &Printer{config: config}
}

type Printer struct {
	config Config
}

func (o *Printer) Print(v any) error {
	p, err := newPrinter(o.config.Output, o.config.OutputFormat, o.config.CSVFieldSeparator, o.config.CSVRecordSeparator)
	if err != nil {
		return err
	}
	if err = p.Print(v); err != nil {
		return err
	}
	return nil
}

func (o *Printer) MustRegisterFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().VarP(
		&o.config.OutputFormat,
		"output",
		"o",
		`Output format: one of yaml|json|csv.`,
	)

	cmd.PersistentFlags().StringVarP(
		&o.config.CSVFieldSeparator,
		csv.FieldSeparatorFlag,
		"",
		csv.DefaultFieldSeparator,
		"Field Separator for CSV.",
	)
	if err := cmd.PersistentFlags().MarkHidden(csv.FieldSeparatorFlag); err != nil {
		panic(err)
	}

	cmd.PersistentFlags().StringVarP(
		&o.config.CSVRecordSeparator,
		csv.RecordSeparatorFlag,
		"",
		csv.DefaultRecordSeparator,
		"Record Separator for CSV.",
	)
	if err := cmd.PersistentFlags().MarkHidden(csv.RecordSeparatorFlag); err != nil {
		panic(err)
	}
}

// All supported output formats by [Printer].
const (
	YAMLFormat Format = "yaml"
	JSONFormat Format = "json"
	CSVFormat  Format = "csv"
)

// Format represents supported printing outputs.
type Format string

func (f *Format) String() string {
	return string(*f)
}

func (f *Format) Set(value string) error {
	switch value {
	case "yaml", "json", "csv":
		*f = Format(value)
		return nil
	default:
		return fmt.Errorf("invalid value for Format: %s", value)
	}
}

func (f *Format) Type() string {
	return "format"
}

// printerInterface represents generic printer for cli
type printerInterface interface {
	Print(interface{}) error
}

// newPrinter returns an instance of a proper [printerInterface] based on format parameter
func newPrinter(out io.Writer, format Format, fieldSeparator, recordSeparator string) (printerInterface, error) {
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
