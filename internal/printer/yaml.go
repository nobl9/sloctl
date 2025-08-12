package printer

import (
	"io"

	"github.com/nobl9/sloctl/internal/yamlenc"
)

type yamlPrinter struct {
	out io.Writer
}

func (p *yamlPrinter) Print(content any) error {
	enc := yamlenc.NewEncoder(p.out)
	return enc.Encode(content)
}
