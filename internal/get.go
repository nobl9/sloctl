package internal

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/manifest/v1alpha"
	v1alphaAnnotation "github.com/nobl9/nobl9-go/manifest/v1alpha/annotation"
	"github.com/nobl9/nobl9-go/sdk"
	objectsV1 "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v1"
	usersV2 "github.com/nobl9/nobl9-go/sdk/endpoints/users/v2"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/nobl9/sloctl/internal/flags"
	"github.com/nobl9/sloctl/internal/printer"
)

//go:embed get_alert_example.sh
var getAlertExample string

//go:embed get_annotation_example.sh
var getAnnotationExample string

//go:embed get_example.sh
var getExample string

type GetCmd struct {
	client    *sdk.Client
	printer   *printer.Printer
	selection objectSelectionFlags
	sloLimit  int
	sloOffset int
}

// NewGetCmd returns cobra command get with all flags for it.
func (r *RootCmd) NewGetCmd() *cobra.Command {
	get := &GetCmd{
		printer: printer.NewPrinter(printer.Config{}),
	}

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get Nobl9 resources",
		Long: "Get resources by name or filter and print them as YAML, JSON, or CSV. YAML is the default.\n\n" +
			"Resource names can be supplied as arguments or read from standard input. Without names, each resource " +
			"command returns all resources matching its filters. Use `--jq` to filter or transform the results.",
		Example: getExample,
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
		{Kind: manifest.KindSLO, Extender: get.newGetSLOCommand},
		{Kind: manifest.KindUserGroup},
		{Kind: manifest.KindBudgetAdjustment},
		{Kind: manifest.KindReport},
	} {
		plural := pluralForKind(subCmd.Kind)
		use := strings.ToLower(plural)
		subCmd.Aliases = append(subCmd.Aliases, subCmd.Kind.ToLower(), subCmd.Kind.String(), plural)

		sc := get.newGetObjectsCommand(subCmd.Kind, use, subCmd.Aliases)
		if subCmd.Extender != nil {
			subCmd.Extender(sc)
		}
		registerObjectSelectionFlags(sc, subCmd.Kind, &get.selection,
			"Select resources across all projects.")
		cmd.AddCommand(sc)
	}
	cmd.AddCommand(get.newGetUserCommand())

	return cmd
}

func (g *GetCmd) newGetObjectsCommand(
	kind manifest.Kind,
	use string,
	aliases []string,
) *cobra.Command {
	resourceName := humanReadablePluralForKind(kind)
	return &cobra.Command{
		Use:     use + " [name...]",
		Aliases: aliases,
		Short:   fmt.Sprintf("Get %s", resourceName),
		Long:    getObjectsLongDescription(kind, resourceName),
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := readStdinArgs(cmd, args)
			if err != nil {
				return err
			}
			objects, err := g.getObjects(cmd.Context(), kind, names)
			if err != nil {
				return err
			}
			return g.printObjects(kind, objects)
		},
	}
}

func getObjectsLongDescription(kind manifest.Kind, resourceName string) string {
	description := fmt.Sprintf(
		"Get %s by name or available filters. Names can be supplied as arguments or read from standard input. "+
			"Without names, all matching %s are returned.",
		resourceName,
		resourceName,
	)
	switch {
	case objectKindSupportsProjectFlag(kind):
		description += " Use `--project` to select a project or `--all-projects` to search all projects."
	case kind == manifest.KindBudgetAdjustment:
		description += " `--project` and `--slo` must be supplied together when filtering by SLO."
	}
	return description
}

func (g *GetCmd) newGetSLOCommand(cmd *cobra.Command) *cobra.Command {
	cmd.Long += "\nBy default, all matching SLOs are returned. Use --limit and --offset to request a page."
	cmd.Flags().IntVar(&g.sloLimit, "limit", 0, "Maximum number of SLOs to return (1-1000).")
	cmd.Flags().IntVar(&g.sloOffset, "offset", 0, "Number of SLOs to skip (requires --limit).")
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("limit") && (g.sloLimit < 1 || g.sloLimit > 1000) {
			return fmt.Errorf("--limit must be between 1 and 1000")
		}
		if g.sloOffset < 0 {
			return fmt.Errorf("--offset must be nonnegative")
		}
		if cmd.Flags().Changed("offset") && !cmd.Flags().Changed("limit") {
			return fmt.Errorf("--offset requires --limit")
		}
		return nil
	}
	return cmd
}

