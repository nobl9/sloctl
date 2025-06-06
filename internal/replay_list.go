package internal

import (
	"encoding/json"
	"fmt"
	"net/http"

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

type ReplayListItem struct {
	Slo            string `json:"slo,omitempty"`
	Project        string `json:"project"`
	ElapsedTime    string `json:"elapsedTime,omitempty"`
	RetrievedScope string `json:"retrievedScope,omitempty"`
	RetrievedFrom  string `json:"retrievedFrom,omitempty"`
	Status         string `json:"status"`
	Cancellation   string `json:"cancellation,omitempty"`
}

func (r *ReplayCmd) listAllReplays(cmd *cobra.Command) error {
	cmd.Println(colorstring.Color("[yellow]Listing all Replays[reset]"))

	response, _, err := r.doRequest(
		cmd.Context(),
		http.MethodGet,
		endpointReplayList,
		"",
		nil,
		nil,
	)
	if err != nil {
		return err
	}

	var replayQueueList []ReplayListItem
	if err := json.Unmarshal(response, &replayQueueList); err != nil {
		fmt.Printf("err: %v\n", err)
	}

	if len(replayQueueList) == 0 {
		cmd.Println(colorstring.Color("[light_gray]Replay not found[reset]"))
		return nil
	}
	return r.printer.Print(replayQueueList)
}
