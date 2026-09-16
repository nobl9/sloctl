Edit resources from the default editor.

The edit command allows you to directly edit Nobl9 resources like SLOs or Alert Policies.
Selected resources are written to a temporary YAML file before the editor opens.
It will open the editor defined by your `{{ .EditorEnvSloctl }}` or `{{ .EditorEnvSystem }}` environment variables.
`{{ .EditorEnvSloctl }}` takes precedence over `{{ .EditorEnvSystem }}`.
If neither is defined, it falls back to:

- `{{ .DefaultEditorWindows }}` for Windows
- The first available editor from `{{ .DefaultEditorUnixVim }}`, `{{ .DefaultEditorUnixVi }}`, `{{ .DefaultEditorUnixFallback }}` for Unix systems, including macOS

When opening the editor, sloctl first uses the shell defined in the `{{ .ShellEnv }}` environment variable.
If this is not defined, the default shell is `{{ .DefaultShellUnix }}` for Unix systems or `{{ .DefaultShellWindows }}` for Windows.

Saving an empty or unchanged file cancels the operation.
Removing a resource from the file does not delete it.
You cannot change its kind, name, or project.
Invalid YAML or server errors reopen the editor with error details.

If the editor fails, sloctl preserves the temporary file and prints its path.
It also preserves the file and prints its path if you abandon invalid changes without fixing or reverting them.
