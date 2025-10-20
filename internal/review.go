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

// NewReviewCmd returns cobra command review with all its flags and subcommands.
func (r *RootCmd) NewReviewCmd() *cobra.Command {
	review := &ReviewCmd{}

	cmd := &cobra.Command{
		Use:   "review",
		Short: "Manage SLO review (Enterprise Edition only)",
		Long: `Manage SLO review.

Note: This feature is only available in Enterprise Edition tier.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			review.client = r.GetClient()
		},
	}

	cmd.AddCommand(review.NewSetStatusCmd())

	return cmd
}

// NewSetStatusCmd creates the set-status parent command
func (r *ReviewCmd) NewSetStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-status",
		Short: "Set SLO review status",
		Long: `Set SLO review status.

This command allows you to update the review status of a specific SLO within a project.

Note: This feature is only available in Enterprise Edition tier.`,
	}

	cmd.AddCommand(r.NewSetStatusReviewedCmd())
	cmd.AddCommand(r.NewSetStatusSkippedCmd())
	cmd.AddCommand(r.NewSetStatusToReviewCmd())
	cmd.AddCommand(r.NewSetStatusOverdueCmd())
	cmd.AddCommand(r.NewSetStatusNotStartedCmd())

	return cmd
}

// NewSetStatusReviewedCmd creates the reviewed subcommand
func (r *ReviewCmd) NewSetStatusReviewedCmd() *cobra.Command {
	return r.newSetStatusCmd("reviewed", "reviewed", true)
}

// NewSetStatusSkippedCmd creates the skipped subcommand
func (r *ReviewCmd) NewSetStatusSkippedCmd() *cobra.Command {
	return r.newSetStatusCmd("skipped", "skipped", true)
}

// NewSetStatusToReviewCmd creates the to-review subcommand
func (r *ReviewCmd) NewSetStatusToReviewCmd() *cobra.Command {
	return r.newSetStatusCmd("to-review", "toReview", false)
}

// NewSetStatusOverdueCmd creates the overdue subcommand
func (r *ReviewCmd) NewSetStatusOverdueCmd() *cobra.Command {
	return r.newSetStatusCmd("overdue", "overdue", false)
}

// NewSetStatusNotStartedCmd creates the not-started subcommand
func (r *ReviewCmd) NewSetStatusNotStartedCmd() *cobra.Command {
	return r.newSetStatusCmd("not-started", "notStarted", false)
}

func (r *ReviewCmd) newSetStatusCmd(commandName, status string, hasNote bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:     commandName + " <slo-name>",
		Short:   setStatusShortDescription(status),
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
		"Project name")

	if hasNote {
		cmd.Flags().StringVarP(&r.note, "note", "n", "",
			"Optional note annotation")
	}

	return cmd
}

func setStatusShortDescription(status string) string {
	return fmt.Sprintf("Set SLO review status to %s", status)
}

func setStatusLongDescription(status string, includeNote bool) string {
	const (
		noteLongDescription = `
You can optionally include a note using the --note flag to provide additional
context or reasoning for the review decision.
`
		projectLongDescription = `

The SLO name must be provided as the first argument, and the project can be specified
using the --project flag or will default to the configured project in your client.

Note: This feature is only available in Enterprise Edition tier.
`
	)

	desc := setStatusShortDescription(status)
	if includeNote {
		desc += noteLongDescription
	}
	desc += projectLongDescription
	return desc
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
