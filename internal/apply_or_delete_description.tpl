{{ .Description }}

Use `--file` for each input source.
A source can be a local file, directory, URL, standard input (`-`), or glob pattern.
Directories are read one level deep; use `**` in a glob to match recursively.

Globs follow [Go's `filepath.Match` syntax](https://pkg.go.dev/path/filepath#Match) with added `**` support.
Directory and glob sources include only {{ .Extensions }} files with a Nobl9 `apiVersion`; unrelated files are skipped.
An explicit file without a Nobl9 `apiVersion` is rejected.
Quote glob patterns so the shell does not expand them before sloctl receives them.
