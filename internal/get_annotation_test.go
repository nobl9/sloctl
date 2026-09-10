//go:build unit_test

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/manifest/v1alpha"
	v1alphaAnnotation "github.com/nobl9/nobl9-go/manifest/v1alpha/annotation"
)

func TestAnnotationResponseToGenericObject(t *testing.T) {
	replayFacts := map[string]any{
		"periodStart":        "2026-09-01T00:00:00Z",
		"periodEnd":          "2026-09-02T00:00:00Z",
		"elapsedTimeSeconds": float64(93),
	}

	tests := map[string]struct {
		response v1alpha.GenericObject
		wantSpec map[string]any
	}{
		"replay facts are carried into spec": {
			response: v1alpha.GenericObject{
				"name":        "replay-annotation",
				"project":     "default",
				"slo":         "my-slo",
				"description": "Replay finished",
				"startTime":   "2026-09-01T00:00:00Z",
				"endTime":     "2026-09-02T00:00:00Z",
				"category":    "Replay",
				"author":      "someone@example.com",
				"replay":      replayFacts,
			},
			wantSpec: map[string]any{
				"slo":         "my-slo",
				"description": "Replay finished",
				"startTime":   "2026-09-01T00:00:00Z",
				"endTime":     "2026-09-02T00:00:00Z",
				"category":    "Replay",
				"createdBy":   "someone@example.com",
				"replay":      replayFacts,
			},
		},
		"non-replay annotation has no replay key": {
			response: v1alpha.GenericObject{
				"name":        "comment-annotation",
				"slo":         "my-slo",
				"description": "a note",
				"startTime":   "2026-09-01T00:00:00Z",
				"category":    "Comment",
			},
			wantSpec: map[string]any{
				"slo":         "my-slo",
				"description": "a note",
				"startTime":   "2026-09-01T00:00:00Z",
				"category":    "Comment",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			object := annotationResponseToGenericObject(test.response)

			assert.Equal(t, manifest.VersionV1alpha.String(), object["apiVersion"])
			assert.Equal(t, manifest.KindAnnotation.String(), object["kind"])
			assert.Equal(t, test.wantSpec, object["spec"])
		})
	}
}

// A response already shaped as a manifest must not be rewritten, so spec.replay survives untouched.
func TestAnnotationResponseToGenericObject_AlreadyAManifest(t *testing.T) {
	response := v1alpha.GenericObject{
		"apiVersion": manifest.VersionV1alpha.String(),
		"kind":       manifest.KindAnnotation.String(),
		"metadata":   map[string]any{"name": "replay-annotation"},
		"spec":       map[string]any{"replay": map[string]any{"periodStart": "2026-09-01T00:00:00Z"}},
	}

	assert.Equal(t, response, annotationResponseToGenericObject(response))
}

func TestBuildGetAnnotationsRequest_Categories(t *testing.T) {
	tests := map[string]struct {
		selection objectSelectionFlags
		want      []v1alphaAnnotation.Category
	}{
		"explicit Replay category": {
			selection: objectSelectionFlags{annotationCategories: []string{"Replay"}},
			want:      []v1alphaAnnotation.Category{v1alphaAnnotation.CategoryReplay},
		},
		"default set is the user categories, which include Replay": {
			selection: objectSelectionFlags{},
			want:      v1alphaAnnotation.GetUserCategories(),
		},
		"--user selects the user categories": {
			selection: objectSelectionFlags{annotationUserCategories: true},
			want:      v1alphaAnnotation.GetUserCategories(),
		},
		"--system selects the system categories": {
			selection: objectSelectionFlags{annotationSystemCategories: true},
			want:      v1alphaAnnotation.GetSystemCategories(),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			params, err := buildGetAnnotationsRequest(nil, test.selection)
			require.NoError(t, err)
			assert.ElementsMatch(t, test.want, params.Categories)
		})
	}

	t.Run("Replay is a valid --category value", func(t *testing.T) {
		assert.Contains(t, v1alphaAnnotation.CategoryValues(), v1alphaAnnotation.CategoryReplay)
	})

	t.Run("an unknown category is rejected", func(t *testing.T) {
		_, err := buildGetAnnotationsRequest(nil, objectSelectionFlags{annotationCategories: []string{"Nope"}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid 'category' flag value")
	})
}
