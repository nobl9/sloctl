package printer

import (
	"io"

	"github.com/goccy/go-yaml"
	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
)

type yamlPrinter struct {
	out io.Writer
}

func (p *yamlPrinter) Print(content any) error {
	switch v := content.(type) {
	case []manifest.Object:
		return sdk.PrintObjects(v, p.out, manifest.ObjectFormatYAML)
	default:
		b, err := yaml.Marshal(content)
		if err != nil {
			return err
		}
		_, err = p.out.Write(b)
		return err
	}
}