func (g *GetCmd) newGetUserCommand() *cobra.Command {
	limit := uint(100)
	cmd := &cobra.Command{
		Use:   "user [id...]",
		Short: "Get users by ID",
		Long: "Get users by ID. IDs can be supplied as arguments or read from standard input. " +
			fmt.Sprintf("Without IDs, up to `--limit` users are returned; the default limit is %d.", limit),
		RunE: func(cmd *cobra.Command, args []string) error {
			ids, err := readStdinArgs(cmd, args)
			if err != nil {
				return err
			}
			request := usersV2.GetUsersRequest{IDs: ids}
			if len(ids) == 0 || cmd.Flags().Changed("limit") {
				request.Limit = limit
			}
			users, err := g.client.Users().V2().GetUsers(
				cmd.Context(),
				request,
			)
			if err != nil {
				return err
			}
			return g.printUsers(users)
		},
	}
	cmd.Flags().UintVar(&limit, "limit", limit,
		"Maximum number of users to return. The default limit applies when no IDs are provided.")
	return cmd
}

// nolint: gocognit
func (g *GetCmd) newGetAlertCommand(cmd *cobra.Command) *cobra.Command {
	cmd.Use = "alerts [id...]"
	cmd.Example = getAlertExample
	cmd.Long = "Get alerts by ID or filter. Alert IDs can also be read from standard input. " +
		"Repeat the same filter to match any supplied value; different filters are combined.\n\n" +
		"Active and resolved alerts are both included by default. Set `--resolved=false` or `--triggered=false` " +
		"to select only one status.\n\n" +
		"A filter can return no alerts when a referenced Alert Policy, SLO, Service, or Objective was deleted. " +
		"Recreating a resource with the same name does not restore its old alerts. " +
		"Unlinking an Alert Policy from an SLO can also hide related alerts.\n\n" +
		"Alert output includes these timing and resolution fields:\n\n" +
		"- `spec.conditions[].status.firstMetMetricTime`: when the condition first became true.\n" +
		"- `spec.conditions[].status.lastsForMetMetricTime`: when the required `lastsFor` duration was met.\n" +
		"- `spec.conditions[].status.lastMetMetricTime`: the last time the condition remained true.\n" +
		"- `spec.coolDownStartedAtMetricTime`: when the cooldown started for a resolved alert.\n" +
		"- `spec.resolutionReason`: why the alert was resolved or canceled.\n\n" +
		"Results may be truncated by the API. If sloctl reports truncation, use narrower filters."

	params := objectsV1.GetAlertsRequest{
		Resolved:  new(bool),
		Triggered: new(bool),
	}
	cmd.Flags().StringArrayVar(
		&params.AlertPolicyNames,
		"alert-policy",
		[]string{},
		"Filter by alert policy name. Repeat to match any of several policies.",
	)
	cmd.Flags().StringArrayVar(
		&params.SLONames,
		"slo",
		[]string{},
		"Filter by SLO name. Repeat to match any of several SLOs.",
	)
	cmd.Flags().StringArrayVar(
		&params.ObjectiveNames,
		"objective",
		[]string{},
		"Filter by objective name. Repeat to match any of several objectives.",
	)
	cmd.Flags().StringArrayVar(
		&params.ServiceNames,
		"service",
		[]string{},
		"Filter by service name. Repeat to match any of several services.",
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
		"Include resolved alerts. Set --resolved=false to exclude them.",
	)
	cmd.Flags().BoolVar(
		params.Triggered,
		"triggered",
		true,
		"Include active alerts. Set --triggered=false to exclude them.",
	)
	flags.RegisterTimeVar(
		cmd,
		&params.From,
		"from",
		"Set the start of the alert metric-time range in RFC3339 format.",
	)
	flags.RegisterTimeVar(
		cmd,
		&params.To,
		"to",
		"Set the end of the alert metric-time range in RFC3339 format.",
	)

	cmd.Flags().SortFlags = false
	cmd.Flags().Lookup("objective-value").Hidden = true

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		names, err := readStdinArgs(cmd, args)
		if err != nil {
			return err
		}
		if len(names) > 0 {
			params.Names = names
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
		"Include agent client_id and client_secret values. "+
			"This performs one additional credential request per returned agent.")
	cmd.Long += "\n\n`--with-keys` includes `client_id` and `client_secret` in the output and performs one " +
		"additional credential request for every returned agent."

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		names, err := readStdinArgs(cmd, args)
		if err != nil {
			return err
		}
		objects, err := g.getObjects(cmd.Context(), manifest.KindAgent, names)
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
	categories := stringsTypeToStrings(v1alphaAnnotation.GetUserCategories())
	for i := range categories {
		categories[i] = "`" + categories[i] + "`"
	}
	cmd.Long = fmt.Sprintf("Get annotations by name or filter. Pass names as arguments or through standard input. "+
		"Without a category selector, this command returns only user categories (%s). "+
		"Use `--system`, `--user`, or repeated `--category` flags to select other categories.\n\n"+
		"Annotation category origins:\n\n"+
		"- `Comment`: added manually to an SLO.\n"+
		"- `ReviewNote`: created when an SLO review status changes.\n"+
		"- `SloEdit`: created when an SLO definition changes.\n\n"+
		"See [Annotation types](https://docs.nobl9.com/features/slo-annotations/#annotation-types).",
		strings.Join(categories, ", "))

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
	if kind == manifest.KindAnnotation {
		return g.getAnnotations(ctx, args)
	}
	query := buildObjectSelectionQuery(kind, args, g.selection)
	if kind == manifest.KindSLO && g.sloLimit > 0 {
		query.Set(objectsV1.QueryKeyPaginationLimit, strconv.Itoa(g.sloLimit))
		if g.sloOffset > 0 {
			query.Set(objectsV1.QueryKeyPaginationOffset, strconv.Itoa(g.sloOffset))
		}
	}
	header := http.Header{sdk.HeaderProject: []string{g.client.Config.Project}}
	objects, err := g.client.Objects().V1().Get(ctx, kind, header, query)
	if err != nil {
		return nil, err
	}
	return objects, nil
}

