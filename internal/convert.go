package internal

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/OpenSLO/go-sdk/pkg/openslo"
	"github.com/OpenSLO/go-sdk/pkg/openslosdk"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tidwall/sjson"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/nobl9/nobl9-openslo/pkg/openslotonobl9"

	"github.com/nobl9/sloctl/internal/printer"
)

//go:embed convert_openslo_example.sh
var convertOpenSLOExample string

type ConvertCmd struct {
	printer         *printer.Printer
	definitionPaths []string
}

func (r *RootCmd) NewConvertCmd() *cobra.Command {
	convert := ConvertCmd{
		printer: printer.NewPrinter(printer.Config{}),
	}

	cmd := &cobra.Command{
		Use:   "convert",
		Short: "Convert SLO definitions to Nobl9 configuration",
		Long:  `Converts external SLO (and more!) definitions to Nobl9 YAML configuration.`,
	}

	cmd.AddCommand(convert.newConvertOpenSLOCommand())

	return cmd
}

func (c ConvertCmd) newConvertOpenSLOCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "openslo",
		Short:   "Convert OpenSLO specification to Nobl9 configuration",
		Long:    "To learn more about how the conversion works, visit https://github.com/nobl9/nobl9-openslo.",
		Example: convertOpenSLOExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.convertOpenSLO(cmd)
		},
	}

	c.printer.MustRegisterFlags(cmd)
	registerFileFlag(cmd, true, &c.definitionPaths)

	return cmd
}

func (c ConvertCmd) convertOpenSLO(cmd *cobra.Command) error {
	if len(c.definitionPaths) == 0 {
		return cmd.Usage()
	}
	objects, err := c.convertOpenSLODefinitions(cmd)
	if err != nil {
		return err
	}
	printSourcesDetails("Converted", objects, os.Stderr)
	fmt.Fprintln(os.Stderr, "---")
	return c.printer.Print(objects)
}

func (c ConvertCmd) convertOpenSLODefinitions(cmd *cobra.Command) ([]manifest.Object, error) {
	containsStdin := false
	filteredPaths := make([]string, 0, len(c.definitionPaths))
	for _, path := range c.definitionPaths {
		if path == "" || path == "-" {
			containsStdin = true
			continue
		}
		filteredPaths = append(filteredPaths, path)
	}
	c.definitionPaths = filteredPaths
	sources, err := sdk.ResolveObjectSources(c.definitionPaths...)
	if err != nil {
		return nil, err
	}
	if containsStdin {
		sources = append(sources, sdk.NewObjectSourceReader(cmd.InOrStdin(), "stdin"))
	}

	definitions, err := sdk.ReadRawDefinitionsFromSources(cmd.Context(), sources...)
	if err != nil {
		return nil, err
	}
	definitions, err = c.filterOpenSLORawDefinitions(definitions)
	if err != nil {
		return nil, err
	}

	opensloObjects := make([]openslo.Object, 0, len(definitions))
	for _, def := range definitions {
		objects, err := c.readOpenSLODefinitionsFromSource(def)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", def.ResolvedSource, err)
		}
		opensloObjects = append(opensloObjects, objects...)
	}

	nobl9Objects, err := openslotonobl9.Convert(opensloObjects)
	if err != nil {
		return nil, err
	}

	return nobl9Objects, nil
}

func (c ConvertCmd) readOpenSLODefinitionsFromSource(def *sdk.RawDefinition) ([]openslo.Object, error) {
	format := openslosdk.FormatYAML
	if isJSONBuffer(def.Definition) {
		format = openslosdk.FormatJSON
	}
	objects, err := openslosdk.Decode(bytes.NewReader(def.Definition), format)
	if err != nil {
		return nil, err
	}
	for i, obj := range objects {
		data, err := json.Marshal(obj)
		if err != nil {
			return nil, errors.Wrap(err, "failed to encode OpenSLO object")
		}
		data, err = c.setManifestSourceForOpenSLOObject(
			data,
			"metadata.annotations",
			def.ResolvedSource,
		)
		if err != nil {
			return nil, err
		}
		o, err := openslosdk.Decode(bytes.NewReader(data), openslosdk.FormatJSON)
		if err != nil || len(o) != 1 {
			return nil, errors.Wrap(err, "failed to decode intermediate OpenSLO object JSON representation")
		}
		objects[i] = o[0]
	}
	return objects, nil
}

func (c ConvertCmd) setManifestSourceForOpenSLOObject(object []byte, basePath, src string) ([]byte, error) {
	object, err := sjson.SetBytes(
		object,
		fmt.Sprintf("%s.%s/manifestSrc", basePath, strings.ReplaceAll(openslotonobl9.DomainNobl9, ".", "\\.")),
		src,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to set manifestSrc annotation for OpenSLO object")
	}
	return object, nil
}

var jsonBufferRegex = regexp.MustCompile(`^\s*\[?\s*{`)

// isJSONBuffer scans the provided buffer, looking for an open brace indicating this is JSON.
// While a simple list like ["a", "b", "c"] is still a valid JSON,
// it does not really concern us when processing complex objects.
func isJSONBuffer(buf []byte) bool {
	return jsonBufferRegex.Match(buf)
}

var opensloAPIVersionRegex = regexp.MustCompile(`"?apiVersion"?\s*:\s*"?openslo`)

func (c ConvertCmd) filterOpenSLORawDefinitions(definitions []*sdk.RawDefinition) ([]*sdk.RawDefinition, error) {
	filtered := make([]*sdk.RawDefinition, 0, len(definitions))
	for _, def := range definitions {
		// nolint: exhaustive
		switch def.SourceType {
		case sdk.ObjectSourceTypeFile:
			if !opensloAPIVersionRegex.Match(def.Definition) {
				return nil, sdk.ErrInvalidFile
			}
		case sdk.ObjectSourceTypeDirectory, sdk.ObjectSourceTypeGlobPattern:
			if !opensloAPIVersionRegex.Match(def.Definition) {
				continue
			}
		}
		filtered = append(filtered, def)
	}
	return filtered, nil
}
