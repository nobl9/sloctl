package internal

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/mitchellh/colorstring"
	"github.com/spf13/cobra"

	"github.com/nobl9/nobl9-go/sdk"
)

var (
	errReviewTooManyArgs    = errors.New("command accepts only single SLO name as an argument")
	errReviewInvalidOptions = errors.New("you must provide the SLO name as an argument")
)

type ReviewCmd struct {
	client  *sdk.Client
	project string
	status  string
	note    string
	sloName string
}

//go:embed review_example.sh
var reviewExample string

func (r *RootCmd) NewReviewCmd() *cobra.Command {
	review := &ReviewCmd{}

	cmd := &cobra.Command{
		Use:   "review",
		Short: "Manage SLO review",
		Long: `Manage SLO review.

This feature requires Nobl9 Enterprise Edition.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			review.client = r.GetClient()
		},
	}

	cmd.AddCommand(review.NewSetStatusCmd())

	return cmd
}

func (r *ReviewCmd) NewSetStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-status",
		Short: "Change an SLO review status",
		Long: "Change the review status of one SLO by selecting a status subcommand.\n" +
			"The Project defaults to the active context's Project. The `reviewed` and `skipped`\n" +
			"statuses accept an optional `--note`.\n\n" +
			"See [SLO review status transitions]" +
			"(https://docs.nobl9.com/slo-oversight/reviews/#status-transitions) " +
			"for allowed manual and automatic transitions.",
	}

	cmd.AddCommand(r.NewSetStatusReviewedCmd())
	cmd.AddCommand(r.NewSetStatusSkippedCmd())
	cmd.AddCommand(r.NewSetStatusToReviewCmd())
	cmd.AddCommand(r.NewSetStatusOverdueCmd())
	cmd.AddCommand(r.NewSetStatusNotStartedCmd())

	return cmd
}

func (r *ReviewCmd) NewSetStatusReviewedCmd() *cobra.Command {
	return r.newSetStatusCmd("reviewed", "reviewed", "Mark an SLO as reviewed", true)
}

func (r *ReviewCmd) NewSetStatusSkippedCmd() *cobra.Command {
	return r.newSetStatusCmd("skipped", "skipped", "Mark an SLO review as skipped", true)
}

func (r *ReviewCmd) NewSetStatusToReviewCmd() *cobra.Command {
	return r.newSetStatusCmd("to-review", "toReview", "Mark an SLO as awaiting review", false)
}

func (r *ReviewCmd) NewSetStatusOverdueCmd() *cobra.Command {
	return r.newSetStatusCmd("overdue", "overdue", "Mark an SLO review as overdue", false)
}

func (r *ReviewCmd) NewSetStatusNotStartedCmd() *cobra.Command {
	return r.newSetStatusCmd("not-started", "notStarted", "Reset an SLO review to not started", false)
}

func (r *ReviewCmd) newSetStatusCmd(commandName, status, short string, hasNote bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:     commandName + " <slo-name>",
		Short:   short,
		Long:    setStatusLongDescription(status, hasNote),
		Example: reviewExample,
		Args:    r.reviewSetArguments,
		RunE: func(cmd *cobra.Command, args []string) error {
			if r.project == "" {
				r.project = r.client.Config.Project
			}
			r.status = status
			return r.runSetStatusReview(cmd, r.sloName)
		},
	}

	cmd.Flags().StringVarP(&r.project, "project", "p", "",
		"Project containing the SLO. Defaults to the active context's Project.")

	if hasNote {
		cmd.Flags().StringVarP(&r.note, "note", "n", "",
			"Note to attach to the review decision.")
	}

	return cmd
}

func setStatusLongDescription(status string, includeNote bool) string {
	desc := fmt.Sprintf(
		"Set one SLO's review status to `%s`.\nThe Project defaults to the active context's Project.",
		status,
	)
	switch status {
	case "notStarted":
		desc += "\nAvailable only when the SLO's Service has no review schedule."
	case "reviewed":
		desc += "\nAvailable whether or not the SLO's Service has a review schedule."
	case "toReview", "skipped", "overdue":
		desc += "\nAvailable only when the SLO's Service has a review schedule."
	}
	if includeNote {
		desc += "\nUse `--note` to attach context to the decision."
	}
	return desc + "\n\nThis feature requires Nobl9 Enterprise Edition."
}

func (r *ReviewCmd) reviewSetArguments(cmd *cobra.Command, args []string) error {
	switch len(args) {
	case 0:
		return errReviewInvalidOptions
	case 1:
		r.sloName = args[0]
		return nil
	default:
		return errReviewTooManyArgs
	}
}

func (r *ReviewCmd) runSetStatusReview(cmd *cobra.Command, sloName string) error {
	ctx := cmd.Context()

	if err := r.doSetReviewRequest(sloName, ctx); err != nil {
		return err
	}

	cmd.Println(
		colorstring.Color(
			fmt.Sprintf(
				"[green]Successfully set review status to '%s' for SLO '%s' in project '%s'.\n[reset]",
				r.status,
				sloName,
				r.project,
			),
		),
	)

	return nil
}

func (r *ReviewCmd) doSetReviewRequest(sloName string, ctx context.Context) error {
	type reviewRequest struct {
		Status string `json:"status"`
		Note   string `json:"note,omitempty"`
	}
	data, err := json.Marshal(reviewRequest{Status: r.status, Note: r.note})
	if err != nil {
		return fmt.Errorf("failed to encode review request: %w", err)
	}

	endpoint := fmt.Sprintf("/objects/v1/slos/%s/review", sloName)
	header := http.Header{sdk.HeaderProject: []string{r.project}}

	req, err := r.client.CreateRequest(ctx, http.MethodPost, endpoint, header, nil, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create review request: %w", err)
	}

	resp, err := r.client.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute review request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		errMsg := fmt.Sprintf("review request failed (HTTP status: %d):", resp.StatusCode)
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, body, "", "  "); err != nil {
			return fmt.Errorf("%s %s", errMsg, string(body))
		}
		return fmt.Errorf("%s %s", errMsg, pretty.String())
	}

	return nil
}
