Fetch selected resources into a temporary YAML file and open it for editing.
Use `{{ .EditorEnvSloctl }}` if set, otherwise `{{ .EditorEnvSystem }}`.
If neither is set, sloctl uses the platform default.
On Windows, the default is `{{ .DefaultEditorWindows }}`.
On Unix systems, it is the first available editor from `{{ .DefaultEditorUnixVim }}`, `{{ .DefaultEditorUnixVi }}`, and `{{ .DefaultEditorUnixFallback }}`.

Saving an empty or unchanged file cancels the operation.
Removing a resource from the file does not delete it.
You cannot change its kind, name, or project.
Invalid YAML or server errors reopen the editor with error details.

If the editor fails, sloctl preserves the temporary file and prints its path.
It also preserves the file and prints its path if you abandon invalid changes without fixing or reverting them.
