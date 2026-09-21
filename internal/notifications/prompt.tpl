{{define "release" -}}
New sloctl version {{.TagName}} is available!

{{range releaseHighlights .Body -}}
{{.}}

{{end -}}
📜 {{.HTMLURL}}

{{end}}

{{define "actions" -}}
Choose update action
{{range $i, $option := . -}}
{{inc $i}}. {{$option.Key}}
{{end -}}
Enter a number between 1 and {{len .}} [2]: {{end}}
