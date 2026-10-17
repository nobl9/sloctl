package internal

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/nobl9/sloctl/internal/style"
)

//go:embed help.tpl
var helpTemplate string

func configureHelp(root *cobra.Command) {
	plainHelp := root.HelpFunc()
	plainUsage := root.UsageFunc()
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		out := cmd.OutOrStdout()
		if !styledHelpEnabled(out) {
			plainHelp(cmd, args)
			return
		}
		rendered, err := renderHelp(cmd, "help", helpWidth(out), helpHasDarkBackground(cmd.InOrStdin(), out))
		if err != nil {
			cmd.PrintErrln(err)
			plainHelp(cmd, args)
			return
		}
		if _, err := io.WriteString(colorprofile.NewWriter(out, os.Environ()), rendered); err != nil {
			cmd.PrintErrln(fmt.Errorf("write help: %w", err))
		}
	})
	root.SetUsageFunc(func(cmd *cobra.Command) error {
		out := cmd.OutOrStderr()
		if !styledHelpEnabled(out) {
			return plainUsage(cmd)
		}
		rendered, err := renderHelp(cmd, "usage", helpWidth(out), helpHasDarkBackground(cmd.InOrStdin(), out))
		if err != nil {
			cmd.PrintErrln(err)
			return plainUsage(cmd)
		}
		if _, err := io.WriteString(colorprofile.NewWriter(out, os.Environ()), rendered); err != nil {
			return fmt.Errorf("write usage: %w", err)
		}
		return nil
	})
}

func styledHelpEnabled(out io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	file, ok := out.(*os.File)
	return ok && (isatty.IsTerminal(file.Fd()) || isatty.IsCygwinTerminal(file.Fd()))
}

func helpWidth(out io.Writer) int {
	if file, ok := out.(*os.File); ok {
		if width, _, err := term.GetSize(file.Fd()); err == nil && width > 0 {
			return width
		}
	}
	return 80
}

func helpHasDarkBackground(in io.Reader, out io.Writer) bool {
	file, ok := in.(*os.File)
	if !ok || !term.IsTerminal(file.Fd()) {
		return true
	}
	// Lip Gloss queries each handle as both input and output on Unix. Pair the
	// streams so read-only stdin and write-only terminal output both work.
	terminal := helpTerminal{File: file, output: out}
	return lipgloss.HasDarkBackground(terminal, terminal)
}

type helpTerminal struct {
	*os.File
	output io.Writer
}

func (t helpTerminal) Write(p []byte) (int, error) {
	return t.output.Write(p)
}

func (t helpTerminal) WriteString(s string) (int, error) {
	return io.WriteString(t.output, s)
}

func renderHelp(cmd *cobra.Command, name string, width int, isDark bool) (string, error) {
	tpl, err := template.New("help").Funcs(template.FuncMap{
		"trim": strings.TrimSpace,
	}).Parse(helpTemplate)
	if err != nil {
		return "", fmt.Errorf("parse help template: %w", err)
	}
	var output strings.Builder
	appendMarkdown := func(markdown string) error {
		if strings.TrimSpace(markdown) == "" {
			return nil
		}
		rendered, err := renderHelpMarkdown(markdown, width, isDark)
		if err != nil {
			return err
		}
		output.WriteString(strings.TrimRight(strings.TrimLeft(rendered, "\n"), " \t\n") + "\n\n")
		return nil
	}
	appendSection := func(name string) error {
		var markdown bytes.Buffer
		if err := tpl.ExecuteTemplate(&markdown, name, cmd); err != nil {
			return fmt.Errorf("format help: %w", err)
		}
		return appendMarkdown(markdown.String())
	}
	if err := appendSection(name); err != nil {
		return "", err
	}
	if name == "usage" || cmd.Runnable() || cmd.HasSubCommands() {
		for _, section := range []struct {
			title string
			flags *pflag.FlagSet
		}{
			{"Flags", cmd.LocalFlags()},
			{"Global Flags", cmd.InheritedFlags()},
		} {
			if !section.flags.HasAvailableFlags() {
				continue
			}
			if err := appendMarkdown("## " + section.title); err != nil {
				return "", err
			}
			rendered, err := renderHelpFlags(section.flags, width, isDark)
			if err != nil {
				return "", err
			}
			output.WriteString(rendered + "\n")
		}
		if err := appendSection("footer"); err != nil {
			return "", err
		}
	}
	return "\n" + strings.TrimRight(output.String(), "\n") + "\n", nil
}

