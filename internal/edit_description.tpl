Fetch selected resources, write them to a temporary YAML file, and open the file in the editor selected by `{{ .EditorEnvSloctl }}`, then `{{ .EditorEnvSystem }}`.
If neither is set, sloctl uses `{{ .DefaultEditorWindows }}` on Windows or the first available editor from `{{ .DefaultEditorUnixVim }}`, `{{ .DefaultEditorUnixVi }}`, and `{{ .DefaultEditorUnixFallback }}` on Unix systems.

Saving an empty or unchanged file cancels the operation.
Removing a resource from the file does not delete it, and changing its kind, name, or project is not supported.
Invalid YAML or server errors reopen the editor with error details.

If the editor fails, or invalid changes are abandoned without being fixed or reverted, sloctl preserves the temporary file and prints its path.
