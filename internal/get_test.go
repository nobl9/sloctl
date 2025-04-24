//go:build unit_test

package internal

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authDataV1 "github.com/nobl9/nobl9-go/sdk/endpoints/authdata/v1"

	"github.com/nobl9/sloctl/internal/printer"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobl9/nobl9-go/manifest"
	v1alphaParser "github.com/nobl9/nobl9-go/manifest/v1alpha/parser"
	"github.com/nobl9/nobl9-go/sdk"

	"github.com/nobl9/nobl9-go/manifest/v1alpha"
)

//go:embed test_data/agent_with_keys_response.yaml
var agentWithKeysResponse []byte

func TestGet_AgentKeys(t *testing.T) {
	rt := mockRoundTripper{
		T: t,
		GetAgentResponse: []v1alpha.GenericObject{
			{
				"apiVersion": manifest.VersionV1alpha,
				"kind":       manifest.KindAgent,
				"metadata": map[string]interface{}{
					"name":    "obi-wan",
					"project": "secret-mission",
				},
			},
			{
				"apiVersion": manifest.VersionV1alpha,
				"kind":       manifest.KindAgent,
				"metadata": map[string]interface{}{
					"name":    "luke-skywalker",
					"project": "jedi-training",
				},
			},
		},
		GetAgentCredsResponse: map[string]authDataV1.M2MAppCredentials{
			"secret-mission": {ClientID: "super-secret-obi", ClientSecret: "even-more-secret-obi!"},
			"jedi-training":  {ClientID: "super-secret-luke", ClientSecret: "even-more-secret-luke!"},
		},
	}

	client, err := sdk.NewClient(&sdk.Config{
		DisableOkta: true,
		Project:     sdk.ProjectsWildcard,
	})
	require.NoError(t, err)
	client.HTTP = &http.Client{Transport: rt}
	var out bytes.Buffer
	g := GetCmd{
		client: client,
		printer: printer.NewPrinter(printer.Config{
			Output:       &out,
			OutputFormat: printer.YAMLFormat,
		}),
	}

	cmd := cobra.Command{}
	g.newGetAgentCommand(&cmd)
	f := cmd.Flag("with-keys")
	require.NoError(t, f.Value.Set("true"))
	v1alphaParser.UseGenericObjects = true
	err = cmd.Execute()
	v1alphaParser.UseGenericObjects = false
	require.NoError(t, err)

	assert.Equal(t, "*", g.client.Config.Project, "Project must not be overwritten")
	assert.YAMLEq(t, string(agentWithKeysResponse), out.String())
}

type mockRoundTripper struct {
	T                     *testing.T
	GetAgentResponse      []v1alpha.GenericObject
	GetAgentCredsResponse map[string]authDataV1.M2MAppCredentials
}

func (m mockRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	m.T.Helper()
	rec := httptest.NewRecorder()
	switch r.URL.Path {
	case "/get/agent":
		data, err := json.Marshal(m.GetAgentResponse)
		if err != nil {
			return nil, err
		}
		_, _ = rec.Write(data)
	case "/internal/agent/clientcreds":
		split := strings.Split(r.URL.RawQuery, "=")
		require.Len(m.T, split, 2, "expected exactly one query parameter")
		projectName := r.Header.Get(sdk.HeaderProject)
		data, err := json.Marshal(m.GetAgentCredsResponse[projectName])
		if err != nil {
			return nil, err
		}
		_, _ = rec.Write(data)
	default:
		fmt.Println(r.URL)
		panic("implement me")
	}
	return rec.Result(), nil
}
