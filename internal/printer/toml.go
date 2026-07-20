package printer

import (
	"io"

	"github.com/BurntSushi/toml"
	"github.com/nobl9/nobl9-go/manifest"
	"github.com/pkg/errors"
)

type tomlPrinter struct {
	out io.Writer
}

func (p *tomlPrinter) Print(content any) error {
	if content == nil {
		// The encoder panics when presented with nil.
		// However, TOML does not have a notion of null values,
		// so we're just printing a new, empty line here.
		_, err := p.out.Write([]byte("\n"))
		return err
	}
	switch v := content.(type) {
	case []manifest.Object:
		return errors.Errorf("TOML encoder does not support %T", v)
	default:
		enc := toml.NewEncoder(p.out)
		enc.Indent = "  "
		if err := enc.Encode(content); err != nil {
			return err
		}
	}
	return nil
}
