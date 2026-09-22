//go:build e2e_test

package dockertest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"

	"github.com/nobl9/nobl9-go/manifest"
	v1alphaProject "github.com/nobl9/nobl9-go/manifest/v1alpha/project"
	"github.com/nobl9/nobl9-go/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	once        sync.Once
	dockerImage = "sloctl"
	moduleRoot  = findModuleRoot()
	tpl         = template.Must(
		template.New("project.tmpl.yaml").
			ParseFiles(filepath.Join(moduleRoot, "test", "inputs", "dockertest", "project.tmpl.yaml")),
	)
)

func TestDocker(t *testing.T) {
	setup(t)
	projectName := fmt.Sprintf("sloctl-docker-e2e-test-%d", time.Now().UnixNano())

	var (
		applyBuf  bytes.Buffer
		deleteBuf bytes.Buffer
	)
	tplBuf := io.MultiWriter(&applyBuf, &deleteBuf)
	err := tpl.Execute(tplBuf, map[string]interface{}{"Name": projectName})
	require.NoError(t, err)

	t.Log("Applying:\n", applyBuf.String())
	out := runSloctl(t, &applyBuf, "apply", "-f", "-")
	assert.Equal(t, "The resources were successfully applied.\n", out.String())

	t.Log("Fetching Project:", projectName)
	out = runSloctl(t, nil, "get", "project", projectName)
	objects, err := sdk.ReadObjectsFromSources(context.Background(), sdk.NewObjectSourceReader(out, ""))
	require.NoError(t, err)
	projects := manifest.FilterByKind[v1alphaProject.Project](objects)
	assert.Len(t, projects, 1)
	project := projects[0]
	assert.NotNil(t, project.Spec.CreatedAt)
	assert.NotNil(t, project.Spec.CreatedBy)
	project.Spec.CreatedAt = ""
	project.Spec.CreatedBy = ""
	assert.Equal(t, v1alphaProject.Project{
		APIVersion: manifest.VersionV1alpha,
		Kind:       manifest.KindProject,
		Metadata: v1alphaProject.Metadata{
			Name:        projectName,
			DisplayName: projectName,
			Labels: map[string][]string{
				"origin": {"sloctl-e2e-tests"},
			},
			Annotations: map[string]string{
				"team": "green",
			},
		},
		Spec: v1alphaProject.Spec{
			Description: "Dummy Project for sloctl docker image e2e tests",
		},
	}, project)

	t.Log("Deleting Project:", projectName)
	out = runSloctl(t, &deleteBuf, "delete", "-f", "-")
	assert.Equal(t, "The resources were successfully deleted.\n", out.String())

	t.Log("Fetching Project:", projectName, "in order to ensure it was deleted")
	out = runSloctl(t, nil, "get", "project", projectName)
	assert.Equal(t, "No resources found.\n", out.String())
}

func setup(t *testing.T) {
	t.Helper()
	once.Do(func() {
		setupImage(t)
		setupCredentials(t)
	})
}

