package internal

import (
	"fmt"

	"github.com/mitchellh/colorstring"
	replayV1 "github.com/nobl9/nobl9-go/sdk/endpoints/replay/v1"
	"github.com/spf13/cobra"
)

// AddDeleteCommand returns cobra command delete, which allows to delete a queued Replay.
func (r *ReplayCmd) AddDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [slo-name]",
		Short: "Remove queued Replays",
		Long: "Remove one queued Replay, or remove every queued Replay across all Projects\n" +
			"with `--all`. When an SLO name is provided, the Project defaults to the\n" +
			"active context's Project. This command does not cancel a Replay that is already\n" +
			"importing data.",
		Example: "sloctl replay delete my-slo --project my-project\nsloctl replay delete --all",
		Args:    r.deleteArguments,
		RunE: func(cmd *cobra.Command, args []string) error {
			if r.project != "" {
				r.client.Config.Project = r.project
			}
			if r.deleteAll {
				return r.deleteAllReplays(cmd)
			} else {
				return r.deleteReplaysForSLO(cmd, r.sloName)
			}
		},
	}

	cmd.Flags().StringVarP(&r.project, "project", "p", "",
		"Project containing the SLO. Defaults to the active context's Project.")
	cmd.Flags().BoolVar(
		&r.deleteAll,
		"all",
		false,
		"Remove all queued Replays across all Projects.",
	)

	return cmd
}

func (r *ReplayCmd) deleteArguments(cmd *cobra.Command, args []string) error {
	if !r.deleteAll && len(args) == 0 {
		_ = cmd.Usage()
		return errReplayDeleteInvalidOptions
	}
	if len(args) > 1 {
		return errReplayDeleteTooManyArgs
	}
	if len(args) == 1 {
		r.sloName = args[0]
	}
	return nil
}

func (r *ReplayCmd) deleteAllReplays(cmd *cobra.Command) error {
	cmd.Println(colorstring.Color("[yellow]Deleting all queued Replays[reset]"))

	err := r.client.Replay().V1().Delete(
		cmd.Context(),
		replayV1.DeleteRequest{
			All: true,
		},
	)
	if err != nil {
		return err
	}

	cmd.Println(colorstring.Color("[green]All queued Replays deleted successfully[reset]"))

	return nil
}

func (r *ReplayCmd) deleteReplaysForSLO(cmd *cobra.Command, sloName string) error {
	cmd.Println(
		colorstring.Color(
			fmt.Sprintf(
				"[yellow]Deleting queued Replay for SLO '%s' in project '%s'[reset]",
				sloName,
				r.client.Config.Project,
			),
		),
	)

	err := r.client.Replay().V1().Delete(
		cmd.Context(),
		replayV1.DeleteRequest{
			Project: r.client.Config.Project,
			SLO:     sloName,
		},
	)
	if err != nil {
		return err
	}

	cmd.Println(
		colorstring.Color(
			fmt.Sprintf("[green]Queued Replays for SLO '%s' in project '%s' deleted successfully[reset]",
				sloName,
				r.client.Config.Project,
			),
		),
	)

	return nil
}
