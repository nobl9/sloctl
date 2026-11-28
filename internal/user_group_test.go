package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobl9/nobl9-go/manifest"
	"github.com/nobl9/nobl9-go/manifest/v1alpha"
	v1alphaParser "github.com/nobl9/nobl9-go/manifest/v1alpha/parser"
	"github.com/nobl9/nobl9-go/sdk"
)

// The local transport verifies the Cobra/file/SDK integration without requiring
// the unreleased backend feature on a real Nobl9 organization.
func TestManualUserGroup_MixedManifestRequest(t *testing.T) {
	const definitions = `apiVersion: n9/v1alpha
kind: RoleBinding
metadata:
  name: platform-viewers
spec:
  groupRef: platform-team
  roleRef: project-viewer
  projectRef: default
---
apiVersion: n9/v1alpha
kind: UserGroup
metadata:
  name: platform-team
origin: manual
revision: "2026-11-13T12:00:00Z"
spec:
  displayName: Platform Team
  members: []
`
	path := filepath.Join(t.TempDir(), "groups.yaml")
	require.NoError(t, os.WriteFile(path, []byte(definitions), 0o600))
	previousGeneric := v1alphaParser.UseGenericObjects
	v1alphaParser.UseGenericObjects = true
	t.Cleanup(func() { v1alphaParser.UseGenericObjects = previousGeneric })

	for _, operation := range []struct {
		name   string
		method string
	}{
		{name: "apply", method: http.MethodPut},
		{name: "delete", method: http.MethodDelete},
	} {
		t.Run(operation.name, func(t *testing.T) {
			client, err := sdk.NewClient(&sdk.Config{DisableOkta: true, Organization: "my-org"})
			require.NoError(t, err)
			requests := 0
			client.HTTP = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				requests++
				assert.Equal(t, operation.method, r.Method)
				assert.Equal(t, "/"+operation.name, r.URL.Path)
				assert.Equal(t, "true", r.URL.Query().Get("dry_run"))
				var objects []map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&objects))
				require.Len(t, objects, 2)
				// Keep the batch together and preserve its order. The server resolves
				// groups before bindings on apply and deletes bindings before groups.
				assert.Equal(t, manifest.KindRoleBinding.String(), objects[0]["kind"])
				assert.Equal(t, manifest.KindUserGroup.String(), objects[1]["kind"])
				assert.Equal(t, "manual", objects[1]["origin"])
				assert.Equal(t, "2026-11-13T12:00:00Z", objects[1]["revision"])
				assert.Equal(t, "my-org", objects[1]["organization"])
				spec := objects[1]["spec"].(map[string]any)
				assert.Equal(t, []any{}, spec["members"])
				return httptest.NewRecorder().Result(), nil
			})}
			root := &RootCmd{Client: client}
			root.once.Do(func() {})
			cmd := root.NewApplyCmd()
			if operation.name == "delete" {
				cmd = root.NewDeleteCmd()
			}
			cmd.SetArgs([]string{"-f", path, "--dry-run", "--yes"})
			cmd.SetContext(t.Context())
			require.NoError(t, cmd.Execute())
			assert.Equal(t, 1, requests)
		})
	}
}

func TestManualUserGroup_DeleteByName(t *testing.T) {
	client, err := sdk.NewClient(&sdk.Config{DisableOkta: true})
	require.NoError(t, err)
	requests := 0
	client.HTTP = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/delete/usergroup", r.URL.Path)
		assert.Equal(t, []string{"platform-team", "support-team"}, r.URL.Query()["name"])
		assert.Equal(t, "true", r.URL.Query().Get("dry_run"))
		return httptest.NewRecorder().Result(), nil
	})}
	root := &RootCmd{Client: client}
	root.once.Do(func() {})
	cmd := root.NewDeleteCmd()
	cmd.SetArgs([]string{"usergroups", "platform-team", "support-team", "--dry-run"})
	cmd.SetContext(t.Context())
	require.NoError(t, cmd.Execute())
	assert.Equal(t, 1, requests)
}

func TestManualUserGroup_EditSelection(t *testing.T) {
	manual := v1alpha.GenericObject{
		"kind": "UserGroup", "metadata": map[string]any{"name": "manual-team"}, "origin": "manual",
	}
	for _, origin := range []string{"scim", "system", ""} {
		t.Run(origin, func(t *testing.T) {
			external := v1alpha.GenericObject{
				"kind": "UserGroup", "metadata": map[string]any{"name": "external-team"}, "origin": origin,
			}
			selection, err := selectEditableUserGroups(nil, []manifest.Object{external, manual})
			require.NoError(t, err)
			assert.Equal(t, []manifest.Object{manual}, selection)
			_, err = selectEditableUserGroups([]string{"external-team"}, []manifest.Object{external})
			require.EqualError(t, err, `UserGroup "external-team" cannot be edited: only manual groups can be changed`)
		})
	}
}
