package printer

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
)

type jsonPrinter struct {
	out io.Writer
}

func (p jsonPrinter) Print(content any) error {
	switch v := content.(type) {
	case []manifest.Object:
		return sdk.EncodeObjects(v, p.out, manifest.ObjectFormatJSON)
	default:
		if str, ok := p.jsonScalarToString(content); ok {
			_, err := fmt.Fprintln(p.out, str)
			if err != nil {
				return err
			}
		} else {
			enc := json.NewEncoder(p.out)
			enc.SetIndent("", "  ")
			if err := enc.Encode(content); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p jsonPrinter) jsonScalarToString(input any) (string, bool) {
	switch v := input.(type) {
	case string:
		return v, true
	case int:
		return strconv.Itoa(v), true
	case float64:
		if math.Trunc(v) == v {
			return strconv.FormatFloat(v, 'f', 0, 64), true
		} else {
			return strconv.FormatFloat(v, 'f', -1, 64), true
		}
	case nil:
		return "null", true
	case bool:
		return fmt.Sprintf("%v", v), true
	default:
		return "", false
	}
}
