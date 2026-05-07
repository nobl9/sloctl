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
	"strings"
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

func TestEditedFileIsEmpty(t *testing.T) {
	tests := map[string]struct {
		contents []byte
		expected bool
	}{
		"empty": {
			expected: true,
		},
		"whitespace": {
			contents: []byte(" \n\t\n"),
			expected: true,
		},
		"comments": {
			contents: []byte("# comment\n  # indented comment\n\n"),
			expected: true,
		},
		"object": {
			contents: []byte("apiVersion: n9/v1alpha\nkind: SLO\n"),
			expected: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected, editedFileIsEmpty(test.contents))
		})
	}
}

func TestWriteObjectsToTemporaryFile_AddsNotice(t *testing.T) {
	path, contents, err := writeObjectsToTemporaryFile([]manifest.Object{
		v1alpha.GenericObject{
			"apiVersion": manifest.VersionV1alpha,
			"kind":       manifest.KindSLO,
			"metadata": map[string]any{
				"project": "default",
				"name":    "my-slo",
			},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.Remove(path))
	})

	fileContents, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.Equal(t, contents, fileContents)
	assert.True(t, strings.HasPrefix(string(contents), editFileNotice))
	assert.Contains(t, string(contents), "kind: SLO")
}

func TestDefaultEditorForOS(t *testing.T) {
	tests := map[string]struct {
		goos           string
		lookup         editorLookup
		expectedEditor string
	}{
		"darwin with vim": {
			goos:           "darwin",
			lookup:         fakeEditorLookup(defaultEditorUnixVim),
			expectedEditor: defaultEditorUnixVim,
		},
		"darwin falls back to nano when vim and vi are missing": {
			goos:           "darwin",
			lookup:         fakeEditorLookup(),
			expectedEditor: defaultEditorUnixFallback,
		},
		"windows": {
			goos:           "windows",
			lookup:         fakeEditorLookup(),
			expectedEditor: defaultEditorWindows,
		},
		"linux with vim": {
			goos:           "linux",
			lookup:         fakeEditorLookup(defaultEditorUnixVim, defaultEditorUnixVi),
			expectedEditor: defaultEditorUnixVim,
		},
		"linux with vi when vim is missing": {
			goos:           "linux",
			lookup:         fakeEditorLookup(defaultEditorUnixVi),
			expectedEditor: defaultEditorUnixVi,
		},
		"linux falls back to nano when vim and vi are missing": {
			goos:           "linux",
			lookup:         fakeEditorLookup(defaultEditorUnixFallback),
			expectedEditor: defaultEditorUnixFallback,
		},
		"linux uses nano even when no editor is found": {
			goos:           "linux",
			lookup:         fakeEditorLookup(),
			expectedEditor: defaultEditorUnixFallback,
		},
		"unknown with vim": {
			goos:           "unknown",
			lookup:         fakeEditorLookup(defaultEditorUnixVim),
			expectedEditor: defaultEditorUnixVim,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expectedEditor, defaultEditorForOS(test.goos, test.lookup))
		})
	}
}

func TestResolveEditor(t *testing.T) {
	t.Setenv(editorEnvSloctl, "")
	t.Setenv(editorEnvSystem, "")

	lookup := fakeEditorLookup(defaultEditorUnixVim)

	assert.Equal(t, defaultEditorForOS(runtime.GOOS, lookup), resolveEditor(runtime.GOOS, lookup))
	assert.Equal(t, defaultEditorUnixVim, resolveEditor("darwin", lookup))

	t.Setenv(editorEnvSystem, defaultEditorUnixVim)
	assert.Equal(t, defaultEditorUnixVim, resolveEditor(runtime.GOOS, lookup))

	t.Setenv(editorEnvSloctl, "code --wait")
	assert.Equal(t, "code --wait", resolveEditor(runtime.GOOS, lookup))
}

func TestResolveShell(t *testing.T) {
	t.Setenv(shellEnv, "")

	assert.Equal(t, defaultShellUnix, resolveShell("linux"))
	assert.Equal(t, defaultShellUnix, resolveShell("darwin"))
	assert.Equal(t, defaultShellWindows, resolveShell("windows"))

	t.Setenv(shellEnv, "/bin/zsh")
	assert.Equal(t, "/bin/zsh", resolveShell("linux"))
}

func TestShellCommandArg(t *testing.T) {
	tests := map[string]struct {
		goos     string
		shell    string
		expected string
	}{
		"unix shell": {
			goos:     "linux",
			shell:    defaultShellUnix,
			expected: shellArgUnix,
		},
		"windows cmd": {
			goos:     "windows",
			shell:    defaultShellWindows,
			expected: shellArgWindows,
		},
		"windows cmd exe": {
			goos:     "windows",
			shell:    defaultShellWindows + ".exe",
			expected: shellArgWindows,
		},
		"windows non-cmd shell": {
			goos:     "windows",
			shell:    "powershell",
			expected: shellArgUnix,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected, shellCommandArg(test.goos, test.shell))
		})
	}
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

func fakeEditorLookup(availableEditors ...string) editorLookup {
	available := make(map[string]struct{}, len(availableEditors))
	for _, editor := range availableEditors {
		available[editor] = struct{}{}
	}
	return func(editor string) (string, error) {
		if _, ok := available[editor]; ok {
			return "/usr/bin/" + editor, nil
		}
		return "", os.ErrNotExist
	}
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
