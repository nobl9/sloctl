package internal

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
	objectsV1 "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type MoveCmd struct {
	client              *sdk.Client
	oldProject          string
	newService          string
	newProject          string
	detachAlertPolicies bool
	out                 io.Writer
}

//go:embed move_slo_example.sh
var moveSLOExample string

func (r *RootCmd) NewMoveCmd() *cobra.Command {
	move := &MoveCmd{out: os.Stderr}

	cmd := &cobra.Command{
		Use:   "move",
		Short: "Move objects between Projects.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			move.client = r.GetClient()
		},
	}

	cmd.AddCommand(move.newMoveSLOCmd())
	return cmd
}

func (m *MoveCmd) newMoveSLOCmd() *cobra.Command {
	moveSubCmd := &cobra.Command{
		Use:   "slo",
		Short: "Move SLOs between Projects or to a different Service within the same Project.",
		Long: `Moves one or more SLOs to a different project or to a different Service within the same project.
The command will also create a new Project and/or Service if the specified target objects do not yet exist.

For cross-project moves, use --to-project flag. For same-project service moves, use --to-service without --to-project.

Moving an SLO between Projects updates references of this SLO in other objects.
If you've adopted SLOs as Code approach, ensure you update these references in your configuration:
  - Component SLO's project in the composite SLO definition.
  - Budget Adjustment filters.

Furthermore, cross-project move operations:
  - Update SLO links — former links won't work anymore.
  - Remove SLOs from reports filtered by their previous path.
  - Unlink Alert Policies (only if --detach-alert-policies flag is provided).`,
		Example: moveSLOExample,
		RunE:    m.moveSLO,
	}

	const toProjectFlagName = "to-project"
	moveSubCmd.Flags().StringVarP(
		&m.oldProject,
		"project",
		"p",
		"",
		`Source Project of the moved SLOs.`,
	)
	moveSubCmd.Flags().StringVarP(
		&m.newProject,
		toProjectFlagName,
		"",
		"",
		`Target Project for the moved SLOs (required for cross-project moves, omit for same-project service moves).`,
	)
	moveSubCmd.Flags().StringVarP(
		&m.newService,
		"to-service",
		"",
		"",
		"Target Service for the moved SLOs (required for same-project moves; if not specified for cross-project "+
			"moves, source Service name is used).",
	)
	moveSubCmd.Flags().BoolVarP(
		&m.detachAlertPolicies,
		"detach-alert-policies",
		"",
		false,
		`Detach all Alert Policies from the moved SLOs.`,
	)

	return moveSubCmd
}

func (m *MoveCmd) moveSLO(cmd *cobra.Command, sloNames []string) error {
	ctx := cmd.Context()
	if m.oldProject != "" {
		m.client.Config.Project = m.oldProject
	}
	oldProject := m.client.Config.Project

	isCrossProjectMove := m.newProject != ""
	isSameProjectMove := m.newProject == "" && m.newService != ""

	if !isCrossProjectMove && !isSameProjectMove {
		return errors.New("Either --to-project or --to-service must be provided.")
	}

	if isSameProjectMove && m.detachAlertPolicies {
		return errors.New("The --detach-alert-policies flag is only applicable for cross-project moves. " +
			"Alert Policies remain valid when moving SLOs within the same Project.")
	}

	if len(sloNames) == 0 {
		var err error
		sloNames, err = m.getSLONamesForProject(ctx, oldProject)
		if err != nil {
			return err
		}
	}
	if len(sloNames) == 0 {
		return errors.Errorf("Found no SLOs in '%s' Project.", oldProject)
	}

	payload := objectsV1.MoveSLOsRequest{
		SLONames:            sloNames,
		OldProject:          oldProject,
		NewProject:          m.newProject,
		Service:             m.newService,
		DetachAlertPolicies: m.detachAlertPolicies,
	}
	if err := payload.Validate(); err != nil {
		return err
	}

	m.printMoveDetails(sloNames, isCrossProjectMove, oldProject)

	if err := m.client.Objects().V1().MoveSLOs(ctx, payload); err != nil {
		_, _ = m.out.Write([]byte("\n"))
		var httpErr *sdk.HTTPError
		if errors.As(err, &httpErr) {
			if len(httpErr.Errors) > 0 && strings.Contains(httpErr.Errors[0].Title, "it has assigned Alert Policies") {
				return errors.New("Cannot move SLOs with attached Alert Policies.\n" +
					"Detach them manually or use the '--detach-alert-policies' flag to detach them automatically.")
			}
		}
		return err
	}
	_, _ = m.out.Write([]byte("\nThe SLOs were successfully moved.\n"))
	return nil
}

func (m *MoveCmd) getSLONamesForProject(ctx context.Context, project string) ([]string, error) {
	_, _ = fmt.Fprintf(m.out, "Fetching all SLOs from '%s' Project...\n", project)
	slos, err := m.client.Objects().V1().Get(
		ctx,
		manifest.KindSLO,
		http.Header{sdk.HeaderProject: []string{project}},
		nil,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch all SLOs for '%s' Project", project)
	}
	sloNames := make([]string, 0, len(slos))
	for _, slo := range slos {
		sloNames = append(sloNames, slo.GetName())
	}
	slices.Sort(sloNames)
	return sloNames, nil
}

func (m *MoveCmd) printMoveDetails(sloNames []string, isCrossProjectMove bool, oldProject string) {
	buf := bytes.Buffer{}
	switch len(sloNames) {
	case 1:
		if isCrossProjectMove {
			buf.WriteString(fmt.Sprintf("Moving '%s' SLO from '%s' Project to '%s' Project.\n",
				sloNames[0], oldProject, m.newProject))
		} else {
			buf.WriteString(fmt.Sprintf("Moving '%s' SLO to a different Service within '%s' Project.\n",
				sloNames[0], oldProject))
		}
	default:
		if isCrossProjectMove {
			buf.WriteString(fmt.Sprintf("Moving the following SLOs from '%s' Project to '%s' Project:\n",
				oldProject, m.newProject))
		} else {
			buf.WriteString(fmt.Sprintf("Moving the following SLOs to a different Service within '%s' Project:\n",
				oldProject))
		}
		for _, sloName := range sloNames {
			buf.WriteString(" - ")
			buf.WriteString(sloName)
			buf.WriteString("\n")
		}
	}
	if m.newService != "" {
		targetProject := m.newProject
		if !isCrossProjectMove {
			targetProject = oldProject
		}
		buf.WriteString(fmt.Sprintf("'%s' Service in '%s' Project will be assigned to all the moved SLOs.\n",
			m.newService, targetProject))
	}
	if isCrossProjectMove {
		buf.WriteString("If the target Service in the new Project does not exist, it will be created.\n")
	} else {
		buf.WriteString("If the target Service does not exist in this Project, it will be created.\n")
	}
	if m.detachAlertPolicies {
		buf.WriteString("Attached Alert Policies will be detached from all the moved SLOs.\n")
	}
	_, _ = m.out.Write(buf.Bytes())
}