func renderHelpMarkdown(markdown string, width int, isDark bool) (string, error) {
	theme := style.MarkdownTheme(isDark)
	theme.Document.BlockPrefix = ""
	theme.Document.BlockSuffix = ""
	var output, block strings.Builder
	renderBlock := func(wrap int) error {
		if strings.TrimSpace(block.String()) == "" {
			block.Reset()
			return nil
		}
		renderer, err := glamour.NewTermRenderer(
			glamour.WithStyles(theme),
			glamour.WithWordWrap(wrap),
			glamour.WithChromaFormatter("terminal16m"),
		)
		if err != nil {
			return fmt.Errorf("create help renderer: %w", err)
		}
		rendered, err := renderer.Render(block.String())
		if err != nil {
			return fmt.Errorf("render help: %w", err)
		}
		output.WriteString(rendered)
		if wrap == 0 {
			output.WriteByte('\n')
		}
		block.Reset()
		return nil
	}

	// Render fenced examples separately so wrapping cannot change shell commands.
	var fence string
	for line := range strings.Lines(markdown) {
		marker := helpCodeFence(line)
		if fence == "" && marker != "" {
			if err := renderBlock(width); err != nil {
				return "", err
			}
			fence = marker
			block.WriteString(line)
			continue
		}
		block.WriteString(line)
		if fence != "" && strings.HasPrefix(marker, fence) &&
			strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), marker)) == "" {
			if err := renderBlock(0); err != nil {
				return "", err
			}
			fence = ""
		}
	}
	if fence != "" {
		width = 0
	}
	if err := renderBlock(width); err != nil {
		return "", err
	}
	return "\n" + output.String() + "\n", nil
}

func helpCodeFence(line string) string {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 || len(trimmed) < 3 || (trimmed[0] != '`' && trimmed[0] != '~') {
		return ""
	}
	i := 1
	for i < len(trimmed) && trimmed[i] == trimmed[0] {
		i++
	}
	if i < 3 {
		return ""
	}
	return trimmed[:i]
}

func renderHelpFlags(flags *pflag.FlagSet, width int, isDark bool) (string, error) {
	var rows []struct{ syntax, description string }
	column := 0
	flags.VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden {
			return
		}
		// Let pflag retain value syntax, defaults, and deprecation notices.
		single := pflag.NewFlagSet(flag.Name, pflag.ContinueOnError)
		single.AddFlag(flag)
		usage := single.FlagUsages()
		trimmed := strings.TrimLeft(usage, " ")
		syntax, description, _ := strings.Cut(trimmed, "  ")
		syntax = usage[:len(usage)-len(trimmed)] + syntax
		column = max(column, ansi.StringWidth(syntax)+3)
		description = strings.TrimSpace(description)
		if values := flag.Annotations[FlagDescriptionMarkdownAnnotation]; len(values) == 1 {
			_, usage := pflag.UnquoteUsage(flag)
			if suffix, ok := strings.CutPrefix(description, usage); ok {
				description = values[0] + suffix
			}
		}
		rows = append(rows, struct{ syntax, description string }{syntax, description})
	})
	// As in pflag, put descriptions below the flags when columns leave too little room.
	stacked := width-column < 24
	if stacked {
		column = min(16, max(width-24, 0))
	}
	indent := strings.Repeat(" ", column)
	var output strings.Builder
	for _, row := range rows {
		trimmed := strings.TrimLeft(row.syntax, " ")
		syntax, err := renderHelpMarkdown("`"+trimmed+"`", 0, isDark)
		if err != nil {
			return "", err
		}
		syntax = row.syntax[:len(row.syntax)-len(trimmed)] + strings.TrimSpace(syntax)
		description, err := renderHelpMarkdown(row.description, max(width-column, 1), isDark)
		if err != nil {
			return "", err
		}
		output.WriteString(syntax)
		if stacked {
			output.WriteString("\n" + indent)
		} else {
			output.WriteString(strings.Repeat(" ", max(column-ansi.StringWidth(syntax), 1)))
		}
		output.WriteString(strings.ReplaceAll(strings.Trim(description, "\n"), "\n", "\n"+indent))
		output.WriteByte('\n')
	}
	return output.String(), nil
}
