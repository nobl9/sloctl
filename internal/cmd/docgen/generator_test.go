package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobl9/sloctl/internal"
)

func TestGenerateCommandReference_ActualSloctlIsDeterministic(t *testing.T) {
	root := internal.NewRootCmd()
	first, err := generateCommandReference(root)
	require.NoError(t, err)
	second, err := generateCommandReference(internal.NewRootCmd())
	require.NoError(t, err)

	assert.Equal(t, first, second)
	assert.True(t, strings.HasSuffix(string(first), "\n"))

	var document commandReferenceDocument
	require.NoError(t, json.Unmarshal(first, &document))
	assert.Equal(t, commandReferenceSchemaVersion, document.SchemaVersion)
	assert.Equal(t, "sloctl", document.Command.Name)
	assert.Equal(t, "sloctl", document.Command.Path)
	assert.Contains(t, document.Command.DescriptionMarkdown, "Manage Nobl9 resources")
	assert.Contains(t, document.Command.Options, optionReference{
		Syntax:              "--config string",
		DescriptionMarkdown: "Path to config.toml. If unset, use `SLOCTL_CONFIG_FILE_PATH` or the platform default.",
	})
	assert.Contains(t, document.Command.Options, optionReference{
		Syntax: "--no-config-file",
		DescriptionMarkdown: "For API authentication, use `SLOCTL_CLIENT_ID` and `SLOCTL_CLIENT_SECRET` " +
			"without reading or creating `config.toml`. Configuration commands still access the file.",
	})

	paths := commandPaths(document.Command)
	assert.Contains(t, paths, "sloctl apply")
	assert.Contains(t, paths, "sloctl budgetadjustments events update")
	assert.Contains(t, paths, "sloctl completion")
	assert.NotContains(t, paths, "sloctl help")

	apply := requireCommandReference(t, document.Command, "sloctl apply")
	assert.Equal(t, "sloctl apply [flags]", apply.Usage)
	applyCommand, _, err := root.Find([]string{"apply"})
	require.NoError(t, err)
	assert.NotEmpty(t, applyCommand.Example)
	assert.Equal(t, applyCommand.Example, apply.Example)
	assert.NotEmpty(t, apply.Options)
	assert.NotEmpty(t, apply.InheritedOptions)
	assert.Contains(t, apply.Options, optionReference{
		Syntax: "-f, --file stringArray",
		DescriptionMarkdown: "Path, directory, URL, glob pattern, or `-` for YAML or JSON from standard input. " +
			"Repeat this flag to use multiple sources.",
	})
	assert.Contains(t, apply.Options, optionReference{
		Syntax: "-y, --yes",
		DescriptionMarkdown: "Skip the file-count confirmation prompt. By default, the prompt appears when a " +
			"directory or glob resolves to more than 23 files. Configure `filesPromptEnabled` and " +
			"`filesPromptThreshold` in the `[sloctl]` section of `config.toml`, or set `SLOCTL_FILES_PROMPT_ENABLED` " +
			"and `SLOCTL_FILES_PROMPT_THRESHOLD`.",
	})
	replay := requireCommandReference(t, document.Command, "sloctl replay")
	assert.Contains(t, replay.Options, optionReference{
		Syntax: "-f, --file stringArray",
		DescriptionMarkdown: "Path to a local YAML or JSON Replay configuration file. " +
			"Repeat this flag to use multiple files.",
	})

	agents := requireCommandReference(t, document.Command, "sloctl get agents")
	assert.Contains(t, agents.DescriptionMarkdown, "`--with-keys`")
	assert.Contains(t, agents.Options, optionReference{
		Syntax: "-k, --with-keys",
		DescriptionMarkdown: "Include agent client_id and client_secret values. " +
			"This performs one additional credential request per returned agent.",
	})
	for _, option := range agents.Options {
		assert.NotContains(t, option.Syntax, "--with-keys client_id")
	}

	awsIAMIDs := requireCommandReference(t, document.Command, "sloctl aws-iam-ids")
	assert.Empty(t, awsIAMIDs.Usage)
}

