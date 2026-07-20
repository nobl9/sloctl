package internal

import (
	_ "embed"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/manifest/v1alpha"
	"github.com/nobl9/nobl9-go/sdk"
	v2 "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v2"

	"github.com/nobl9/sloctl/internal/flags"
)

type ApplyCmd struct {
	client            *sdk.Client
	projectFlagWasSet bool
	definitionPaths   []string
	dryRun            bool
	autoConfirm       bool
	replay            bool
	replayFrom        time.Time
	project           string
}

//go:embed apply_example.sh
var applyExample string

// NewApplyCmd returns cobra command apply with all its flags.
func (r *RootCmd) NewApplyCmd() *cobra.Command {
	apply := &ApplyCmd{}

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply object definition in YAML or JSON format",
		Long: getApplyOrDeleteDescription(
			"The apply command commits the changes by sending the updates to the application."),
		Example: applyExample,
		Args:    noPositionalArgsCondition,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			apply.client = r.GetClient()
			if apply.project != "" {
				apply.projectFlagWasSet = true
				apply.client.Config.Project = apply.project
			}
			if apply.dryRun {
				notifyDryRunFlag()
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error { return apply.Run(cmd) },
	}

	registerFileFlag(cmd, true, &apply.definitionPaths)
	registerDryRunFlag(cmd, &apply.dryRun)
	registerAutoConfirmationFlag(cmd, &apply.autoConfirm)
	cmd.Flags().StringVarP(&apply.project, "project", "p", "",
		`Assigns the provided Project to the resources if no Project is defined in the object's definition.`)

	const (
		replayFlagName     = "replay"
		replayFromFlagName = "from"
	)
	cmd.Flags().BoolVar(&apply.replay, replayFlagName, false,
		"Run Replay for the applied SLOs. If Replay fails, the applied changes are not rolled back.")
	flags.RegisterTimeVar(
		cmd,
		&apply.replayFrom,
		replayFromFlagName,
		"Sets the start of Replay time window.",
	)
	cmd.MarkFlagsRequiredTogether(replayFlagName, replayFromFlagName)

	return cmd
}

func (a ApplyCmd) Run(cmd *cobra.Command) error {
	if len(a.definitionPaths) == 0 {
		return cmd.Usage()
	}
	objects, err := readObjectsDefinitions(
		cmd.Context(),
		a.client.Config,
		cmd,
		a.definitionPaths,
		newFilesPrompt(a.client.Config.FilesPromptEnabled, a.autoConfirm, a.client.Config.FilesPromptThreshold),
		a.projectFlagWasSet)
	if err != nil {
		return err
	}
	printSourcesDetails("Applying", objects, os.Stdout)
	if err = a.client.Objects().V2().Apply(cmd.Context(), v2.ApplyRequest{
		Objects: objects,
		DryRun:  a.dryRun,
	}); err != nil {
		return err
	}
	printCommandResult("The resources were successfully applied.", a.dryRun)
	if a.replay {
		return a.runReplay(cmd, objects)
	}
	return nil
}

func (a ApplyCmd) runReplay(cmd *cobra.Command, objects []manifest.Object) error {
	slos := filterByKind(objects, manifest.KindSLO)
	if a.dryRun {
		fmt.Printf("Skipping Replay. Found %d SLOs eligible for data import. (dry run)\n", len(slos))
		return nil
	}
	if len(slos) == 0 {
		fmt.Println("Skipping Replay. No SLOs were found in the applied resources.")
		return nil
	}
	replayCmd := ReplayCmd{client: a.client}
	replayCmd.arePlaylistEnabled(cmd.Context())
	replays := make([]ReplayConfig, 0, len(slos))
	for _, slo := range slos {
		replays = append(replays, ReplayConfig{
			Project: slo.GetProject(),
			SLO:     slo.GetName(),
			From:    a.replayFrom,
		})
	}
	failedReplays, err := replayCmd.RunReplays(cmd, replays)
	if err != nil || failedReplays > 0 {
		fmt.Println("Warning! Applied changes are not rolled back when Replay fails." +
			" Once you've fixed all related issues, we recommend using the 'sloctl replay' command" +
			" to run Replay, or reapply the resources with the '--replay' flag.")
	}
	return err
}

func filterByKind(objects []manifest.Object, kind manifest.Kind) []v1alpha.GenericObject {
	var filtered []v1alpha.GenericObject
	for i := range objects {
		v, ok := objects[i].(v1alpha.GenericObject)
		if ok && v.GetKind() == kind {
			filtered = append(filtered, v)
		}
	}
	return filtered
}
