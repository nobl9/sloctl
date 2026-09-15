package internal

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/nobl9/nobl9-go/manifest"
	v1alphaAnnotation "github.com/nobl9/nobl9-go/manifest/v1alpha/annotation"
	objectsV1 "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v1"
	objectsV2 "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v2"
	"github.com/spf13/cobra"

	"github.com/nobl9/sloctl/internal/collections"
	"github.com/nobl9/sloctl/internal/flags"
)

type objectSelectionFlags struct {
	labels                     []string
	project                    string
	services                   []string
	allProjects                bool
	slo                        string
	annotationFrom             time.Time
	annotationTo               time.Time
	annotationCategories       []string
	annotationUserCategories   bool
	annotationSystemCategories bool
}

func registerObjectSelectionFlags(
	cmd *cobra.Command,
	kind manifest.Kind,
	selection *objectSelectionFlags,
	allProjectsUsage string,
) {
	if objectKindSupportsProjectFlag(kind) {
		registerProjectFlag(cmd, &selection.project)
		cmd.Flags().BoolVarP(&selection.allProjects, "all-projects", "A", false, allProjectsUsage)
	}
	if objectKindSupportsLabelsFlag(kind) {
		registerLabelsFlag(cmd, &selection.labels)
	}
	if kind == manifest.KindSLO {
		registerSLOServiceFlag(cmd, &selection.services)
	}
	if kind == manifest.KindBudgetAdjustment {
		cmd.Flags().StringVarP(&selection.project, "project", "p", "",
			"Filter budget adjustments by SLO project. Must be used with --slo.")
		cmd.Flags().StringVarP(&selection.slo, "slo", "", "",
			"Filter budget adjustments by SLO name. Must be used with --project.")
		cmd.MarkFlagsRequiredTogether("slo", "project")
	}
	if kind == manifest.KindAnnotation {
		registerAnnotationSelectionFlags(cmd, selection)
	}
}

func buildObjectSelectionQuery(kind manifest.Kind, names []string, selection objectSelectionFlags) url.Values {
	query := url.Values{objectsV1.QueryKeyName: names}
	if len(selection.labels) > 0 {
		query.Set(objectsV1.QueryKeyLabels, parseFilterLabel(selection.labels))
	}
	if len(selection.services) > 0 && kind == manifest.KindSLO {
		query[objectsV1.QueryKeyServiceName] = selection.services
	}
	if selection.slo != "" && selection.project != "" && kind == manifest.KindBudgetAdjustment {
		query.Set(objectsV1.QueryKeySLOProjectName, selection.project)
		query.Set(objectsV1.QueryKeySLOName, selection.slo)
	}
	return query
}

func objectKindSupportsSelectionProjectFlag(kind manifest.Kind) bool {
	return objectKindSupportsProjectFlag(kind) || kind == manifest.KindBudgetAdjustment
}

func humanReadablePluralForKind(kind manifest.Kind) string {
	switch kind {
	case manifest.KindAlertMethod:
		return "alert methods"
	case manifest.KindAlertPolicy:
		return "alert policies"
	case manifest.KindAlertSilence:
		return "alert silences"
	case manifest.KindBudgetAdjustment:
		return "budget adjustments"
	case manifest.KindDataExport:
		return "data exports"
	case manifest.KindDirect:
		return "direct data sources"
	case manifest.KindRoleBinding:
		return "role bindings"
	case manifest.KindSLO:
		return "SLOs"
	case manifest.KindUserGroup:
		return "user groups"
	default:
		return strings.ToLower(pluralForKind(kind))
	}
}

func registerAnnotationSelectionFlags(cmd *cobra.Command, selection *objectSelectionFlags) {
	cmd.Flags().StringVar(
		&selection.slo,
		"slo",
		"",
		"Filter annotations by SLO name.",
	)
	flags.RegisterTimeVar(
		cmd,
		&selection.annotationFrom,
		"from",
		"Filter annotations whose spec.startTime is at or after this RFC3339 timestamp.",
	)
	flags.RegisterTimeVar(
		cmd,
		&selection.annotationTo,
		"to",
		"Filter annotations whose spec.endTime is at or before this RFC3339 timestamp.",
	)
	cmd.Flags().BoolVar(
		&selection.annotationUserCategories,
		"user",
		false,
		"Include annotations in user categories.",
	)
	cmd.Flags().BoolVar(
		&selection.annotationSystemCategories,
		"system",
		false,
		"Include annotations in system categories.",
	)
	cmd.Flags().StringArrayVar(
		&selection.annotationCategories,
		"category",
		nil,
		fmt.Sprintf(
			"Filter by annotation category (one of: %s). Repeat to select multiple categories.",
			strings.Join(stringsTypeToStrings(v1alphaAnnotation.CategoryValues()), ", "),
		),
	)
}

func buildGetAnnotationsRequest(
	names []string,
	selection objectSelectionFlags,
) (objectsV2.GetAnnotationsRequest, error) {
	params := objectsV2.GetAnnotationsRequest{
		Project: selection.project,
		Names:   names,
		SLOName: selection.slo,
		From:    selection.annotationFrom,
		To:      selection.annotationTo,
	}
	for _, cat := range selection.annotationCategories {
		parsed, err := v1alphaAnnotation.ParseCategory(cat)
		if err != nil {
			return params, fmt.Errorf("invalid 'category' flag value: %w", err)
		}
		params.Categories = append(params.Categories, parsed)
	}
	if selection.annotationSystemCategories {
		params.Categories = append(params.Categories, v1alphaAnnotation.GetSystemCategories()...)
	}
	if selection.annotationUserCategories {
		params.Categories = append(params.Categories, v1alphaAnnotation.GetUserCategories()...)
	}
	if len(params.Categories) == 0 {
		params.Categories = v1alphaAnnotation.GetUserCategories()
	}
	params.Categories = collections.RemoveDuplicates(params.Categories)
	return params, nil
}

func buildGetAnnotationsQuery(params objectsV2.GetAnnotationsRequest) url.Values {
	query := url.Values{objectsV2.QueryKeyName: params.Names}
	if !params.From.IsZero() {
		query.Add(objectsV2.QueryKeyFrom, params.From.Format(time.RFC3339))
	}
	if !params.To.IsZero() {
		query.Add(objectsV2.QueryKeyTo, params.To.Format(time.RFC3339))
	}
	if params.SLOName != "" {
		query.Set(objectsV2.QueryKeySLOName, params.SLOName)
	}
	for _, category := range params.Categories {
		query.Add(objectsV2.QueryKeyCategory, category.String())
	}
	return query
}