func (g *GetCmd) getAnnotations(ctx context.Context, names []string) ([]manifest.Object, error) {
	params, err := buildGetAnnotationsRequest(names, g.selection)
	if err != nil {
		return nil, err
	}
	header := http.Header{}
	if params.Project != "" {
		header.Set(sdk.HeaderProject, params.Project)
	}
	req, err := g.client.CreateRequest(
		ctx,
		http.MethodGet,
		"/annotations",
		header,
		buildGetAnnotationsQuery(params),
		nil,
	)
	if err != nil {
		return nil, err
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var annotations []v1alpha.GenericObject
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	if err = decoder.Decode(&annotations); err != nil {
		return nil, err
	}
	objects := make([]manifest.Object, 0, len(annotations))
	for _, annotation := range annotations {
		objects = append(objects, annotationResponseToGenericObject(annotation))
	}
	return objects, nil
}

func annotationResponseToGenericObject(annotation v1alpha.GenericObject) v1alpha.GenericObject {
	if _, ok := annotation["apiVersion"]; ok {
		if _, ok = annotation["kind"]; ok {
			return annotation
		}
	}

	metadata := map[string]any{
		"name": annotation["name"],
	}
	if project, ok := annotation["project"]; ok {
		metadata["project"] = project
	}
	if labels, ok := annotation["labels"]; ok {
		metadata["labels"] = labels
	}

	spec := map[string]any{
		"slo":         annotation["slo"],
		"description": annotation["description"],
		"startTime":   annotation["startTime"],
	}
	if objectiveName, ok := annotation["objectiveName"]; ok {
		spec["objectiveName"] = objectiveName
	}
	if endTime, ok := annotation["endTime"]; ok && endTime != nil {
		spec["endTime"] = endTime
	}
	if category, ok := annotation["category"]; ok {
		spec["category"] = category
	}
	if replay, ok := annotation["replay"]; ok {
		spec["replay"] = replay
	}
	if author, ok := annotation["author"]; ok {
		spec["createdBy"] = author
	}

	object := v1alpha.GenericObject{
		"apiVersion": manifest.VersionV1alpha.String(),
		"kind":       manifest.KindAnnotation.String(),
		"metadata":   metadata,
		"spec":       spec,
	}
	if status, ok := annotation["status"]; ok {
		object["status"] = status
	}
	return object
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

func (g *GetCmd) printUsers(users []usersV2.User) error {
	if len(users) == 0 {
		fmt.Printf("No resources found.\n")
		return nil
	}
	return g.printer.Print(users)
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
