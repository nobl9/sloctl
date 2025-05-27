package printer

import (
	"encoding/json"
	"io"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
)

type jsonPrinter struct {
	out io.Writer
}

func (p *jsonPrinter) Print(content any) error {
	switch v := content.(type) {
	case []manifest.Object:
		return sdk.PrintObjects(v, p.out, manifest.ObjectFormatJSON)
	default:
		b, err := json.MarshalIndent(content, "", "  ")
		if err != nil {
			return err
		}
		_, err = p.out.Write(b)
		return err
	}
}
