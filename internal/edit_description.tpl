Edit resources from the default editor.

The edit command allows you to directly edit resources you can retrieve via
sloctl. It will open the editor defined by your {{ .EditorEnvSloctl }} or
{{ .EditorEnvSystem }} environment variables. If neither is defined, it falls
back to {{ .DefaultEditorMacOS }} for macOS, {{ .DefaultEditorWindows }} for
Windows, or the first available editor from {{ .DefaultEditorUnixVim }},
{{ .DefaultEditorUnixVi }}, {{ .DefaultEditorUnixFallback }} for Unix systems.

When attempting to open the editor, sloctl will first attempt to use the shell
defined in the {{ .ShellEnv }} environment variable. If this is not defined, the
default shell will be used, which is {{ .DefaultShellUnix }} for Unix systems or
{{ .DefaultShellWindows }} for Windows.

In the event an error occurs while applying your changes, a temporary file will
be preserved on disk with your unapplied changes.
