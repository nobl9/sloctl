package printer

import (
	"io"

	"github.com/nobl9/sloctl/internal/yamlenc"
)

type yamlPrinter struct {
	out io.Writer
}

func (p yamlPrinter) Print(content any) error {
	switch v := content.(type) {
	case []manifest.Object:
		return sdk.EncodeObjects(v, p.out, manifest.ObjectFormatYAML)
	default:
		b, err := yaml.Marshal(content)
		if err != nil {
			return err
		}
		_, err = p.out.Write(b)
		return err
	}
}
