package internal

import (
	"fmt"
	"net/http"

	"github.com/mitchellh/colorstring"
	"github.com/spf13/cobra"
)

// AddDeleteCommand returns cobra command delete, which allows to delete a queued Replay.
func (r *ReplayCmd) AddDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <slo-name>",
		Short: "Delete a queued Replay",
		Args:  r.deleteArguments,
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
		`Specifies the Project of the SLO you want to remove queued Replays for.`)
	cmd.Flags().BoolVar(&r.deleteAll, "all", false, "Delete ALL queued Replays.")

	return cmd
}

func (r *ReplayCmd) deleteArguments(cmd *cobra.Command, args []string) error {
	if !r.deleteAll && len(args) == 0 {
		_ = cmd.Usage()
		return errReplayDeleteInvalidOptions
	}
	if len(args) > 1 {
		return errReplayTooManyArgs
	}
	if len(args) == 1 {
		r.sloName = args[0]
	}
	return nil
}

type deleteReplayRequest struct {
	Project string `json:"project,omitempty"`
	Slo     string `json:"slo,omitempty"`
	All     bool   `json:"all,omitempty"`
}

func (r *ReplayCmd) deleteAllReplays(cmd *cobra.Command) error {
	cmd.Println(colorstring.Color("[yellow]Deleting all queued Replays[reset]"))

	_, _, err := r.doRequest(
		cmd.Context(),
		http.MethodDelete,
		endpointReplayDelete,
		"",
		nil,
		deleteReplayRequest{
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
		colorstring.Printf(
			"[yellow]Deleting queued Replays for SLO '%s' in project '%s'[reset]",
			sloName,
			r.client.Config.Project,
		))

	_, _, err := r.doRequest(
		cmd.Context(),
		http.MethodDelete,
		endpointReplayDelete,
		r.client.Config.Project,
		nil,
		deleteReplayRequest{
			Project: r.client.Config.Project,
			Slo:     sloName,
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
