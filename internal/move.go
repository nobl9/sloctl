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
		Short: "Move SLOs between Projects or Services",
		Long:  "Move SLOs to another Project or assign them to another Service within the same Project.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			move.client = r.GetClient()
		},
	}

	cmd.AddCommand(move.newMoveSLOCmd())
	return cmd
}

func (m *MoveCmd) newMoveSLOCmd() *cobra.Command {
	moveSubCmd := &cobra.Command{
		Use:   "slo [slo-name...]",
		Short: "Move SLOs to another Project or Service",
		Long: "Move the named SLOs from the Project selected by `--project` or the active configuration.\n" +
			"If no SLO names are provided, every SLO in the source Project is moved.\n\n" +
			"Use `--to-project` for a cross-Project move.\n" +
			"Use `--to-service` without `--to-project` to reassign SLOs within the source Project.\n" +
			"Missing target Projects and Services are created.\n" +
			"For a cross-Project move without `--to-service`, each SLO retains its source Service name.\n\n" +
			"**Cross-Project moves:**\n\n" +
			"- Change SLO links, so previous links no longer work.\n" +
			"- Remove SLOs from reports filtered by their previous paths.\n" +
			"- Fail when SLOs have attached Alert Policies unless you detach the policies\n" +
			"  manually or with `--detach-alert-policies`.\n" +
			"- Can make moved SLOs inaccessible to users without access to the target Project.\n\n" +
			"Nobl9 updates references to moved SLOs.\n" +
			"Update local SLO-as-code definitions that reference moved SLOs in composite SLOs\n" +
			"or Budget Adjustment filters.",
		Example: moveSLOExample,
		RunE:    m.moveSLO,
	}

	const toProjectFlagName = "to-project"
	moveSubCmd.Flags().StringVarP(
		&m.oldProject,
		"project",
		"p",
		"",
		"Source Project. Defaults to the Project in the active configuration.",
	)
	moveSubCmd.Flags().StringVarP(
		&m.newProject,
		toProjectFlagName,
		"",
		"",
		"Target Project for a cross-Project move. Omit when moving within the source Project.",
	)
	moveSubCmd.Flags().StringVarP(
		&m.newService,
		"to-service",
		"",
		"",
		"Target Service. Required for same-Project moves; for cross-Project moves, "+
			"the source Service name is used when omitted.",
	)
	moveSubCmd.Flags().BoolVarP(
		&m.detachAlertPolicies,
		"detach-alert-policies",
		"",
		false,
		"Detach all Alert Policies from moved SLOs during a cross-Project move.",
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
			fmt.Fprintf(&buf, "Moving '%s' SLO from '%s' Project to '%s' Project.\n",
				sloNames[0], oldProject, m.newProject)
		} else {
			fmt.Fprintf(&buf, "Moving '%s' SLO to a different Service within '%s' Project.\n",
				sloNames[0], oldProject)
		}
	default:
		if isCrossProjectMove {
			fmt.Fprintf(&buf, "Moving the following SLOs from '%s' Project to '%s' Project:\n",
				oldProject, m.newProject)
		} else {
			fmt.Fprintf(&buf, "Moving the following SLOs to a different Service within '%s' Project:\n",
				oldProject)
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
		fmt.Fprintf(&buf, "'%s' Service in '%s' Project will be assigned to all the moved SLOs.\n",
			m.newService, targetProject)
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
