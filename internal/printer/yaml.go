package printer

import (
	"io"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/nobl9/sloctl/internal/yamlenc"
)

type yamlPrinter struct {
	out io.Writer
}

func (p *yamlPrinter) Print(content any) error {
	switch v := content.(type) {
	case []manifest.Object:
		return sdk.PrintObjects(v, p.out, manifest.ObjectFormatYAML)
	default:
		enc := yamlenc.NewEncoder(p.out)
		return enc.Encode(content)
	}
}
