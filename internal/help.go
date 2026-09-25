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
	"github.com/charmbracelet/colorprofile"
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
		rendered, err := renderHelp(cmd, "help", helpWidth(out))
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
		rendered, err := renderHelp(cmd, "usage", helpWidth(out))
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

func renderHelp(cmd *cobra.Command, name string, width int) (string, error) {
	tpl, err := template.New("help").Funcs(template.FuncMap{
		"flags": flagsMarkdown,
		"trim":  strings.TrimSpace,
	}).Parse(helpTemplate)
	if err != nil {
		return "", fmt.Errorf("parse help template: %w", err)
	}
	var markdown bytes.Buffer
	if err := tpl.ExecuteTemplate(&markdown, name, cmd); err != nil {
		return "", fmt.Errorf("format help: %w", err)
	}
	return renderHelpMarkdown(markdown.String(), width)
}

func renderHelpMarkdown(markdown string, width int) (string, error) {
	theme := style.MarkdownTheme()
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
			glamour.WithChromaFormatter("terminal16"),
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

func flagsMarkdown(flags *pflag.FlagSet) string {
	var markdown strings.Builder
	escape := strings.NewReplacer("|", "\\|", "\n", " ")
	markdown.WriteString("| Flag | Description |\n| :--- | :--- |\n")
	flags.VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden {
			return
		}
		// Let pflag retain value syntax, defaults, and deprecation notices.
		single := pflag.NewFlagSet(flag.Name, pflag.ContinueOnError)
		single.AddFlag(flag)
		syntax, description, _ := strings.Cut(strings.TrimSpace(single.FlagUsages()), "  ")
		if flag.Shorthand == "" || flag.ShorthandDeprecated != "" {
			syntax = "    " + syntax
		}
		description = strings.TrimSpace(description)
		if values := flag.Annotations[FlagDescriptionMarkdownAnnotation]; len(values) == 1 {
			_, usage := pflag.UnquoteUsage(flag)
			if suffix, ok := strings.CutPrefix(description, usage); ok {
				description = values[0] + suffix
			}
		}
		fmt.Fprintf(&markdown, "| `%s` | %s |\n", escape.Replace(syntax), escape.Replace(description))
	})
	return markdown.String()
}
