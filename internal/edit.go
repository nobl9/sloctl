package internal

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
	objectsV1 "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v1"
	objectsV2 "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v2"
	"github.com/spf13/cobra"
)

type EditCmd struct {
	client            *sdk.Client
	dryRun            bool
	project           string
	projectFlagWasSet bool
}

const (
	editorEnvSloctl = "SLOCTL_EDITOR"
	editorEnvSystem = "EDITOR"
	shellEnv        = "SHELL"

	defaultEditorWindows      = "notepad"
	defaultEditorUnixVim      = "vim"
	defaultEditorUnixVi       = "vi"
	defaultEditorUnixFallback = "nano"
	defaultShellUnix          = "/bin/bash"
	defaultShellWindows       = "cmd"
	shellArgUnix              = "-c"
	shellArgWindows           = "/C"
)

//go:embed edit_example.sh
var editExample string

//go:embed edit_description.tpl
var editDescriptionTemplate string

// NewEditCmd returns cobra command edit.
func (r *RootCmd) NewEditCmd() *cobra.Command {
	edit := &EditCmd{}

	cmd := &cobra.Command{
		Use:     "edit",
		Short:   "Edit one or more than one resource",
		Long:    getEditDescription(),
		Example: editExample,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			edit.client = r.GetClient()
		},
	}
	cmd.PersistentFlags().BoolVarP(&edit.dryRun, flagDryRun, "", false,
		"Submit server-side request without persisting the configured resources.")

	for _, kind := range manifest.KindValues() {
		if !kind.Applicable() {
			continue
		}
		plural := pluralForKind(kind)
		short := fmt.Sprintf("Edits one or more than one of the %s.", plural)
		use := strings.ToLower(plural)
		aliases := append(aliasesForKind(kind), kind.ToLower(), kind.String(), plural)

		sc := edit.newEditObjectsCommand(kind, short, use, aliases)
		if objectKindSupportsProjectFlag(kind) {
			registerProjectFlag(sc, &edit.project)
		}
		cmd.AddCommand(sc)
	}

	return cmd
}

func getEditDescription() string {
	tpl, err := template.New("editDescription").Parse(editDescriptionTemplate)
	if err != nil {
		panic(err)
	}

	var b strings.Builder
	if err = tpl.Execute(&b, map[string]string{
		"EditorEnvSloctl":           editorEnvSloctl,
		"EditorEnvSystem":           editorEnvSystem,
		"ShellEnv":                  shellEnv,
		"DefaultEditorWindows":      defaultEditorWindows,
		"DefaultEditorUnixVim":      defaultEditorUnixVim,
		"DefaultEditorUnixVi":       defaultEditorUnixVi,
		"DefaultEditorUnixFallback": defaultEditorUnixFallback,
		"DefaultShellUnix":          defaultShellUnix,
		"DefaultShellWindows":       defaultShellWindows,
	}); err != nil {
		panic(err)
	}
	return b.String()
}

func (e *EditCmd) newEditObjectsCommand(
	kind manifest.Kind,
	short, use string,
	aliases []string,
) *cobra.Command {
	return &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   short,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return e.run(cmd, kind, args)
		},
	}
}

func (e *EditCmd) run(cmd *cobra.Command, kind manifest.Kind, names []string) error {
	e.projectFlagWasSet = false
	if objectKindSupportsProjectFlag(kind) {
		e.projectFlagWasSet = cmd.Flags().Changed("project")
	}

	if e.project != "" {
		if !objectKindSupportsProjectFlag(kind) {
			return fmt.Errorf("--project is not supported for %s resources", strings.ToLower(kind.String()))
		}
		e.client.Config.Project = e.project
	}

	objects, err := e.getObjects(cmd.Context(), kind, names)
	if err != nil {
		return err
	}
	if len(objects) == 0 {
		if objectKindSupportsProjectFlag(kind) {
			fmt.Printf("No resources found in '%s' project.\n", e.client.Config.Project)
			return nil
		}
		fmt.Println("No resources found.")
		return nil
	}
	if err = validateRequestedObjectsFound(names, objects); err != nil {
		return err
	}

	return e.editAndApply(cmd, objects)
}

func (e *EditCmd) getObjects(ctx context.Context, kind manifest.Kind, names []string) ([]manifest.Object, error) {
	query := url.Values{objectsV1.QueryKeyName: names}
	header := http.Header{sdk.HeaderProject: []string{e.client.Config.Project}}
	return e.client.Objects().V1().Get(ctx, kind, header, query)
}

