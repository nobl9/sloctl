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

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/nobl9/nobl9-go/sdk/models"
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
		Short: "Move SLOs between Projects.",
		Long: `Moves one or more SLOs to a different project.
The command will also create a new Project or Service if the specified target does not yet exist.

Moving an SLO between Projects:
  - Updates its link — the former link won't work anymore.
  - Removes it from reports filtered by its previous path.
  - Unlinks Alert Policies (only if --detach-alert-policies flag is provided).
  - Updates SLO’s parent project in the composite definition.`,
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
		`Target Project for the moved SLOs.`,
	)
	moveSubCmd.Flags().StringVarP(
		&m.newService,
		"to-service",
		"",
		"",
		`Target Service for the moved SLOs (if not specified, the source Service name will be used).`,
	)
	moveSubCmd.Flags().BoolVarP(
		&m.detachAlertPolicies,
		"detach-alert-policies",
		"",
		false,
		`Detach all Alert Policies from the moved SLOs.`,
	)
	if err := moveSubCmd.MarkFlagRequired(toProjectFlagName); err != nil {
		panic(err)
	}

	return moveSubCmd
}

func (m *MoveCmd) moveSLO(cmd *cobra.Command, sloNames []string) error {
	ctx := cmd.Context()
	if m.oldProject != "" {
		m.client.Config.Project = m.oldProject
	}
	oldProject := m.client.Config.Project

	if len(sloNames) == 0 {
		var err error
		sloNames, err = m.getSLONamesForProject(ctx, oldProject)
		if err != nil {
			return err
		}
	}

	buf := bytes.Buffer{}
	switch len(sloNames) {
	case 0:
		return errors.Errorf("found no SLOs in '%s' Project", oldProject)
	case 1:
		buf.WriteString(fmt.Sprintf("Moving '%s' SLO from '%s' Project to '%s' Project.\n",
			sloNames[0], oldProject, m.newProject))
	default:
		buf.WriteString(fmt.Sprintf("Moving the following SLOs from '%s' Project to '%s' Project:\n",
			oldProject, m.newProject))
		for _, sloName := range sloNames {
			buf.WriteString(" - ")
			buf.WriteString(sloName)
			buf.WriteString("\n")
		}
	}
	if m.newService != "" {
		buf.WriteString(fmt.Sprintf("'%s' Service in '%s' Project will be assigned to all the moved SLOs.\n",
			m.newService, m.newProject))
	}
	buf.WriteString("If the target Service in the new Project does not exist, it will be copied.\n")
	if m.detachAlertPolicies {
		buf.WriteString("Attached Alert Policies will be detached from all the moved SLOs.\n")
	}
	_, _ = m.out.Write(buf.Bytes())

	return m.client.Objects().V1().MoveSLOs(ctx, models.MoveSLOs{
		SLONames:            sloNames,
		OldProject:          oldProject,
		NewProject:          m.newProject,
		Service:             m.newService,
		DetachAlertPolicies: m.detachAlertPolicies,
	})
}

func (m *MoveCmd) getSLONamesForProject(ctx context.Context, project string) ([]string, error) {
	_, _ = m.out.Write([]byte(fmt.Sprintf("Fetching all SLOs from '%s' Project...\n", project)))
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
