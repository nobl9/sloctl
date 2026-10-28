package internal

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/template"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
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
		isDark, err := helpHasDarkBackground(cmd, out)
		if err != nil {
			return
		}
		rendered, err := renderHelp(cmd, "help", helpWidth(out), isDark)
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
		isDark, err := helpHasDarkBackground(cmd, out)
		if err != nil {
			return err
		}
		rendered, err := renderHelp(cmd, "usage", helpWidth(out), isDark)
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

func helpHasDarkBackground(cmd *cobra.Command, out io.Writer) (bool, error) {
	file, ok := cmd.InOrStdin().(*os.File)
	if !ok || !term.IsTerminal(file.Fd()) {
		return true, nil
	}
	parent := cmd.Context()
	if parent == nil {
		parent = context.Background()
	}
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// WithoutRenderer also disables Bubble Tea's terminal setup. Own raw mode
	// here so help adds no cursor controls or blank lines and restores input on exit.
	state, err := term.MakeRaw(file.Fd())
	if err != nil {
		return true, nil
	}
	defer func() { _ = term.Restore(file.Fd(), state) }()

	model, err := tea.NewProgram(helpBackground{isDark: true},
		tea.WithInput(file), tea.WithOutput(out), tea.WithContext(ctx),
		tea.WithoutRenderer(), tea.WithoutSignalHandler(),
	).Run()
	if errors.Is(err, tea.ErrInterrupted) || ctx.Err() != nil {
		// Cobra's help callback cannot return an error. Carry cancellation back
		// to executeRootCommand through the root context, including usage callbacks.
		canceled, cancel := context.WithCancelCause(parent)
		cancel(tea.ErrInterrupted)
		cmd.Root().SetContext(canceled)
		cmd.Root().SilenceErrors = true
		return true, tea.ErrInterrupted
	}
	if err != nil {
		return true, nil
	}
	return model.(helpBackground).isDark, nil
}

type helpBackground struct {
	isDark bool
}

type helpBackgroundTimeout struct{}

func (m helpBackground) Init() tea.Cmd {
	return tea.Batch(
		tea.Raw(ansi.RequestBackgroundColor+ansi.RequestPrimaryDeviceAttributes),
		tea.Tick(2*time.Second, func(time.Time) tea.Msg { return helpBackgroundTimeout{} }),
	)
}

func (m helpBackground) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.isDark = msg.IsDark()
	case uv.PrimaryDeviceAttributesEvent, helpBackgroundTimeout:
		return m, tea.Quit
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Interrupt
		}
	}
	return m, nil
}

func (m helpBackground) View() tea.View {
	return tea.NewView("")
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
	return strings.TrimRight(output.String(), "\n") + "\n", nil
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