func TestGenerateCommandReference_PreservesCommandDataAndHierarchy(t *testing.T) {
	root := &cobra.Command{
		Use:   "tool",
		Short: "Root short.",
		Long:  "Root **Markdown**.",
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.PersistentFlags().String("context", "", "Use another context.")

	visible := &cobra.Command{
		Use:     "visible <name>",
		Short:   "Visible short.",
		Long:    "Visible long.",
		Example: "tool visible example",
		Run:     func(*cobra.Command, []string) {},
	}
	visible.Flags().String("format", "yaml", "Output `format`.")
	visible.Flags().Bool("hidden-option", false, "Not public.")
	require.NoError(t, visible.Flags().MarkHidden("hidden-option"))
	shortOnly := &cobra.Command{
		Use:   "short-only",
		Short: "Short only.",
		Run:   func(*cobra.Command, []string) {},
	}
	hidden := &cobra.Command{
		Use:    "hidden",
		Hidden: true,
		Run:    func(*cobra.Command, []string) {},
	}
	root.AddCommand(visible, shortOnly, hidden)

	contents, err := generateCommandReference(root)
	require.NoError(t, err)

	var document commandReferenceDocument
	require.NoError(t, json.Unmarshal(contents, &document))
	assert.Equal(t, []string{"tool", "tool short-only", "tool visible"}, commandPaths(document.Command))
	assert.Equal(t, "Root **Markdown**.", document.Command.DescriptionMarkdown)
	assert.Equal(t, "Root short.", document.Command.ShortDescription)
	assert.Empty(t, document.Command.Usage)

	command := requireCommandReference(t, document.Command, "tool visible")
	assert.Equal(t, "tool visible <name> [flags]", command.Usage)
	assert.Equal(t, "Visible long.", command.DescriptionMarkdown)
	assert.Equal(t, "Visible short.", command.ShortDescription)
	assert.Equal(t, "tool visible example", command.Example)
	assert.Contains(t, command.Options, optionReference{
		Syntax:              "--format format",
		DescriptionMarkdown: "Output format. (default \"yaml\")",
	})
	assert.NotContains(t, command.Options, optionReference{
		Syntax:              "--hidden-option",
		DescriptionMarkdown: "Not public.",
	})
	assert.Contains(t, command.InheritedOptions, optionReference{
		Syntax:              "--context string",
		DescriptionMarkdown: "Use another context.",
	})

	shortOnlyCommand := requireCommandReference(t, document.Command, "tool short-only")
	assert.Equal(t, "Short only.", shortOnlyCommand.DescriptionMarkdown)
	assert.NotNil(t, shortOnlyCommand.Options)
	assert.NotNil(t, shortOnlyCommand.InheritedOptions)
	assert.NotNil(t, shortOnlyCommand.Subcommands)
}

func TestCommandReferenceJSONContract(t *testing.T) {
	document := commandReferenceDocument{
		SchemaVersion: 2,
		Command: commandReference{
			Name:                "child",
			Path:                "tool child",
			ShortDescription:    "Child short.",
			DescriptionMarkdown: "Description.",
			Usage:               "tool child [flags]",
			Example:             "tool child --option",
			Options: []optionReference{
				{Syntax: "--option string", DescriptionMarkdown: "Set an option."},
			},
			InheritedOptions: []optionReference{},
			Subcommands:      []commandReference{},
		},
	}

	contents, err := json.Marshal(document)

	require.NoError(t, err)
	assert.JSONEq(t, `{
		"schemaVersion": 2,
		"command": {
			"name": "child",
			"path": "tool child",
			"shortDescription": "Child short.",
			"descriptionMarkdown": "Description.",
			"usage": "tool child [flags]",
			"example": "tool child --option",
			"options": [{"syntax": "--option string", "description": "Set an option."}],
			"inheritedOptions": [],
			"subcommands": []
		}
	}`, string(contents))
}

func Test_optionReferenceFromPreservesPflagFormatting(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.StringP(
		"name",
		"n",
		"default",
		"Select a `value  with spaces` | path\\segment.",
	)
	flag := flags.Lookup("name")
	require.NotNil(t, flag)
	flag.NoOptDefVal = "foo  bar"

	option, err := optionReferenceFrom(flag)

	require.NoError(t, err)
	assert.Equal(t, "-n, --name value  with spaces[=\"foo  bar\"]", option.Syntax)
	assert.Equal(
		t,
		"Select a value  with spaces | path\\segment. (default \"default\")",
		option.DescriptionMarkdown,
	)
}

func Test_optionReferenceFromUsesMarkdownAnnotation(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("source", "default.yaml", "Read a source.")
	flag := flags.Lookup("source")
	flag.Annotations = map[string][]string{
		internal.FlagDescriptionMarkdownAnnotation: {"Read `source.yaml`."},
	}

	option, err := optionReferenceFrom(flag)

	require.NoError(t, err)
	assert.Equal(t,
		"Read `source.yaml`. (default \"default.yaml\")",
		option.DescriptionMarkdown,
	)
}

func Test_optionReferenceFromRejectsInvalidMarkdownAnnotation(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("source", "", "Read a source.")
	flag := flags.Lookup("source")
	flag.Annotations = map[string][]string{
		internal.FlagDescriptionMarkdownAnnotation: {"first", "second"},
	}

	_, err := optionReferenceFrom(flag)

	require.ErrorContains(t, err, "must contain one non-empty value")
}

func Test_optionSyntaxPreservesPflagNoOptAndShorthandRules(t *testing.T) {
	tests := []struct {
		name     string
		flag     func(*pflag.FlagSet) *pflag.Flag
		expected string
	}{
		{
			name: "boolean true is implicit",
			flag: func(flags *pflag.FlagSet) *pflag.Flag {
				flags.BoolP("verbose", "v", false, "Enable verbose output.")
				return flags.Lookup("verbose")
			},
			expected: "-v, --verbose",
		},
		{
			name: "nonstandard boolean value",
			flag: func(flags *pflag.FlagSet) *pflag.Flag {
				flags.BoolP("verbose", "v", false, "Enable verbose output.")
				flag := flags.Lookup("verbose")
				flag.NoOptDefVal = "false"
				return flag
			},
			expected: "-v, --verbose[=false]",
		},
		{
			name: "count increment is implicit",
			flag: func(flags *pflag.FlagSet) *pflag.Flag {
				flags.CountP("count", "c", "Increase the count.")
				return flags.Lookup("count")
			},
			expected: "-c, --count count",
		},
		{
			name: "nonstandard count value",
			flag: func(flags *pflag.FlagSet) *pflag.Flag {
				flags.CountP("count", "c", "Increase the count.")
				flag := flags.Lookup("count")
				flag.NoOptDefVal = "2"
				return flag
			},
			expected: "-c, --count count[=2]",
		},
		{
			name: "other value type",
			flag: func(flags *pflag.FlagSet) *pflag.Flag {
				flags.Float64P("ratio", "r", 0, "Set the `ratio`.")
				flag := flags.Lookup("ratio")
				flag.NoOptDefVal = "1.5"
				return flag
			},
			expected: "-r, --ratio ratio[=1.5]",
		},
		{
			name: "deprecated shorthand",
			flag: func(flags *pflag.FlagSet) *pflag.Flag {
				flags.BoolP("legacy", "l", false, "Use legacy behavior.")
				flag := flags.Lookup("legacy")
				flag.ShorthandDeprecated = "use --legacy"
				return flag
			},
			expected: "--legacy",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			flags := pflag.NewFlagSet("test", pflag.ContinueOnError)

			assert.Equal(t, test.expected, optionSyntax(test.flag(flags)))
		})
	}
}

