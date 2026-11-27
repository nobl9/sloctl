package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/nobl9/sloctl/internal"
)

const commandReferenceSchemaVersion = 2

type commandReferenceDocument struct {
	SchemaVersion int              `json:"schemaVersion"`
	Command       commandReference `json:"command"`
}

type commandReference struct {
	Name                string             `json:"name"`
	Path                string             `json:"path"`
	ShortDescription    string             `json:"shortDescription"`
	DescriptionMarkdown string             `json:"descriptionMarkdown,omitzero"`
	Usage               string             `json:"usage,omitzero"`
	Example             string             `json:"example,omitzero"`
	Options             []optionReference  `json:"options"`
	InheritedOptions    []optionReference  `json:"inheritedOptions"`
	Subcommands         []commandReference `json:"subcommands"`
}

type optionReference struct {
	Syntax              string `json:"syntax"`
	DescriptionMarkdown string `json:"description,omitzero"`
}

func writeCommandReference(root *cobra.Command, outputPath string) error {
	contents, err := generateCommandReference(root)
	if err != nil {
		return err
	}
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0o750); err != nil {
		return fmt.Errorf("create output directory %q: %w", outputDir, err)
	}
	temporaryFile, err := os.CreateTemp(outputDir, ".sloctl-command-reference-*.json")
	if err != nil {
		return fmt.Errorf("create temporary command reference data in %q: %w", outputDir, err)
	}
	temporaryPath := temporaryFile.Name()
	defer func() {
		_ = os.Remove(temporaryPath)
	}()
	if err := temporaryFile.Chmod(0o600); err != nil {
		_ = temporaryFile.Close()
		return fmt.Errorf("set permissions on temporary command reference data %q: %w", temporaryPath, err)
	}
	if _, err := temporaryFile.Write(contents); err != nil {
		_ = temporaryFile.Close()
		return fmt.Errorf("write temporary command reference data %q: %w", temporaryPath, err)
	}
	if err := temporaryFile.Close(); err != nil {
		return fmt.Errorf("close temporary command reference data %q: %w", temporaryPath, err)
	}
	if err := os.Rename(temporaryPath, outputPath); err != nil {
		return fmt.Errorf("replace command reference data %q: %w", outputPath, err)
	}
	return nil
}

func generateCommandReference(root *cobra.Command) ([]byte, error) {
	root.InitDefaultCompletionCmd()
	command, err := commandReferenceFrom(root)
	if err != nil {
		return nil, err
	}
	document := commandReferenceDocument{
		SchemaVersion: commandReferenceSchemaVersion,
		Command:       command,
	}
	contents, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode command reference data: %w", err)
	}
	return append(contents, '\n'), nil
}

func commandReferenceFrom(cmd *cobra.Command) (commandReference, error) {
	cmd.InitDefaultHelpFlag()
	options, err := optionReferences(cmd.NonInheritedFlags())
	if err != nil {
		return commandReference{}, fmt.Errorf("collect options for %q: %w", cmd.CommandPath(), err)
	}
	inheritedOptions, err := optionReferences(cmd.InheritedFlags())
	if err != nil {
		return commandReference{}, fmt.Errorf(
			"collect inherited options for %q: %w",
			cmd.CommandPath(),
			err,
		)
	}

	subcommands := make([]commandReference, 0, len(cmd.Commands()))
	for _, subcommand := range cmd.Commands() {
		if !subcommand.IsAvailableCommand() {
			continue
		}
		reference, err := commandReferenceFrom(subcommand)
		if err != nil {
			return commandReference{}, err
		}
		subcommands = append(subcommands, reference)
	}

	description := cmd.Long
	if strings.TrimSpace(description) == "" {
		description = cmd.Short
	}
	usage := ""
	if cmd.Runnable() {
		usage = cmd.UseLine()
	}
	return commandReference{
		Name:                cmd.Name(),
		Path:                cmd.CommandPath(),
		ShortDescription:    cmd.Short,
		DescriptionMarkdown: description,
		Usage:               usage,
		Example:             cmd.Example,
		Options:             options,
		InheritedOptions:    inheritedOptions,
		Subcommands:         subcommands,
	}, nil
}

func optionReferences(flags *pflag.FlagSet) ([]optionReference, error) {
	options := make([]optionReference, 0)
	var optionErr error
	flags.VisitAll(func(flag *pflag.Flag) {
		if optionErr != nil || flag.Hidden {
			return
		}
		option, err := optionReferenceFrom(flag)
		if err != nil {
			optionErr = fmt.Errorf("format option %q: %w", flag.Name, err)
			return
		}
		options = append(options, option)
	})
	return options, optionErr
}

func optionReferenceFrom(flag *pflag.Flag) (optionReference, error) {
	// Mark the description before pflag formats the flag. Value names and
	// optional values can contain the same spaces that separate the columns.
	const descriptionStart = "\x00"
	flagCopy := *flag
	flagCopy.Usage = descriptionStart + flag.Usage
	flags := pflag.NewFlagSet(flag.Name, pflag.ContinueOnError)
	flags.AddFlag(&flagCopy)
	syntax, description, ok := strings.Cut(flags.FlagUsages(), descriptionStart)
	if !ok {
		return optionReference{}, fmt.Errorf("formatted usage has no description marker")
	}
	syntax = strings.Trim(syntax, " ")
	if strings.ContainsAny(syntax, "\r\n") {
		return optionReference{}, fmt.Errorf("option syntax contains a line break: %q", syntax)
	}
	formattedDescription := normalizeOptionDescription(description)
	descriptionMarkdown, err := optionDescriptionMarkdown(flag, formattedDescription)
	if err != nil {
		return optionReference{}, err
	}
	return optionReference{
		Syntax:              syntax,
		DescriptionMarkdown: descriptionMarkdown,
	}, nil
}

func optionDescriptionMarkdown(flag *pflag.Flag, formattedDescription string) (string, error) {
	annotationName := internal.FlagDescriptionMarkdownAnnotation
	values, ok := flag.Annotations[annotationName]
	if !ok {
		return formattedDescription, nil
	}
	if len(values) != 1 || strings.TrimSpace(values[0]) == "" {
		return "", fmt.Errorf(
			"annotation %q must contain one non-empty value",
			annotationName,
		)
	}
	descriptionMarkdown := values[0]
	if descriptionMarkdown != strings.TrimSpace(descriptionMarkdown) ||
		strings.ContainsAny(descriptionMarkdown, "\r\n") {
		return "", fmt.Errorf(
			"annotation %q must contain one trimmed line",
			annotationName,
		)
	}

	_, usage := pflag.UnquoteUsage(flag)
	usage = normalizeOptionDescription(usage)
	suffix, ok := strings.CutPrefix(formattedDescription, usage)
	if !ok {
		return "", fmt.Errorf("formatted description does not start with flag usage %q", usage)
	}
	return descriptionMarkdown + suffix, nil
}

func normalizeOptionDescription(description string) string {
	descriptionLines := make([]string, 0, strings.Count(description, "\n")+1)
	for line := range strings.Lines(description) {
		line = strings.TrimSpace(line)
		if line != "" {
			descriptionLines = append(descriptionLines, line)
		}
	}
	return strings.Join(descriptionLines, " ")
}
