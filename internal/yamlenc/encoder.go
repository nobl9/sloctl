package yamlenc

import (
	"encoding/json"
	"io"

	"github.com/goccy/go-yaml"
)

func NewEncoder(writer io.Writer) *yaml.Encoder {
	return yaml.NewEncoder(writer,
		yaml.IndentSequence(true),
		yaml.UseLiteralStyleIfMultiline(true),
		yaml.CustomMarshaler(yamlNumberMarshaler))
}

// yamlNumberMarshaler is a custom marshaler for [json.Number].
// It is used to avoid converting int to float64 when converting JSON to YAML for generic
// [manifest.Object] representations, like [v1alpha.GenericObject].
func yamlNumberMarshaler(number json.Number) ([]byte, error) {
	return []byte(number.String()), nil
}
