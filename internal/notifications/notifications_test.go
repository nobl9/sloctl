package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFirstFeature(t *testing.T) {
	tests := map[string]struct {
		body     string
		expected feature
		ok       bool
	}{
		"release drafter features section": {
			body: `# What's Changed

## 🚀 Features

- feat: Add workflow insights (#123) @octocat
- Improve import behavior (#124) @hubot

## 🐞 Bug Fixes

- Fix output formatting (#125) @octocat
`,
			expected: feature{ID: "123", Title: "Add workflow insights"},
			ok:       true,
		},
		"feature with release notes blockquote": {
			body: `# What's Changed

## 🚀 Features

- Add replay controls (#222) @octocat
  > Release note text is not part of the notification title.

## 🧰 Maintenance

- Update dependencies (#223) @renovate
`,
			expected: feature{ID: "222", Title: "Add replay controls"},
			ok:       true,
		},
		"feature without author": {
			body: `## Features

- Add direct upload (#333)
`,
			expected: feature{ID: "333", Title: "Add direct upload"},
			ok:       true,
		},
		"missing features section": {
			body: `## 🐞 Bug Fixes

- Fix output formatting (#125) @octocat
`,
		},
		"empty features section": {
			body: `## 🚀 Features

## 🐞 Bug Fixes

- Fix output formatting (#125) @octocat
`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, ok := firstFeature(tt.body)

			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestFeaturesSection(t *testing.T) {
	body := `# What's Changed

## 🚀 Features

- Add workflow insights (#123) @octocat
  > Extra release-note detail.

### Details

- Preserves nested feature details.

## 🐞 Bug Fixes

- Fix output formatting (#125) @octocat
`

	got, ok := featuresSection(body)

	require.True(t, ok)
	assert.Equal(t, `## 🚀 Features

- Add workflow insights (#123) @octocat
  > Extra release-note detail.

### Details

- Preserves nested feature details.`, got)
}

func TestNotifier_notifyDisplaysFeatureAndCaches(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	requests := 0
	client := testHTTPClient(func(r *http.Request) (int, string) {
		requests++
		assert.Equal(t, "application/vnd.github+json", r.Header.Get("Accept"))
		assert.Equal(t, "sloctl", r.Header.Get("User-Agent"))
		return http.StatusOK, releaseJSON(t, githubRelease{
			TagName: "v1.2.0",
			Body: `## 🚀 Features

- feat: Add workflow insights (#123) @octocat
`,
			HTMLURL: "https://github.com/nobl9/sloctl/releases/tag/v1.2.0",
		})
	})

	var out bytes.Buffer
	config := testConfig(t, &out, now)
	config.HTTPClient = client
	var renderedMarkdown string
	var renderWidth int
	config.RenderMarkdown = func(markdown string, width int) (string, error) {
		renderedMarkdown = markdown
		renderWidth = width
		return markdown, nil
	}

	newNotifier(config).notify(context.Background())

	assert.Equal(t, 1, requests)
	assert.Equal(t, 82, renderWidth)
	assert.Equal(t, `### New sloctl features in v1.2.0

## 🚀 Features

- feat: Add workflow insights (#123) @octocat

[View release](https://github.com/nobl9/sloctl/releases/tag/v1.2.0)`, renderedMarkdown)
	plainOutput := stripANSI(out.String())
	assert.Contains(t, plainOutput, "╭")
	assert.Contains(t, plainOutput, "New sloctl features in v1.2.0")
	assert.Contains(t, plainOutput, "## 🚀 Features")
	assert.Contains(t, plainOutput, "- feat: Add workflow insights (#123) @octocat")
	assert.Contains(t, plainOutput, "[View release](https://github.com/nobl9/sloctl/releases/tag/v1.2.0)")

	out.Reset()
	newNotifier(config).notify(context.Background())

	assert.Equal(t, 1, requests)
	assert.Empty(t, out.String())
}

func TestNotifier_notifyFallsBackWhenMarkdownRenderingFails(t *testing.T) {
	client := testHTTPClient(func(*http.Request) (int, string) {
		return http.StatusOK, releaseJSON(t, githubRelease{
			TagName: "v1.2.0",
			Body: `## 🚀 Features

- Add workflow insights (#123) @octocat
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
	assert.Contains(t, plainOutput, "New sloctl features in v1.2.0")
	assert.Contains(t, plainOutput, "- Add workflow insights (#123) @octocat")
	assert.Contains(t, plainOutput, "https://github.com/nobl9/sloctl/releases/tag/v1.2.0")
}

func TestNotifier_notifySkipsBeforeFetch(t *testing.T) {
	tests := map[string]struct {
		configure func(Config) Config
	}{
		"ci": {
			configure: func(config Config) Config {
				config.Getenv = func(key string) string {
					if key == ciEnv {
						return "true"
					}
					return ""
				}
				return config
			},
		},
		"development version": {
			configure: func(config Config) Config {
				config.CurrentVersion = "1.0.0-test"
				return config
			},
		},
		"non tty": {
			configure: func(config Config) Config {
				config.IsTTY = func() bool { return false }
				return config
			},
		},
		"opt out": {
			configure: func(config Config) Config {
				config.Getenv = func(key string) string {
					if key == optOutEnv {
						return "1"
					}
					return ""
				}
				return config
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			requests := 0
			config := testConfig(t, &out, time.Now())
			config.HTTPClient = testHTTPClient(func(*http.Request) (int, string) {
				requests++
				return http.StatusOK, "{}"
			})
			config = tt.configure(config)

			newNotifier(config).notify(context.Background())

			assert.Zero(t, requests)
			assert.Empty(t, out.String())
		})
	}
}

func TestNotifier_notifySkipsCurrentRelease(t *testing.T) {
	requests := 0
	client := testHTTPClient(func(*http.Request) (int, string) {
		requests++
		return http.StatusOK, releaseJSON(t, githubRelease{
			TagName: "v1.2.0",
			Body: `## 🚀 Features

- Add workflow insights (#123) @octocat
`,
			HTMLURL: "https://github.com/nobl9/sloctl/releases/tag/v1.2.0",
		})
	})

	var out bytes.Buffer
	config := testConfig(t, &out, time.Now())
	config.CurrentVersion = "1.2.0"
	config.HTTPClient = client

	newNotifier(config).notify(context.Background())

	assert.Equal(t, 1, requests)
	assert.Empty(t, out.String())
}

func TestNotifier_notifySuppressesFetchFailuresAndCachesCheck(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	requests := 0
	client := testHTTPClient(func(*http.Request) (int, string) {
		requests++
		return http.StatusForbidden, "rate limited"
	})

	var out bytes.Buffer
	config := testConfig(t, &out, now)
	config.HTTPClient = client

	newNotifier(config).notify(context.Background())
	config.Now = func() time.Time { return now.Add(time.Hour) }
	newNotifier(config).notify(context.Background())

	assert.Equal(t, 1, requests)
	assert.Empty(t, out.String())

	currentState := readState(t, config.CachePath)
	assert.Equal(t, now, currentState.LastCheckedAt)
}

func TestNotifier_notifySuppressesMalformedRelease(t *testing.T) {
	client := testHTTPClient(func(*http.Request) (int, string) {
		return http.StatusOK, "{"
	})

	var out bytes.Buffer
	config := testConfig(t, &out, time.Now())
	config.HTTPClient = client

	newNotifier(config).notify(context.Background())

	assert.Empty(t, out.String())
}

func TestNotifier_notifyDoesNotSurfaceCacheWriteFailure(t *testing.T) {
	client := testHTTPClient(func(*http.Request) (int, string) {
		return http.StatusOK, releaseJSON(t, githubRelease{
			TagName: "v1.2.0",
			Body: `## 🚀 Features

- Add workflow insights (#123) @octocat
`,
			HTMLURL: "https://github.com/nobl9/sloctl/releases/tag/v1.2.0",
		})
	})

	var out bytes.Buffer
	config := testConfig(t, &out, time.Now())
	config.HTTPClient = client
	config.CachePath = t.TempDir()

	newNotifier(config).notify(context.Background())

	assert.Contains(t, stripANSI(out.String()), "New sloctl features in v1.2.0")
	assert.Contains(t, stripANSI(out.String()), "Add workflow insights")
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
		TerminalWidth:  func() int { return defaultBoxWidth },
		RenderMarkdown: func(markdown string, _ int) (string, error) { return markdown, nil },
	}
}

func releaseJSON(t *testing.T, release githubRelease) string {
	t.Helper()
	data, err := json.Marshal(release)
	require.NoError(t, err)
	return string(data)
}

func readState(t *testing.T, path string) state {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var currentState state
	require.NoError(t, json.Unmarshal(data, &currentState))
	return currentState
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
