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
	"slices"
	"strings"

	"github.com/mitchellh/colorstring"
	"github.com/spf13/cobra"

	"github.com/nobl9/nobl9-go/sdk"
)

var (
	errReviewTooManyArgs    = errors.New("'review set' command accepts only single SLO name as an argument")
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

	cmd.AddCommand(review.NewSetCmd())

	return cmd
}

// NewSetCmd creates the set subcommand for reviews
func (r *ReviewCmd) NewSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <SLO_NAME>",
		Short: "Set SLO review status (Enterprise Edition only)",
		Long: `Set SLO review status.

This command allows you to update the review status of a specific SLO within a project.
Available statuses are: reviewed, skipped, pending, overdue, notStarted.

When setting status to 'reviewed' or 'skipped', you can optionally include a note
using the --note flag to provide additional context or reasoning for the review decision.

The SLO name must be provided as the first argument, and the project can be specified
using the --project flag or will default to the configured project in your client.

Note: This feature is only available in Enterprise Edition tier.`,
		Example: reviewExample,
		Args:    r.reviewSetArguments,
		RunE: func(cmd *cobra.Command, args []string) error {
			if r.project == "" {
				r.project = r.client.Config.Project
			}
			return r.runSetReview(cmd, r.sloName)
		},
	}

	cmd.Flags().StringVar(&r.status, "status", "",
		"Review status: reviewed, skipped, pending, overdue, notStarted (required)")
	cmd.Flags().StringVarP(&r.project, "project", "p", "",
		"Project name")
	cmd.Flags().StringVarP(&r.note, "note", "n", "",
		"Optional note annotation (only applicable for reviewed and skipped statuses)")

	_ = cmd.MarkFlagRequired("status")

	return cmd
}

// ReviewRequest represents the API request structure for review endpoint
type ReviewRequest struct {
	Status string `json:"status"`
	Note   string `json:"note,omitempty"`
}

func (r *ReviewCmd) reviewSetArguments(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errReviewInvalidOptions
	}
	if len(args) > 1 {
		return errReviewTooManyArgs
	}
	if len(args) == 1 {
		r.sloName = args[0]
	}

	if err := r.validateArguments(); err != nil {
		return err
	}

	return nil
}

func (r *ReviewCmd) validateArguments() error {
	validStatuses := []string{"reviewed", "skipped", "pending", "overdue", "notStarted"}
	if !slices.Contains(validStatuses, r.status) {
		return fmt.Errorf("invalid status '%s': must be one of: %s", r.status, strings.Join(validStatuses, ", "))
	}

	if r.note != "" && r.status != "reviewed" && r.status != "skipped" {
		return fmt.Errorf("note annotation is only applicable for reviewed and skipped statuses")
	}

	return nil
}

func (r *ReviewCmd) runSetReview(cmd *cobra.Command, sloName string) error {
	ctx := context.Background()

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
	// Create request payload
	reviewReq := ReviewRequest{
		Status: r.status,
		Note:   r.note,
	}

	// Encode payload
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(reviewReq); err != nil {
		return fmt.Errorf("failed to encode review request: %w", err)
	}

	endpoint := fmt.Sprintf("/objects/v1/slos/%s/review", sloName)
	header := http.Header{sdk.HeaderProject: []string{r.project}}

	req, err := r.client.CreateRequest(ctx, http.MethodPost, endpoint, header, nil, buf)
	if err != nil {
		return fmt.Errorf("failed to create review request: %w", err)
	}

	resp, err := r.client.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute review request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, respBody, "", "  "); err != nil {
			return fmt.Errorf("review request failed (status: %d): %s", resp.StatusCode, string(respBody))
		}
		return fmt.Errorf("review request failed (status: %d): %s", resp.StatusCode, prettyJSON.String())
	}

	return nil
}