func setupImage(t *testing.T) {
	t.Helper()
	if imageOverride := os.Getenv("SLOCTL_E2E_DOCKER_TEST_IMAGE"); imageOverride != "" {
		dockerImage = strings.Replace(imageOverride, ":v", ":", 1)
		t.Log("Using sloctl image override:", dockerImage)
		return
	}
	gitBranch := mustExecCmd(t, exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")).String()
	gitRevision := mustExecCmd(t, exec.Command("git", "rev-parse", "--short=8", "HEAD")).String()
	dockerImage = "sloctl-docker-test"
	_ = mustExecCmd(t, exec.Command("docker",
		"build",
		"--build-arg",
		fmt.Sprintf(`LDFLAGS=-X %[1]s.BuildVersion=%[2]s -X %[1]s.BuildGitBranch=%[3]s -X %[1]s.BuildGitRevision=%[4]s`,
			"github.com/nobl9/sloctl/internal", "", gitBranch, gitRevision),
		"-t", dockerImage,
		"../../.",
	))
}

var sloctlEnvVars = []string{
	"SLOCTL_CLIENT_ID",
	"SLOCTL_CLIENT_SECRET",
	"SLOCTL_OKTA_ORG_URL",
	"SLOCTL_OKTA_AUTH_SERVER",
}

// sloctlEnv holds the values of sloctlEnvVars passed to the sloctl container.
var sloctlEnv = make(map[string]string, len(sloctlEnvVars))

// setupCredentials takes the credentials from the environment
// and fills the missing ones from the current sloctl context, like scripts/run-e2e-tests.sh.
func setupCredentials(t *testing.T) {
	t.Helper()
	missing := false
	for _, env := range sloctlEnvVars {
		sloctlEnv[env] = os.Getenv(env)
		if sloctlEnv[env] == "" {
			missing = true
		}
	}
	if missing {
		fillCredentialsFromCurrentContext(t)
	}
	if sloctlEnv["SLOCTL_CLIENT_ID"] == "" || sloctlEnv["SLOCTL_CLIENT_SECRET"] == "" {
		t.Fatal("SLOCTL_CLIENT_ID and SLOCTL_CLIENT_SECRET must be set or available in the current sloctl context")
	}
}

func fillCredentialsFromCurrentContext(t *testing.T) {
	t.Helper()
	configPath := os.Getenv("SLOCTL_CONFIG_FILE_PATH")
	if configPath == "" {
		home, err := os.UserHomeDir()
		require.NoError(t, err)
		configPath = filepath.Join(home, ".config", "nobl9", "config.toml")
	}
	// Docker would create a directory in place of a missing bind mount source.
	//nolint:gosec // The config path is provided by the developer running the tests.
	if _, err := os.Stat(configPath); err != nil {
		return
	}
	t.Log("Reading missing credentials from the current context in", configPath)
	out := mustExecCmd(t, exec.Command("docker", "run", "--rm",
		"-v", configPath+":/config.toml:ro",
		dockerImage,
		"config", "current-context", "--verbose", "--show-secret", "-o", "json", "--config=/config.toml",
	))
	var current struct {
		ClientID       string `json:"clientId"`
		ClientSecret   string `json:"clientSecret"`
		OktaOrgURL     string `json:"oktaOrgUrl"`
		OktaAuthServer string `json:"oktaAuthServer"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &current))
	for env, value := range map[string]string{
		"SLOCTL_CLIENT_ID":        current.ClientID,
		"SLOCTL_CLIENT_SECRET":    current.ClientSecret,
		"SLOCTL_OKTA_ORG_URL":     current.OktaOrgURL,
		"SLOCTL_OKTA_AUTH_SERVER": current.OktaAuthServer,
	} {
		if sloctlEnv[env] == "" {
			sloctlEnv[env] = value
		}
	}
}

func runSloctl(t *testing.T, input io.Reader, sloctlArgs ...string) *bytes.Buffer {
	args := make([]string, 0, 3+2*len(sloctlEnvVars)+1+len(sloctlArgs))
	args = append(args, "run", "-i", "--rm")
	for _, env := range sloctlEnvVars {
		args = append(args, "-e", fmt.Sprintf("%s=%s", env, sloctlEnv[env]))
	}
	args = append(args, dockerImage)
	args = append(args, sloctlArgs...)
	cmd := exec.Command("docker", args...)
	if input != nil {
		cmd.Stdin = input
	}
	return mustExecCmd(t, cmd)
}

func mustExecCmd(t *testing.T, cmd *exec.Cmd) *bytes.Buffer {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if cmd.Stdout == nil {
		cmd.Stdout = &stdout
	}
	if cmd.Stderr == nil {
		cmd.Stderr = &stderr
	}
	if err := cmd.Run(); err != nil {
		cmdStr := cmd.String()
		if secret := sloctlEnv["SLOCTL_CLIENT_SECRET"]; secret != "" {
			cmdStr = strings.ReplaceAll(cmdStr, secret, "***")
		}
		t.Fatalf("Failed to execute '%s' command: %s", cmdStr, stderr.String())
	}
	return &stdout
}

// findModuleRoot returns the absolute path to the modules root.
func findModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	dir = filepath.Clean(dir)
	for {
		if fi, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !fi.IsDir() {
			return dir
		}
		d := filepath.Dir(dir)
		if d == dir {
			break
		}
		dir = d
	}
	return ""
}
