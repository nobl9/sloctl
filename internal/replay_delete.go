package internal

import (
	"fmt"

	"github.com/spf13/cobra"
)

// AddDeleteCommand returns cobra command delete, allows to delete a replay from a queue.
func (r *ReplayCmd) AddDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <slo-name>",
		Short: "Delete a replay from a queue",
		Long:  "Delete a replay from a queue.",
		Args:  r.deleteArguments,
		RunE: func(cmd *cobra.Command, args []string) error {
			if r.all {
				return r.deleteAllReplays()
			} else {
				return r.deleteReplaysForSLO(r.sloName, r.project)
			}
		},
	}

	cmd.Flags().StringVarP(&r.project, "project", "p", "",
		`Specifies the Project of the SLO you want to remove Replays from queue for.`)
	cmd.Flags().BoolVar(&r.all, "all", false, "Delete ALL replays in queue.")

	return cmd
}

func (r *ReplayCmd) deleteArguments(cmd *cobra.Command, args []string) error {
	if !r.all && len(args) == 0 {
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

func (r *ReplayCmd) deleteAllReplays() error {
	fmt.Printf("Deleting all replays from queue\n")
	return nil
}

func (r *ReplayCmd) deleteReplaysForSLO(sloName, project string) error {
	fmt.Printf("Deleting replays from a queue for SLO %s in project %s\n", sloName, project)
	return nil
}
