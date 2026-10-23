{{define "help" -}}
# {{.CommandPath}}

{{with (or .Long .Short)}}{{. | trim}}

{{end}}{{if or .Runnable .HasSubCommands}}{{template "usage" .}}{{end}}
{{- end}}

{{define "usage" -}}
## Usage

~~~text
{{if .Runnable}}{{.UseLine}}
{{end}}{{if .HasAvailableSubCommands}}{{.CommandPath}} [command]
{{end}}~~~
{{if .Aliases}}
## Aliases

{{.NameAndAliases}}
{{end}}{{if .HasExample}}
## Examples

~~~sh
{{.Example | trim}}
~~~
{{end}}{{if .HasAvailableSubCommands}}{{$commands := .Commands}}{{if not .Groups}}
## Available Commands
{{range $commands}}{{if or .IsAvailableCommand (eq .Name "help")}}
- `{{.Name}}`: {{.Short}}
{{end}}{{end}}{{else}}{{range .Groups}}{{$group := .}}
## {{.Title}}
{{range $commands}}{{if and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help"))}}
- `{{.Name}}`: {{.Short}}
{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}
## Additional Commands
{{range $commands}}{{if and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help"))}}
- `{{.Name}}`: {{.Short}}
{{end}}{{end}}{{end}}{{end}}{{end}}{{end}}

{{define "footer" -}}
{{if .HasHelpSubCommands}}
## Additional help topics
{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
- `{{.CommandPath}}`: {{.Short}}
{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}
Use `{{.CommandPath}} [command] --help` for more information about a command.
{{end}}{{end}}
