package internal

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
)

// ref: https://github.com/spf13/cobra/issues/1466
// Ways to prevent shell glob expansion:
//
//   - quote it:
//     `$ sloctl apply -f '*'`
//
//   - escape it:
//     `$ sloctl apply -f \*`
//
//   - disable the glob expansion
//     `$ set -f`
//     or
//     `$ set -o noglob`
func positionalArgsCondition(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	return fmt.Errorf("command accepts 0 args, received %d, make sure you're quoting"+
		" the glob pattern to prevent shell from doing it for you", len(args))
}

func printSourcesDetails(verb string, objects []manifest.Object) {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s %d objects from the following sources: \n", verb, len(objects)))
	uniq := make(map[string]struct{}, len(objects)/2) // Rough estimation of the objects from provided sources.
	sort.SliceStable(objects, func(i, j int) bool {
		return objects[j].GetManifestSource() > objects[i].GetManifestSource()
	})
	for i := range objects {
		src := objects[i].GetManifestSource()
		if _, ok := uniq[src]; ok {
			continue
		}
		uniq[src] = struct{}{}
		b.WriteString(" - ")
		b.WriteString(src)
		b.WriteString("\n")
	}
	_, isStdin := uniq["stdin"]
	if len(uniq) == 1 && isStdin {
		return
	}
	fmt.Print(b.String())
}

func printCommandResult(message string, dryRun bool) {
	if dryRun {
		message += " (dry run)"
	}
	fmt.Println(message)
}

//go:embed apply_or_delete_description.tpl
var applyOrDeleteDescription string

func getApplyOrDeleteDescription(description string) string {
	tpl, err := template.New("applyOrDeleteDescription").Parse(applyOrDeleteDescription)
	if err != nil {
		panic(err)
	}
	extensionsBuilder := strings.Builder{}
	extensions := sdk.GetSupportedFileExtensions()
	for i, ext := range extensions {
		extensionsBuilder.WriteString("'" + ext + "'")
		if i == len(extensions)-1 {
			break
		}
		if i == len(extensions)-2 {
			extensionsBuilder.WriteString(" and ")
			continue
		}
		extensionsBuilder.WriteString(", ")
	}
	var b strings.Builder
	if err = tpl.Execute(&b, struct {
		Description string
		Extensions  string
		Regex       string
	}{
		Description: description,
		Extensions:  extensionsBuilder.String(),
		Regex:       sdk.APIVersionRegex,
	}); err != nil {
		panic(err)
	}
	return b.String()
}