func (e *EditCmd) editAndApply(cmd *cobra.Command, objects []manifest.Object) error {
	tempFilePath, lastEditedContents, err := writeObjectsToTemporaryFile(objects)
	if err != nil {
		return err
	}
	hadInvalidChanges := false

	keepEditedFile := false
	defer func() {
		if keepEditedFile {
			return
		}
		if removeErr := os.Remove(tempFilePath); removeErr != nil && !os.IsNotExist(removeErr) {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove temporary file %q: %v\n", tempFilePath, removeErr)
		}
	}()

	for {
		if err = runEditor(tempFilePath); err != nil {
			keepEditedFile = true
			return fmt.Errorf("%w\nA copy of your changes has been stored to %q", err, tempFilePath)
		}

		editedContents, readErr := os.ReadFile(tempFilePath)
		if readErr != nil {
			keepEditedFile = true
			return fmt.Errorf("failed to read edited definitions: %w", readErr)
		}
		if bytes.Equal(lastEditedContents, editedContents) {
			if hadInvalidChanges {
				keepEditedFile = true
				cmd.SilenceErrors = true
				cmd.PrintErrf("A copy of your changes has been stored to %q\n%s\n", tempFilePath, cancelNoValidChangesMessage)
				return errors.New("edit canceled, no valid changes were saved")
			}
			fmt.Println("Edit canceled, no changes made.")
			return nil
		}

		editedObjects, parseErr := readObjectsDefinitions(
			cmd.Context(),
			e.client.Config,
			cmd,
			[]string{tempFilePath},
			filesPrompt{},
			e.projectFlagWasSet,
		)
		if parseErr != nil {
			hadInvalidChanges = true
			lastEditedContents, err = refreshEditedFileWithError(tempFilePath, parseErr)
			if err != nil {
				keepEditedFile = true
				return err
			}
			continue
		}

		validationErr := validateEditedObjectsMatchSelection(objects, editedObjects)
		if validationErr != nil {
			hadInvalidChanges = true
			lastEditedContents, err = refreshEditedFileWithError(tempFilePath, validationErr)
			if err != nil {
				keepEditedFile = true
				return err
			}
			continue
		}

		if err = e.client.Objects().V2().Apply(cmd.Context(), objectsV2.ApplyRequest{
			Objects: editedObjects,
			DryRun:  e.dryRun,
		}); err != nil {
			keepEditedFile = true
			return fmt.Errorf("%w\nA copy of your changes has been stored to %q", err, tempFilePath)
		}
		printCommandResult("The resources were successfully applied.", e.dryRun)
		return nil
	}
}

func refreshEditedFileWithError(tempFilePath string, editErr error) ([]byte, error) {
	if writeErr := writeEditErrorToFile(tempFilePath, editErr); writeErr != nil {
		return nil, fmt.Errorf("%w\n%w\nA copy of your changes has been stored to %q", editErr, writeErr, tempFilePath)
	}

	contents, err := os.ReadFile(tempFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read edited file after writing error details: %w", err)
	}

	return contents, nil
}

const (
	editErrorHeader             = "# The edited file had a syntax error: "
	cancelNoValidChangesMessage = "error: Edit cancel" + "led, no valid changes were saved."
)

func writeEditErrorToFile(path string, editErr error) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read edited file: %w", err)
	}

	updatedContents := addEditErrorToContents(contents, editErr)

	temporaryFile, err := os.CreateTemp(filepath.Dir(path), ".sloctl-edit-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary edited file: %w", err)
	}
	defer func() {
		if removeErr := os.Remove(temporaryFile.Name()); removeErr != nil && !os.IsNotExist(removeErr) { //nolint:gosec
			fmt.Fprintf(os.Stderr, "Warning: failed to remove temporary file %q: %v\n", temporaryFile.Name(), removeErr)
		}
	}()

	if _, err = temporaryFile.Write(updatedContents); err != nil {
		if closeErr := temporaryFile.Close(); closeErr != nil {
			return fmt.Errorf(
				"failed to write edit error details to file: %w",
				errors.Join(err, fmt.Errorf("failed to close temporary edited file: %w", closeErr)),
			)
		}
		return fmt.Errorf("failed to write edit error details to file: %w", err)
	}
	if err = temporaryFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary edited file: %w", err)
	}

	if err = os.Rename(temporaryFile.Name(), path); err != nil { //nolint:gosec
		return fmt.Errorf("failed to write edit error details to file: %w", err)
	}
	return nil
}

