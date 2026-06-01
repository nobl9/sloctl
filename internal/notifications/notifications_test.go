package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotifier_notifyFallsBackWhenMarkdownRenderingFails(t *testing.T) {
	client := testHTTPClient(func(*http.Request) (int, string) {
		return http.StatusOK, releaseJSON(t, githubRelease{
			TagName: "v1.2.0",
			Body: `## 🚀 Features

- Add workflow insights (#123) @bot
`,
			HTMLURL: "https://github.com/nobl9/sloctl/releases/tag/v1.2.0",
		})
	})

	var out bytes.Buffer
	config := testConfig(t, &out, time.Now())
	config.HTTPClient = client
	config.RenderMarkdown = func(string, int) (string, error) {
		return "", errors.New("render failed")
	}

	newNotifier(config).notify(context.Background())

	plainOutput := stripANSI(out.String())
	assert.Contains(t, plainOutput, "New sloctl updates in v1.2.0!")
	assert.Contains(t, plainOutput, "- Add workflow insights (#123) @bot")
	assert.Contains(t, plainOutput, "📜 https://github.com/nobl9/sloctl/releases/tag/v1.2.0")
	assert.NotContains(t, plainOutput, "Release notes:")
	assert.Contains(t, plainOutput, "Update sloctl with:")
	assert.Contains(t, plainOutput, "curl -fsSL https://raw.githubusercontent.com/nobl9/sloctl/main/install.bash | bash")
}

func TestNotifier_notifySkipsDevelopmentVersion(t *testing.T) {
	var out bytes.Buffer
	requests := 0
	config := testConfig(t, &out, time.Now())
	config.CurrentVersion = "1.0.0-test"
	config.HTTPClient = testHTTPClient(func(*http.Request) (int, string) {
		requests++
		return http.StatusOK, "{}"
	})

	newNotifier(config).notify(context.Background())

	assert.Zero(t, requests)
	assert.Empty(t, out.String())
}

func testConfig(t *testing.T, out *bytes.Buffer, now time.Time) Config {
	t.Helper()
	return Config{
		CurrentVersion: "1.1.0",
		Stderr:         out,
		ReleaseURL:     "https://example.com/repos/nobl9/sloctl/releases/latest",
		CachePath:      filepath.Join(t.TempDir(), "notifications.json"),
		Now:            func() time.Time { return now },
		Getenv:         func(string) string { return "" },
		IsTTY:          func() bool { return true },
		Lookup:         func(name string) (string, error) { return "/bin/" + name, nil },
		TerminalWidth:  func() int { return defaultBoxWidth },
		RenderMarkdown: func(markdown string, _ int) (string, error) { return markdown, nil },
		Executable:     func() (string, error) { return "/usr/local/bin/sloctl", nil },
		EvalSymlinks:   func(path string) (string, error) { return path, nil },
	}
}

func releaseJSON(t *testing.T, release githubRelease) string {
	t.Helper()
	data, err := json.Marshal(release)
	require.NoError(t, err)
	return string(data)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testHTTPClient(handler func(*http.Request) (int, string)) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		statusCode, body := handler(req)
		return &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}
