package internal

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/manifest/v1alpha"
	v1alphaAnnotation "github.com/nobl9/nobl9-go/manifest/v1alpha/annotation"
	"github.com/nobl9/nobl9-go/sdk"
	objectsV1 "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v1"
	objectsV2 "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v2"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/nobl9/sloctl/internal/collections"
	"github.com/nobl9/sloctl/internal/flags"
	"github.com/nobl9/sloctl/internal/printer"
)

//go:embed get_alert_example.sh
var getAlertExample string

//go:embed get_annotation_example.sh
var getAnnotationExample string

type GetCmd struct {
	client    *sdk.Client
	printer   *printer.Printer
	selection objectSelectionFlags
}

// NewGetCmd returns cobra command get with all flags for it.
func (r *RootCmd) NewGetCmd() *cobra.Command {
	get := &GetCmd{
		printer: printer.NewPrinter(printer.Config{}),
	}

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Display one or more than one resource",
		Long: `Prints a table of the most important information about the specified resources.
To get more details in output use one of the available flags.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			get.client = r.GetClient()
			if get.selection.allProjects {
				get.client.Config.Project = "*"
			} else if get.selection.project != "" {
				get.client.Config.Project = get.selection.project
			}
		},
	}

	// All shared flags for 'get' and its subcommands.
	get.printer.MustRegisterFlags(cmd)

	// All subcommands for get.
	for _, subCmd := range []struct {
		Kind     manifest.Kind
		Aliases  []string
		Extender func(cmd *cobra.Command) *cobra.Command
	}{
		{Kind: manifest.KindAgent, Extender: get.newGetAgentCommand},
		{Kind: manifest.KindAlertMethod},
		{Kind: manifest.KindAlertPolicy},
		{Kind: manifest.KindAlert, Extender: get.newGetAlertCommand},
		{Kind: manifest.KindAlertSilence},
		{Kind: manifest.KindAnnotation, Extender: get.newGetAnnotationCommand},
		{Kind: manifest.KindDataExport, Extender: get.newGetDataExportCommand},
		{Kind: manifest.KindDirect},
		{Kind: manifest.KindProject},
		{Kind: manifest.KindRoleBinding},
		{Kind: manifest.KindService, Aliases: aliasesForKind(manifest.KindService)},
		{Kind: manifest.KindSLO},
		{Kind: manifest.KindUserGroup},
		{Kind: manifest.KindBudgetAdjustment},
		{Kind: manifest.KindReport},
	} {
		plural := pluralForKind(subCmd.Kind)
		short := fmt.Sprintf("Displays all of the %s.", plural)
		use := strings.ToLower(plural)
		subCmd.Aliases = append(subCmd.Aliases, subCmd.Kind.ToLower(), subCmd.Kind.String(), plural)

		sc := get.newGetObjectsCommand(subCmd.Kind, short, use, subCmd.Aliases)
		if subCmd.Extender != nil {
			subCmd.Extender(sc)
		}
		registerObjectSelectionFlags(sc, subCmd.Kind, &get.selection,
			`List the requested object(s) across all projects.`)
		cmd.AddCommand(sc)
	}

	return cmd
}

func (g *GetCmd) newGetObjectsCommand(
	kind manifest.Kind,
	short, use string,
	aliases []string,
) *cobra.Command {
	return &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   short,
		RunE: func(cmd *cobra.Command, args []string) error {
			objects, err := g.getObjects(cmd.Context(), kind, args)
			if err != nil {
				return err
			}
			return g.printObjects(kind, objects)
		},
	}
}

// nolint: gocognit
func (g *GetCmd) newGetAlertCommand(cmd *cobra.Command) *cobra.Command {
	cmd.Example = getAlertExample
	cmd.Long = "Get alerts based on search criteria. You can use specific criteria using flags to find alerts " +
		"related to specific SLO, objective, service, alert policy, time range, or alert status.\n\n" +
		"For example, you can use the same flag multiple times to find alerts triggered for a given SLO OR " +
		"another SLO. Keep in mind that the different types of flags are linked by the logical AND operator.\n\n" +
		"If you don't have permission to view SLO in a given project, alerts from that project will not be returned.\n\n"

	params := objectsV1.GetAlertsRequest{
		Resolved:  new(bool),
		Triggered: new(bool),
	}
	cmd.Flags().StringArrayVar(
		&params.AlertPolicyNames,
		"alert-policy",
		[]string{},
		"Get alerts triggered for a given alert policy (name) only.",
	)
	cmd.Flags().StringArrayVar(
		&params.SLONames,
		"slo",
		[]string{},
		"Get alerts triggered for a given SLO (name) only.",
	)
	cmd.Flags().StringArrayVar(
		&params.ObjectiveNames,
		"objective",
		[]string{},
		"Get alerts triggered for a given objective name of the SLO only.",
	)
	cmd.Flags().StringArrayVar(
		&params.ServiceNames,
		"service",
		[]string{},
		"Get alerts triggered for SLOs related to a given service only.",
	)
	objectiveValuesFlag := flags.FloatArray{}
	cmd.Flags().Var(
		&objectiveValuesFlag,
		"objective-value",
		"Get alerts triggered for a given objective value of the SLO only.",
	)
	cmd.Flags().BoolVar(
		params.Resolved,
		"resolved",
		true,
		"Get alerts that are resolved only.",
	)
	cmd.Flags().BoolVar(
		params.Triggered,
		"triggered",
		true,
		"Get alerts that are still active (not resolved yet) only.",
	)
	flags.RegisterTimeVar(
		cmd,
		&params.From,
		"from",
		"Get active alerts after `from` time only, based on metric timestamp (RFC3339), "+
			"for example 2023-02-09T10:00:00Z.",
	)
	flags.RegisterTimeVar(
		cmd,
		&params.To,
		"to",
		"Get active alerts before `to` time only, based on metric timestamp (RFC3339), "+
			"for example 2023-02-09T10:00:00Z.",
	)

	cmd.Flags().SortFlags = false
	cmd.Flags().Lookup("objective-value").Hidden = true

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			params.Names = args
		}
		params.ObjectiveValues = objectiveValuesFlag

		//nolint: staticcheck
		alerts, truncatedMax, err := g.client.Objects().V1().GetAlerts(cmd.Context(), params)
		if err != nil {
			return err
		}
		if len(alerts) == 0 {
			fmt.Printf("No resources found in '%s' project.\n", g.client.Config.Project)
			return nil
		}
		if err = g.printer.Print(alerts); err != nil {
			return err
		}
		if truncatedMax > 0 {
			fmt.Fprintf(os.Stderr, "Warning: %d new alerts have been returned from the API according to the "+
				"provided searching criteria. Specify more restrictive filters (by SLO, objective, service, "+
				"alert policy, time range, or alert status) to get more limited results.\n", truncatedMax)
		}
		return nil
	}
	return cmd
}

func (g *GetCmd) newGetDataExportCommand(cmd *cobra.Command) *cobra.Command {
	displayExternalID := cmd.Flags().Bool(
		"aws-external-id",
		false,
		"Display AWS external id, which will be used by Nobl9 to assume the IAM role when performing data export",
	)
	err := cmd.Flags().MarkDeprecated(
		"aws-external-id", "use `sloctl aws-iam-ids dataexport` instead",
	)
	if err != nil {
		cmd.PrintErr(err)
	}

	wrap := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if *displayExternalID {
			id, err := g.client.AuthData().V1().GetDataExportIAMRoleIDs(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Println(id)
			return nil
		}
		return wrap(cmd, args)
	}
	return cmd
}

func (g *GetCmd) newGetAgentCommand(cmd *cobra.Command) *cobra.Command {
	withAccessKeysFlag := cmd.Flags().BoolP("with-keys", "k", false,
		`Displays client_secret and client_id.`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		objects, err := g.getObjects(cmd.Context(), manifest.KindAgent, args)
		if err != nil {
			return err
		}
		var agents []manifest.Object
		switch {
		case len(objects) > 0 && *withAccessKeysFlag:
			agentsWithSecrets, err := g.getAgentsWithSecrets(cmd.Context(), objects)
			if err != nil {
				return err
			}
			agents = make([]manifest.Object, 0, len(agentsWithSecrets))
			for _, agent := range agentsWithSecrets {
				agents = append(agents, agent)
			}
		default:
			agents = objects
		}
		return g.printObjects(manifest.KindAgent, agents)
	}
	return cmd
}

func (g *GetCmd) newGetAnnotationCommand(cmd *cobra.Command) *cobra.Command {
	cmd.Example = getAnnotationExample
	cmd.Long = fmt.Sprintf("Get annotations based on search criteria. "+
		"You can use specific criteria using flags to find annotations "+
		"related to specific project, SLO, time range, or categories.\n"+
		"By default only %s categories are returned.\n\n"+
		"Keep in mind that the different types of flags are linked by the logical AND operator.\n\n",
		strings.Join(stringsTypeToStrings(v1alphaAnnotation.GetUserCategories()), ", "))

	params := objectsV2.GetAnnotationsRequest{}
	var (
		categoriesFlag   []string
		userCategories   bool
		systemCategories bool
	)
	cmd.Flags().StringVar(
		&params.SLOName,
		"slo",
		"",
		"Get annotations for a given SLO (name) only.",
	)
	flags.RegisterTimeVar(
		cmd,
		&params.From,
		"from",
		"Get annotations which have 'spec.startTime' after or equal to the given time.",
	)
	flags.RegisterTimeVar(
		cmd,
		&params.To,
		"to",
		"Get annotations which have 'spec.endTime' before or equal to the given time.",
	)
	cmd.Flags().BoolVar(
		&userCategories,
		"user",
		false,
		"Get annotations which were created by user actions.",
	)
	cmd.Flags().BoolVar(
		&systemCategories,
		"system",
		false,
		"Get annotations which were automatically created by Nobl9 platform.",
	)
	cmd.Flags().StringArrayVar(
		&categoriesFlag,
		"category",
		nil,
		fmt.Sprintf(
			"Filter annotations by their category (one of: %s).",
			strings.Join(stringsTypeToStrings(v1alphaAnnotation.CategoryValues()), ", "),
		),
	)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			params.Names = args
		}
		for _, cat := range categoriesFlag {
			parsed, err := v1alphaAnnotation.ParseCategory(cat)
			if err != nil {
				return fmt.Errorf("invalid 'category' flag value: %w", err)
			}
			params.Categories = append(params.Categories, parsed)
		}
		if systemCategories {
			params.Categories = append(params.Categories, v1alphaAnnotation.GetSystemCategories()...)
		}
		if userCategories {
			params.Categories = append(params.Categories, v1alphaAnnotation.GetUserCategories()...)
		}
		if len(params.Categories) == 0 {
			params.Categories = v1alphaAnnotation.GetUserCategories()
		}
		params.Categories = collections.RemoveDuplicates(params.Categories)

		annotations, err := g.client.Objects().V2().GetV1alphaAnnotations(cmd.Context(), params)
		if err != nil {
			return err
		}
		if len(annotations) == 0 {
			fmt.Printf("No resources found in '%s' project.\n", g.client.Config.Project)
			return nil
		}
		return g.printer.Print(annotations)
	}
	return cmd
}

func (g *GetCmd) getAgentsWithSecrets(ctx context.Context, objects []manifest.Object) ([]v1alpha.GenericObject, error) {
	agents := make([]v1alpha.GenericObject, 0, len(objects))
	var mu sync.Mutex
	eg, ctx := errgroup.WithContext(ctx)
	for i := range objects {
		eg.Go(func() error {
			agent, ok := objects[i].(v1alpha.GenericObject)
			if !ok {
				return nil
			}
			withSecrets, err := g.enrichAgentWithSecrets(ctx, agent)
			if err != nil {
				return err
			}
			mu.Lock()
			agents = append(agents, withSecrets)
			mu.Unlock()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].GetName() < agents[j].GetName()
	})
	return agents, nil
}

func (g *GetCmd) enrichAgentWithSecrets(
	ctx context.Context,
	agent v1alpha.GenericObject,
) (v1alpha.GenericObject, error) {
	keys, err := g.client.AuthData().V1().GetAgentCredentials(ctx, agent.GetProject(), agent.GetName())
	if err != nil {
		return nil, err
	}
	meta, ok := agent["metadata"].(map[string]any)
	if !ok {
		return agent, nil
	}
	meta["client_id"] = keys.ClientID
	meta["client_secret"] = keys.ClientSecret
	agent["metadata"] = meta
	return agent, nil
}

func (g *GetCmd) getObjects(ctx context.Context, kind manifest.Kind, args []string) ([]manifest.Object, error) {
	query := buildObjectSelectionQuery(kind, args, g.selection)
	header := http.Header{sdk.HeaderProject: []string{g.client.Config.Project}}
	objects, err := g.client.Objects().V1().Get(ctx, kind, header, query)
	if err != nil {
		return nil, err
	}
	return objects, nil
}

func (g *GetCmd) printObjects(kind manifest.Kind, objects []manifest.Object) error {
	if len(objects) == 0 {
		switch {
		case objectKindSupportsProjectFlag(kind):
			fmt.Printf("No resources found in '%s' project.\n", g.client.Config.Project)
		default:
			fmt.Printf("No resources found.\n")
		}
		return nil
	}
	return g.printer.Print(objects)
}

func parseFilterLabel(filterLabels []string) string {
	labels := make(v1alpha.Labels)
	for _, filterLabel := range filterLabels {
		filteredLabels := strings.SplitSeq(filterLabel, ",")
		for currentLabel := range filteredLabels {
			values := strings.Split(currentLabel, "=")
			key := values[0]
			if _, ok := labels[key]; !ok {
				labels[key] = nil
			}
			if len(values) == 2 {
				labels[key] = append(labels[key], values[1])
			}
		}
	}
	var strLabels []string
	for key, values := range labels {
		if len(values) > 0 {
			for _, value := range values {
				strLabels = append(strLabels, fmt.Sprintf("%s:%s", key, value))
			}
		} else {
			strLabels = append(strLabels, key)
		}
	}
	return strings.Join(strLabels, ",")
}

func stringsTypeToStrings[T ~string](generic []T) []string {
	s := make([]string, 0, len(generic))
	for _, v := range generic {
		s = append(s, string(v))
	}
	return s
}