func Test_optionReferencesOmitsHiddenFlags(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.Bool("visible", false, "Visible option.")
	flags.Bool("hidden", false, "Hidden option.")
	require.NoError(t, flags.MarkHidden("hidden"))

	options, err := optionReferences(flags)

	require.NoError(t, err)
	assert.Equal(t,
		[]optionReference{{Syntax: "--visible", DescriptionMarkdown: "Visible option."}},
		options,
	)
}

func Test_optionReferenceFromRejectsMultilineSyntax(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("name", "", "Select a `value\nname`.")

	_, err := optionReferenceFrom(flags.Lookup("name"))

	require.ErrorContains(t, err, "option syntax contains a line break")
}

func TestWriteCommandReference(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "nested", "reference.json")
	root := &cobra.Command{Use: "tool", Short: "Test command."}
	root.CompletionOptions.DisableDefaultCmd = true
	require.NoError(t, os.MkdirAll(filepath.Dir(outputPath), 0o750))
	require.NoError(t, os.WriteFile(outputPath, []byte("stale"), 0o600))

	require.NoError(t, writeCommandReference(root, outputPath))
	contents, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	var document commandReferenceDocument
	require.NoError(t, json.Unmarshal(contents, &document))
	assert.Equal(t, "tool", document.Command.Path)
}

func TestWriteCommandReferenceCleansUpAfterRenameFailure(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "existing-directory")
	require.NoError(t, os.Mkdir(outputPath, 0o750))
	root := &cobra.Command{Use: "tool", Short: "Test command."}
	root.CompletionOptions.DisableDefaultCmd = true

	err := writeCommandReference(root, outputPath)

	require.ErrorContains(t, err, "replace command reference data")
	entries, readErr := os.ReadDir(filepath.Dir(outputPath))
	require.NoError(t, readErr)
	assert.Len(t, entries, 1)
	assert.True(t, entries[0].IsDir())
	assert.Equal(t, filepath.Base(outputPath), entries[0].Name())
}

func TestNewCommand(t *testing.T) {
	cmd := newCommand()

	outputFlag := cmd.Flags().Lookup("output")
	require.NotNil(t, outputFlag)
	assert.Equal(t, defaultOutputPath, outputFlag.DefValue)
	assert.Nil(t, cmd.Flags().Lookup("overrides-dir"))
}

func requireCommandReference(
	t *testing.T,
	root commandReference,
	path string,
) commandReference {
	t.Helper()
	if reference, ok := findCommandReference(root, path); ok {
		return reference
	}
	require.FailNow(t, "command reference not found", path)
	return commandReference{}
}

func findCommandReference(root commandReference, path string) (commandReference, bool) {
	if root.Path == path {
		return root, true
	}
	for _, command := range root.Subcommands {
		if reference, ok := findCommandReference(command, path); ok {
			return reference, true
		}
	}
	return commandReference{}, false
}

func commandPaths(root commandReference) []string {
	paths := make([]string, 1, 1+len(root.Subcommands))
	paths[0] = root.Path
	for _, command := range root.Subcommands {
		paths = append(paths, commandPaths(command)...)
	}
	return paths
}
