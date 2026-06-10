package internal

import (
	"fmt"

	"github.com/mitchellh/colorstring"
	replayV1 "github.com/nobl9/nobl9-go/sdk/endpoints/replay/v1"
	"github.com/spf13/cobra"
)

// AddCancelCommand returns cobra command delete, which allows to cancel running Replay.
func (r *ReplayCmd) AddCancelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel <slo-name>",
		Short: "Cancel an importing Replay",
		Args:  r.cancelArguments,
		RunE: func(cmd *cobra.Command, args []string) error {
			if r.project != "" {
				r.client.Config.Project = r.project
			}

			return r.cancelReplaysForSLO(cmd, r.sloName)
		},
	}

	cmd.Flags().StringVarP(&r.project, "project", "p", "",
		`Specifies the Project of the SLO you want to cancel importing Replay for.`)

	return cmd
}

func (r *ReplayCmd) cancelArguments(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		_ = cmd.Usage()
		return errReplayCancelInvalidOptions
	}
	if len(args) > 1 {
		return errReplayCancelTooManyArgs
	}
	if len(args) == 1 {
		r.sloName = args[0]
	}
	return nil
}

func (r *ReplayCmd) cancelReplaysForSLO(cmd *cobra.Command, sloName string) error {
	cmd.Println(
		colorstring.Color(
			fmt.Sprintf(
				"[yellow]Canceling importing phase of Replay for SLO '%s' in project '%s'[reset]",
				sloName,
				r.client.Config.Project,
			),
		),
	)

	err := r.client.Replay().V1().Cancel(
		cmd.Context(),
		replayV1.CancelRequest{
			Project: r.client.Config.Project,
			SLO:     sloName,
		})
	if err != nil {
		return err
	}

	cmd.Println(
		colorstring.Color(
			fmt.Sprintf(
				"[green]Cancellation of Replay for SLO '%s' in project '%s' successfully requested.[reset]",
				sloName,
				r.client.Config.Project,
			),
		),
	)

	return nil
}
