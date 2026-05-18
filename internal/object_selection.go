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
		registerProjectFlag(cmd, &selection.project)
		cmd.Flags().StringVarP(&selection.slo, "slo", "", "",
			`Filter resource by SLO name. Example: my-sample-slo-name.`)
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
	if len(selection.slo) > 0 && len(selection.project) > 0 && kind == manifest.KindBudgetAdjustment {
		query.Set(objectsV1.QueryKeySLOProjectName, selection.project)
		query.Set(objectsV1.QueryKeySLOName, selection.slo)
	}
	return query
}

func objectKindSupportsSelectionProjectFlag(kind manifest.Kind) bool {
	return objectKindSupportsProjectFlag(kind) || kind == manifest.KindBudgetAdjustment
}

func registerAnnotationSelectionFlags(cmd *cobra.Command, selection *objectSelectionFlags) {
	cmd.Flags().StringVar(
		&selection.slo,
		"slo",
		"",
		"Get annotations for a given SLO (name) only.",
	)
	flags.RegisterTimeVar(
		cmd,
		&selection.annotationFrom,
		"from",
		"Get annotations which have 'spec.startTime' after or equal to the given time.",
	)
	flags.RegisterTimeVar(
		cmd,
		&selection.annotationTo,
		"to",
		"Get annotations which have 'spec.endTime' before or equal to the given time.",
	)
	cmd.Flags().BoolVar(
		&selection.annotationUserCategories,
		"user",
		false,
		"Get annotations which were created by user actions.",
	)
	cmd.Flags().BoolVar(
		&selection.annotationSystemCategories,
		"system",
		false,
		"Get annotations which were automatically created by Nobl9 platform.",
	)
	cmd.Flags().StringArrayVar(
		&selection.annotationCategories,
		"category",
		nil,
		fmt.Sprintf(
			"Filter annotations by their category (one of: %s).",
			strings.Join(stringsTypeToStrings(v1alphaAnnotation.CategoryValues()), ", "),
		),
	)
}

func buildGetAnnotationsRequest(
	names []string,
	selection objectSelectionFlags,
) (objectsV2.GetAnnotationsRequest, error) {
	params := objectsV2.GetAnnotationsRequest{
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
