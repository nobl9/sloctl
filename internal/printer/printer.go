// Package printer provides utilities for printing standard structures from api in convenient formats.
package printer

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/nobl9/sloctl/internal/csv"
	"github.com/nobl9/sloctl/internal/jq"
)

type Config struct {
	Output             io.Writer
	OutputFormat       Format
	CSVFieldSeparator  string
	CSVRecordSeparator string
	SupportedFromats   []Format
}

func NewPrinter(config Config) *Printer {
	if len(config.SupportedFromats) == 0 {
		config.SupportedFromats = ObjectsSupportedFormats
	}
	if config.Output == nil {
		config.Output = os.Stdout
	}
	if config.OutputFormat == "" {
		config.OutputFormat = YAMLFormat
	}
	if config.CSVFieldSeparator == "" {
		config.CSVFieldSeparator = csv.DefaultFieldSeparator
	}
	if config.CSVRecordSeparator == "" {
		config.CSVRecordSeparator = csv.DefaultRecordSeparator
	}
	printer := &Printer{config: config}
	printer.jq = jq.NewExpressionRunner(jq.Config{})
	return printer
}

type Printer struct {
	config Config
	jq     *jq.ExpressionRunner
}

// Validate should be called before [Printer.Print], after the flag values are asigned.
func (o *Printer) Validate() error {
	if !slices.Contains(o.config.SupportedFromats, o.config.OutputFormat) {
		return errInvalidFormat(o.config.OutputFormat)
	}
	return nil
}

func (o *Printer) Print(v any) error {
	p, err := newPrinter(o.config.Output, o.config.OutputFormat, o.config.CSVFieldSeparator, o.config.CSVRecordSeparator)
	if err != nil {
		return err
	}
	switch {
	case v == nil:
		return nil
	case o.jq.ShouldRun():
		values, err := o.jq.Evaluate(v)
		if err != nil {
			return err
		}
		for v, err := range values {
			if err != nil {
				return err
			}
			if err = p.Print(v); err != nil {
				return err
			}
		}
	default:
		if err := p.Print(v); err != nil {
			return err
		}
	}
	return nil
}

// printerInterface represents generic printer for cli
type printerInterface interface {
	Print(any) error
}

// newPrinter returns an instance of a proper [printerInterface] based on format parameter
func newPrinter(out io.Writer, format Format, fieldSeparator, recordSeparator string) (printerInterface, error) {
	switch format {
	case JSONFormat:
		return &jsonPrinter{out: out}, nil
	case YAMLFormat:
		return &yamlPrinter{out: out}, nil
	case CSVFormat:
		return &csvPrinter{out: out, fieldSeparator: fieldSeparator, recordSeparator: recordSeparator}, nil
	case TOMLFormat:
		return &tomlPrinter{out: out}, nil
	default:
		return nil, fmt.Errorf("unknown output format %q", format)
	}
}
