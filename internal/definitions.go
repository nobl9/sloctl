package sloctl

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const filesPromptPattern = "You're applying more than %d files (%d). Do you want to continue? (y/n): "

// readObjectsDefinitions reads object definitions from the provided definition paths.
// Empty definition path or '-' are treated as input from os.Stdin.
func readObjectsDefinitions(
	config *sdk.Config,
	cmd *cobra.Command,
	definitionPaths []string,
	prompt filesPrompt,
	projectFlagWasSet bool,
) ([]manifest.Object, error) {
	containsStdin := false
	for i := range definitionPaths {
		if definitionPaths[i] == "" || definitionPaths[i] == "-" {
			definitionPaths = append(definitionPaths[:i], definitionPaths[i+1:]...)
			containsStdin = true
		}
	}
	sources, err := sdk.ResolveObjectSources(definitionPaths...)
	if err != nil {
		return nil, err
	}
	if err = runPrompt(cmd, prompt, sources); err != nil {
		return nil, err
	}
	if containsStdin {
		sources = append(sources, sdk.NewObjectSourceReader(cmd.InOrStdin(), "stdin"))
	}
	defs, err := sdk.ReadObjectsFromSources(cmd.Context(), sources...)
	if err != nil {
		return nil, err
	}
	defs = manifest.SetDefaultProject(defs, config.Project)
	if !projectFlagWasSet {
		return defs, nil
	}
	// Make sure the --project flag matches all the parsed definitions projects.
	for i := range defs {
		obj, isProjectScoped := defs[i].(manifest.ProjectScopedObject)
		// Since v1alpha.ObjectGeneric fulfills manifest.ProjectScopedObject
		if !isProjectScoped || obj.GetProject() == "" {
			continue
		}
		if obj.GetProject() != config.Project {
			return nil, errors.Errorf(
				"The %[1]s project from the provided object %[2]s.%[1]s does not match "+
					"the project '%[3]s'. You must pass '--project=%[1]s' to perform this operation or"+
					" allow the Project to be inferred from the object definition.",
				obj.GetProject(), obj.GetName(), config.Project)
		}
	}
	return defs, nil
}

func runPrompt(cmd *cobra.Command, prompt filesPrompt, sources []*sdk.ObjectSource) error {
	if !prompt.Enabled || prompt.AutoConfirm {
		return nil
	}
	resolvedPathsCount := 0
	for i := range sources {
		if sources[i].Type == sdk.ObjectSourceTypeDirectory ||
			sources[i].Type == sdk.ObjectSourceTypeGlobPattern {
			resolvedPathsCount += len(sources[i].Paths)
		}
	}
	if resolvedPathsCount > 0 && resolvedPathsCount > prompt.Threshold {
		cmd.Printf(filesPromptPattern, prompt.Threshold, resolvedPathsCount)
		return prompt.Prompt()
	}
	return nil
}

func newFilesPrompt(enabled, autoConfirm bool, threshold int) filesPrompt {
	return filesPrompt{
		Enabled:     enabled,
		AutoConfirm: autoConfirm,
		Threshold:   threshold,
		ReadFrom:    os.Stdin,
	}
}

type filesPrompt struct {
	Enabled     bool
	AutoConfirm bool
	Threshold   int
	ReadFrom    io.Reader
}

var errOperationAborted = errors.New("operation aborted")

func (f filesPrompt) Prompt() error {
	var choice string
	if _, err := fmt.Fscanln(f.ReadFrom, &choice); err != nil {
		// When a single '\n' is provided, fmt.ScanState.SkipSpace()
		// implementation will return error "unexpected newline".
		if errors.Is(err, io.EOF) || err.Error() == "unexpected newline" {
			return errOperationAborted
		}
		return errors.Wrap(err, "failed to read confirmation from stdin")
	}
	switch strings.ToLower(choice) {
	case "y", "yes":
		return nil
	default:
		return errOperationAborted
	}
}
