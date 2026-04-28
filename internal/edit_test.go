package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/manifest/v1alpha"
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEditCmd_Subcommands(t *testing.T) {
	root := &RootCmd{}
	cmd := root.NewEditCmd()

	sloCmd, _, err := cmd.Find([]string{"slo"})
	require.NoError(t, err)
	assert.Equal(t, "slos", sloCmd.Name())

	serviceCmd, _, err := cmd.Find([]string{"svc"})
	require.NoError(t, err)
	assert.Equal(t, "services", serviceCmd.Name())

	sloCmdAlias, _, err := cmd.Find([]string{"SLOs"})
	require.NoError(t, err)
	assert.Equal(t, "slos", sloCmdAlias.Name())

	_, _, err = cmd.Find([]string{"slo/my-slo"})
	require.Error(t, err)
	assert.Equal(t, `unknown command "slo/my-slo" for "edit"`, err.Error())
}

func TestValidateEditedObjectsMatchSelection(t *testing.T) {
	original := []manifest.Object{
		v1alpha.GenericObject{
			"apiVersion": manifest.VersionV1alpha,
			"kind":       manifest.KindSLO,
			"metadata": map[string]any{
				"project": "project-1",
				"name":    "my-slo",
			},
		},
	}

	t.Run("same identity", func(t *testing.T) {
		edited := []manifest.Object{
			v1alpha.GenericObject{
				"apiVersion": manifest.VersionV1alpha,
				"kind":       manifest.KindSLO,
				"metadata": map[string]any{
					"project": "project-1",
					"name":    "my-slo",
					"labels": map[string]any{
						"team": []string{"green"},
					},
				},
			},
		}

		require.NoError(t, validateEditedObjectsMatchSelection(original, edited))
	})

	t.Run("changed identity", func(t *testing.T) {
		edited := []manifest.Object{
			v1alpha.GenericObject{
				"apiVersion": manifest.VersionV1alpha,
				"kind":       manifest.KindSLO,
				"metadata": map[string]any{
					"project": "project-1",
					"name":    "my-slo-renamed",
				},
			},
		}

		err := validateEditedObjectsMatchSelection(original, edited)
		require.Error(t, err)
		assert.Equal(t,
			"edited resources must match the selected resources; changing kind, name, or project is not supported",
			err.Error(),
		)
	})
}

func TestAddEditErrorToContents(t *testing.T) {
	contents := []byte("apiVersion: n9/v1alpha\nkind: SLO\n")
	err := fmt.Errorf("unable to decode \"edited-file\": json: cannot unmarshal bool into Go struct field")

	updated := addEditErrorToContents(contents, err)

	expected := "# The edited file had a syntax error: " +
		"unable to decode \"edited-file\": json: cannot unmarshal bool into Go struct field\n" +
		"#\n\n" +
		"apiVersion: n9/v1alpha\nkind: SLO\n"

	assert.Equal(t,
		expected,
		string(updated),
	)
}

func TestAddEditErrorToContents_ReplacesPreviousEditError(t *testing.T) {
	contents := []byte(
		"# The edited file had a syntax error: previous error\n#\n\napiVersion: n9/v1alpha\nkind: SLO\n",
	)
	err := fmt.Errorf("new error")

	updated := addEditErrorToContents(contents, err)

	assert.Equal(t,
		"# The edited file had a syntax error: new error\n#\n\napiVersion: n9/v1alpha\nkind: SLO\n",
		string(updated),
	)
}

func TestDefaultEditorForOS(t *testing.T) {
	tests := map[string]struct {
		goos           string
		expectedEditor string
	}{
		"darwin": {
			goos:           "darwin",
			expectedEditor: "open -W -n -t",
		},
		"windows": {
			goos:           "windows",
			expectedEditor: "notepad",
		},
		"linux": {
			goos:           "linux",
			expectedEditor: "vi",
		},
		"unknown": {
			goos:           "unknown",
			expectedEditor: "vi",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expectedEditor, defaultEditorForOS(test.goos))
		})
	}
}

func TestResolveEditor(t *testing.T) {
	t.Setenv("SLOCTL_EDITOR", "")
	t.Setenv("KUBE_EDITOR", "")
	t.Setenv("EDITOR", "")

	assert.Equal(t, defaultEditorForOS(runtime.GOOS), resolveEditor(runtime.GOOS))

	t.Setenv("EDITOR", "vim")
	assert.Equal(t, "vim", resolveEditor(runtime.GOOS))

	t.Setenv("KUBE_EDITOR", "nano")
	assert.Equal(t, "nano", resolveEditor(runtime.GOOS))

	t.Setenv("SLOCTL_EDITOR", "code --wait")
	assert.Equal(t, "code --wait", resolveEditor(runtime.GOOS))
}

func TestEditCmd_Run_NoResourcesFoundInProject(t *testing.T) {
	client := newEditTestClient(t, "default", []manifest.Object{})
	edit := EditCmd{client: client}
	cmd := new(cobra.Command)
	cmd.SetContext(context.Background())

	output := captureStdout(t, func() {
		err := edit.run(cmd, manifest.KindSLO, []string{"foo"})
		require.NoError(t, err)
	})

	assert.Equal(t, "No resources found in 'default' project.\n", output)
}

func TestEditCmd_Run_ReturnsMissingNamesWhenOnlySomeObjectsExist(t *testing.T) {
	client := newEditTestClient(t, "default", []manifest.Object{
		v1alpha.GenericObject{
			"apiVersion": manifest.VersionV1alpha,
			"kind":       manifest.KindSLO,
			"metadata": map[string]any{
				"project": "default",
				"name":    "foo",
			},
		},
	})
	edit := EditCmd{client: client}
	cmd := new(cobra.Command)
	cmd.SetContext(context.Background())

	output := captureStdout(t, func() {
		err := edit.run(cmd, manifest.KindSLO, []string{"foo", "bar", "bar", "baz"})
		require.EqualError(t, err, "resource(s) not found: bar, baz")
	})

	assert.Empty(t, output)
}

func newEditTestClient(t *testing.T, project string, objects []manifest.Object) *sdk.Client {
	t.Helper()

	body, err := json.Marshal(objects)
	require.NoError(t, err)

	client, err := sdk.NewClient(&sdk.Config{
		DisableOkta: true,
		Project:     project,
	})
	require.NoError(t, err)

	client.HTTP = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, project, r.Header.Get(sdk.HeaderProject))
			rec := httptest.NewRecorder()
			_, writeErr := rec.Write(body)
			require.NoError(t, writeErr)
			return rec.Result(), nil
		}),
	}

	return client
}

func captureStdout(t *testing.T, run func()) string {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = writer
	t.Cleanup(func() {
		os.Stdout = originalStdout
	})

	run()

	require.NoError(t, writer.Close())
	output, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())

	return string(output)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
