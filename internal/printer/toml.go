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
