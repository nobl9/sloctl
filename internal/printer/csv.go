package printer

import (
	"io"

	"github.com/nobl9/sloctl/internal/csv"
)

type csvPrinter struct {
	out             io.Writer
	fieldSeparator  string
	recordSeparator string
}

func (p *csvPrinter) Print(content any) error {
	b, err := csv.Marshal(content, p.fieldSeparator, p.recordSeparator)
	if err != nil {
		return err
	}
	_, err = p.out.Write(b)
	return err
}
