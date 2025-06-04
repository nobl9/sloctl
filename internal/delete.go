package internal

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
)

type DeleteCmd struct {
	client            *sdk.Client
	projectFlagWasSet bool
	definitionPaths   []string
	dryRun            bool
	autoConfirm       bool
	project           string
}

//go:embed delete_example.sh
var deleteExample string

// NewDeleteCmd returns cobra command delete with all its flags.
func (r *RootCmd) NewDeleteCmd() *cobra.Command {
	deleteCmd := &DeleteCmd{}

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete object definition by name or definition file",
		Long: getApplyOrDeleteDescription(
			"One or more definitions can be specified by name or provide a path to file with definitions to remove."),
		Example: deleteExample,
		Args:    positionalArgsCondition,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			deleteCmd.client = r.GetClient()
			if deleteCmd.project != "" {
				deleteCmd.projectFlagWasSet = true
				deleteCmd.client.Config.Project = deleteCmd.project
			}
			if deleteCmd.dryRun {
				notifyDryRunFlag()
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error { return deleteCmd.Run(cmd) },
	}

	registerFileFlag(cmd, false, &deleteCmd.definitionPaths)
	registerDryRunFlag(cmd, &deleteCmd.dryRun)
	registerAutoConfirmationFlag(cmd, &deleteCmd.autoConfirm)
	cmd.Flags().StringVarP(&deleteCmd.project, "project", "p", "",
		`Assigns the provided Project to the resources if no Project is defined in the object's definition.`)

	// register all subcommands for delete
	for _, def := range []struct {
		kind manifest.Kind
		// plural if not provided will append 's' at the end of a singular manifest.Kind name.
		plural string
		// aliases always contains the singular lowercase name of an manifest.Kind.
		aliases []string
	}{
		{kind: manifest.KindAgent},
		{kind: manifest.KindAlertMethod},
		{kind: manifest.KindAlertPolicy, plural: "AlertPolicies"},
		{kind: manifest.KindAlertSilence},
		{kind: manifest.KindAnnotation},
		{kind: manifest.KindDataExport},
		{kind: manifest.KindDirect},
		{kind: manifest.KindProject},
		{kind: manifest.KindRoleBinding},
		{kind: manifest.KindService, aliases: []string{"svc", "svcs"}},
		{kind: manifest.KindSLO},
		{kind: manifest.KindBudgetAdjustment},
		{kind: manifest.KindReport},
	} {
		if len(def.plural) == 0 {
			def.plural = def.kind.String() + "s"
		}
		cmd.AddCommand(newSubcommand(
			deleteCmd,
			def.kind,
			fmt.Sprintf("Delete the %s.", def.plural),
			strings.ToLower(def.plural),
			append(def.aliases, def.kind.ToLower(), def.kind.String())...))
	}

	return cmd
}

func (d DeleteCmd) Run(cmd *cobra.Command) error {
	if d.dryRun {
		d.client.WithDryRun()
	}
	if len(d.definitionPaths) == 0 {
		return cmd.Usage()
	}
	objects, err := readObjectsDefinitions(
		cmd.Context(),
		d.client.Config,
		cmd,
		d.definitionPaths,
		newFilesPrompt(d.client.Config.FilesPromptEnabled, d.autoConfirm, d.client.Config.FilesPromptThreshold),
		d.projectFlagWasSet)
	if err != nil {
		return err
	}
	printSourcesDetails("Deleting", objects, os.Stdout)
	if err = d.client.Objects().V1().Delete(cmd.Context(), objects); err != nil {
		return err
	}
	printCommandResult("The resources were successfully deleted.", d.dryRun)
	return nil
}

func newSubcommand(
	deleteCmd *DeleteCmd,
	kind manifest.Kind,
	shortDesc, useCmd string,
	aliases ...string,
) *cobra.Command {
	sc := &cobra.Command{
		Use:     useCmd,
		Aliases: aliases,
		Short:   shortDesc,
		Args:    cobra.MinimumNArgs(1), //nolint: gomnd
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSubcommand(cmd.Context(), deleteCmd, kind, args)
		},
	}
	if objectKindSupportsProjectFlag(kind) {
		sc.Flags().StringVarP(&deleteCmd.project, "project", "p", "",
			`Specifies the Project from which to delete the resources. If not provided, the default Project will be used.`)
	}
	registerDryRunFlag(sc, &deleteCmd.dryRun)
	return sc
}

func runSubcommand(ctx context.Context, deleteCmd *DeleteCmd, kind manifest.Kind, args []string) error {
	if err := deleteCmd.client.Objects().V1().DeleteByName(
		ctx,
		kind,
		deleteCmd.client.Config.Project,
		args...,
	); err != nil {
		return err
	}
	printCommandResult("The resources were successfully deleted.", deleteCmd.dryRun)
	return nil
}
