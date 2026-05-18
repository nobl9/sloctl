package internal

import (
	"net/url"

	"github.com/nobl9/nobl9-go/manifest"
	objectsV1 "github.com/nobl9/nobl9-go/sdk/endpoints/objects/v1"
	"github.com/spf13/cobra"
)

type objectSelectionFlags struct {
	labels      []string
	project     string
	services    []string
	allProjects bool
	slo         string
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