func addEditErrorToContents(contents []byte, editErr error) []byte {
	cleanContents := trimPreviousEditError(contents)
	errLines := strings.Split(editErr.Error(), "\n")

	var b strings.Builder
	b.WriteString(editErrorHeader)
	b.WriteString(errLines[0])
	b.WriteString("\n")
	for _, line := range errLines[1:] {
		b.WriteString("# ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("#\n\n")
	b.Write(cleanContents)

	return []byte(b.String())
}

func trimPreviousEditError(contents []byte) []byte {
	lines := strings.Split(string(contents), "\n")
	if len(lines) == 0 {
		return contents
	}
	if !strings.HasPrefix(lines[0], editErrorHeader) {
		return contents
	}

	index := 0
	for index < len(lines) && strings.HasPrefix(lines[index], "#") {
		index++
	}
	if index < len(lines) && lines[index] == "" {
		index++
	}

	return []byte(strings.Join(lines[index:], "\n"))
}

func writeObjectsToTemporaryFile(objects []manifest.Object) (path string, contents []byte, err error) {
	var encoded bytes.Buffer
	if err = sdk.EncodeObjects(objects, &encoded, manifest.ObjectFormatYAML); err != nil {
		return "", nil, fmt.Errorf("failed to encode objects for editing: %w", err)
	}

	tempFile, err := os.CreateTemp("", "sloctl-edit-*.yaml")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temporary file: %w", err)
	}

	if _, err = tempFile.Write(encoded.Bytes()); err != nil {
		if closeErr := tempFile.Close(); closeErr != nil {
			return "", nil, fmt.Errorf("failed to write objects to temporary file: %w (close error: %v)", err, closeErr)
		}
		return "", nil, fmt.Errorf("failed to write objects to temporary file: %w", err)
	}
	if err = tempFile.Close(); err != nil {
		return "", nil, fmt.Errorf("failed to close temporary file: %w", err)
	}

	return tempFile.Name(), encoded.Bytes(), nil
}

type objectIdentity struct {
	kind    manifest.Kind
	project string
	name    string
}

func validateEditedObjectsMatchSelection(original, edited []manifest.Object) error {
	originalIdentities := objectIdentities(original)
	editedIdentities := objectIdentities(edited)
	if maps.Equal(originalIdentities, editedIdentities) {
		return nil
	}
	return fmt.Errorf(
		"edited resources must match the selected resources; changing kind, name, or project is not supported",
	)
}

func objectIdentities(objects []manifest.Object) map[objectIdentity]int {
	identities := make(map[objectIdentity]int, len(objects))
	for i := range objects {
		key := objectIdentity{
			kind:    objects[i].GetKind(),
			project: getObjectProject(objects[i]),
			name:    objects[i].GetName(),
		}
		identities[key]++
	}
	return identities
}

func getObjectProject(object manifest.Object) string {
	projectScopedObject, ok := object.(manifest.ProjectScopedObject)
	if !ok {
		return ""
	}
	return projectScopedObject.GetProject()
}

func validateRequestedObjectsFound(names []string, objects []manifest.Object) error {
	foundNames := make(map[string]struct{}, len(objects))
	for i := range objects {
		foundNames[objects[i].GetName()] = struct{}{}
	}

	seenMissingNames := make(map[string]struct{}, len(names))
	missingNames := make([]string, 0)
	for _, name := range names {
		if _, ok := foundNames[name]; ok {
			continue
		}
		if _, ok := seenMissingNames[name]; ok {
			continue
		}
		seenMissingNames[name] = struct{}{}
		missingNames = append(missingNames, name)
	}
	if len(missingNames) == 0 {
		return nil
	}
	return fmt.Errorf("resource(s) not found: %s", strings.Join(missingNames, ", "))
}

func runEditor(filePath string) error {
	goOS := runtime.GOOS
	editor := resolveEditor(goOS, exec.LookPath)
	shell := resolveShell(goOS)
	editorCmd := exec.Command(shell, shellCommandArg(goOS, shell), fmt.Sprintf("%s %q", editor, filePath))
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr
	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("failed to run editor %q: %w", editor, err)
	}
	return nil
}

type editorLookup func(string) (string, error)

func resolveEditor(goOS string, lookPath editorLookup) string {
	for _, envName := range []string{editorEnvSloctl, editorEnvSystem} {
		if editor := strings.TrimSpace(os.Getenv(envName)); editor != "" {
			return editor
		}
	}
	return defaultEditorForOS(goOS, lookPath)
}

func defaultEditorForOS(goOS string, lookPath editorLookup) string {
	switch goOS {
	case "windows":
		return defaultEditorWindows
	default:
		return defaultEditorForUnix(lookPath)
	}
}

func defaultEditorForUnix(lookPath editorLookup) string {
	for _, editor := range []string{defaultEditorUnixVim, defaultEditorUnixVi} {
		if _, err := lookPath(editor); err == nil {
			return editor
		}
	}
	return defaultEditorUnixFallback
}

func resolveShell(goOS string) string {
	if shell := strings.TrimSpace(os.Getenv(shellEnv)); shell != "" {
		return shell
	}
	return defaultShellForOS(goOS)
}

func defaultShellForOS(goOS string) string {
	if goOS == "windows" {
		return defaultShellWindows
	}
	return defaultShellUnix
}

func shellCommandArg(goOS, shell string) string {
	if goOS == "windows" && isWindowsCommandPrompt(shell) {
		return shellArgWindows
	}
	return shellArgUnix
}

func isWindowsCommandPrompt(shell string) bool {
	base := strings.ToLower(filepath.Base(shell))
	base = strings.TrimSuffix(base, ".exe")
	return base == defaultShellWindows
}
