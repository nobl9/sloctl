package internal

import (
	"github.com/mitchellh/colorstring"
	"github.com/spf13/cobra"
)

// AddListCommand returns cobra command list, which allows to list all queued Replays.
func (r *ReplayCmd) AddListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all Replays",
		RunE: func(cmd *cobra.Command, args []string) error {
			return r.listAllReplays(cmd)
		},
	}
	return cmd
}

func (r *ReplayCmd) listAllReplays(cmd *cobra.Command) error {
	cmd.Println(colorstring.Color("[yellow]Listing all Replays[reset]"))

	replayQueueList, err := r.client.Replay().V1().List(cmd.Context())
	if err != nil {
		return err
	}

	if len(replayQueueList) == 0 {
		cmd.Println(colorstring.Color("[light_gray]Replay not found[reset]"))
		return nil
	}
	return r.printer.Print(replayQueueList)
}
