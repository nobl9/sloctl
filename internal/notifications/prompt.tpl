{{define "release" -}}
New sloctl version {{.TagName}} is available!

{{range releaseHighlights .Body -}}
{{.}}

{{end -}}
📜 {{.HTMLURL}}

{{end}}
