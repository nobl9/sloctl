{{ .Description }}
Sloctl supports glob patterns when using '-f' flag, it uses the standard Go glob patterns grammar and extends it with support of '**' for recursive reading of files and directories.
The standard Go grammar can be found here: https://pkg.go.dev/path/filepath#Match.
Only files with extensions: {{ .Extensions }} are processed when using glob patterns.
Additionally, before processing the file contents, sloctl checks if it contains Nobl9 API version with the following regex: '{{ .Regex }}'.
Remember that glob patterns must be quoted to prevent the shell from evaluating them.
