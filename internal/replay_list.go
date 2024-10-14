package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/mitchellh/colorstring"
	"github.com/nobl9/sloctl/internal/printer"
	"github.com/spf13/cobra"
)

// AddListCommand returns cobra command list, which allows to list all queued Replays.
func (r *ReplayCmd) AddListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List queued Replays",
		RunE: func(cmd *cobra.Command, args []string) error {
			return r.listAllReplays(cmd)
		},
	}
	return cmd
}

type ReplayList struct {
	Slo            string `json:"slo,omitempty"`
	Project        string `json:"project"`
	ElapsedTime    string `json:"elapsedTime,omitempty"`
	RetrievedScope string `json:"retrievedScope,omitempty"`
	RetrievedFrom  string `json:"retrievedFrom,omitempty"`
	Status         string `json:"status"`
}

func (r *ReplayCmd) listAllReplays(cmd *cobra.Command) error {
	cmd.Println(colorstring.Color("[yellow]Listing all queued Replays[reset]"))

	response, err := r.doRequest(
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

	var replayList []ReplayList
	if err := json.Unmarshal(response, &replayList); err != nil {
		fmt.Printf("err: %v\n", err)
	}

	p, err := printer.New(os.Stdout, "yaml", "|", "-")
	if err != nil {
		return err
	}
	if err = p.Print(replayList); err != nil {
		return err
	}
	return nil
}
