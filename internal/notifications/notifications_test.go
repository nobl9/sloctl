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

func TestFirstReleaseNote(t *testing.T) {
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
		"bug fix": {
			body: `## 🐞 Bug Fixes

- Fix output formatting (#125) @octocat
`,
			expected: feature{ID: "125", Title: "Fix output formatting"},
			ok:       true,
		},
		"empty section": {
			body: `## 🚀 Features

## 🐞 Bug Fixes

`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, ok := firstReleaseNote(tt.body)

			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestReleaseNotesSection(t *testing.T) {
	tests := map[string]struct {
		body     string
		expected string
		ok       bool
	}{
		"features section": {
			body: `# What's Changed

## 🚀 Features

- Add workflow insights (#123) @octocat
  > Extra release-note detail.

### Details

- Preserves nested feature details.

## 🐞 Bug Fixes

- Fix output formatting (#125) @octocat
`,
			expected: `## 🚀 Features

- Add workflow insights (#123) @octocat
  > Extra release-note detail.

### Details

- Preserves nested feature details.`,
			ok: true,
		},
		"bug fixes section": {
			body: `# What's Changed

## 🧰 Maintenance

- chore: Update dependencies (#124) @renovate

## 🐞 Bug Fixes

- Fix output formatting (#125) @octocat
`,
			expected: `## 🐞 Bug Fixes

- Fix output formatting (#125) @octocat`,
			ok: true,
		},
		"only maintenance": {
			body: `# What's Changed

## 🧰 Maintenance

- chore: Update dependencies (#124) @renovate
`,
		},
		"empty features then bug fixes": {
			body: `# What's Changed

## 🚀 Features

## 🐞 Bug Fixes

- Fix output formatting (#125) @octocat
`,
			expected: `## 🐞 Bug Fixes

- Fix output formatting (#125) @octocat`,
			ok: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, ok := releaseNotesSection(tt.body)

			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.expected, got)
		})
	}
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
	assert.Equal(t, 86, renderWidth)
	assert.Equal(t, strings.Join([]string{
		"### New sloctl updates in v1.2.0!",
		"## 🚀 Features\n\n- feat: Add workflow insights (#123) @octocat",
		"📜 https://github.com/nobl9/sloctl/releases/tag/v1.2.0",
		strings.Join([]string{
			"Update sloctl with:",
			"",
			"```shell",
			"curl -fsSL https://raw.githubusercontent.com/nobl9/sloctl/main/install.bash | bash",
			"```",
		}, "\n"),
	}, "\n\n"), renderedMarkdown)
	plainOutput := stripANSI(out.String())
	assert.Contains(t, plainOutput, "╭")
	assert.Contains(t, plainOutput, "New sloctl updates in v1.2.0!")
	assert.Contains(t, plainOutput, "## 🚀 Features")
	assert.Contains(t, plainOutput, "- feat: Add workflow insights (#123) @octocat")
	assert.Contains(t, plainOutput, "📜 https://github.com/nobl9/sloctl/releases/tag/v1.2.0")
	assert.NotContains(t, plainOutput, "Release notes:")
	assert.Contains(t, plainOutput, "Update sloctl")
	assert.Contains(t, plainOutput, "curl -fsSL https://raw.githubusercontent.com/nobl9/sloctl/main/install.bash | bash")

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
	assert.Contains(t, plainOutput, "New sloctl updates in v1.2.0!")
	assert.Contains(t, plainOutput, "- Add workflow insights (#123) @octocat")
	assert.Contains(t, plainOutput, "📜 https://github.com/nobl9/sloctl/releases/tag/v1.2.0")
	assert.NotContains(t, plainOutput, "Release notes:")
	assert.Contains(t, plainOutput, "Update sloctl with:")
	assert.Contains(t, plainOutput, "curl -fsSL https://raw.githubusercontent.com/nobl9/sloctl/main/install.bash | bash")
}

func TestNotifier_notifyDisplaysVersionWhenReleaseHasNoFeatures(t *testing.T) {
	client := testHTTPClient(func(*http.Request) (int, string) {
		return http.StatusOK, releaseJSON(t, githubRelease{
			TagName: "v1.2.0",
			Body: `# What's Changed

## 🧰 Maintenance

- chore: Update release-drafter/release-drafter action to v7.3.0 (#465) @renovate
`,
			HTMLURL: "https://github.com/nobl9/sloctl/releases/tag/v1.2.0",
		})
	})

	var out bytes.Buffer
	config := testConfig(t, &out, time.Now())
	config.HTTPClient = client
	config.TerminalWidth = func() int { return 80 }
	config.RenderMarkdown = func(string, int) (string, error) {
		t.Fatal("version-only notification should not use markdown rendering")
		return "", nil
	}

	newNotifier(config).notify(context.Background())

	plainOutput := stripANSI(out.String())
	assert.Contains(t, plainOutput, "New sloctl version v1.2.0 is available!")
	assert.Contains(t, out.String(), "\x1b[38;2;255;255;255mNew sloctl version v1.2.0 is available!\x1b[m")
	assert.Contains(t, out.String(), "\x1b[4;38;2;99;214;229;4mh")
	assert.NotContains(t, plainOutput, "###")
	assert.Contains(t, plainOutput, "📜 https://github.com/nobl9/sloctl/releases/tag/v1.2.0")
	assert.NotContains(t, plainOutput, "Release notes:")
	assert.NotContains(t, plainOutput, "View release")
	assert.Contains(t, plainOutput, "Update sloctl")
	assert.Contains(t, out.String(), "\x1b[3;")
	assert.Contains(t, plainOutput, "curl -fsSL \\")
	assert.Contains(t, plainOutput, "https://raw.githubusercontent.com/nobl9/sloctl/main/install.bash \\")
	assert.Contains(t, plainOutput, "| bash")
	assert.NotContains(t, plainOutput, "Maintenance")

	currentState := readState(t, config.CachePath)
	assert.Equal(t, "v1.2.0", currentState.LastShownReleaseTag)
	assert.Empty(t, currentState.LastShownFeatureID)
}

func TestNotifier_notifyKeepsInstallCommandOnOneLineWhenTerminalIsWide(t *testing.T) {
	client := testHTTPClient(func(*http.Request) (int, string) {
		return http.StatusOK, releaseJSON(t, githubRelease{
			TagName: "v1.2.0",
			Body: `# What's Changed

## 🧰 Maintenance

- chore: Update dependencies (#124) @renovate
`,
			HTMLURL: "https://github.com/nobl9/sloctl/releases/tag/v1.2.0",
		})
	})

	var out bytes.Buffer
	config := testConfig(t, &out, time.Now())
	config.HTTPClient = client
	config.TerminalWidth = func() int { return 120 }

	newNotifier(config).notify(context.Background())

	plainOutput := stripANSI(out.String())
	assert.Contains(t, plainOutput, "curl -fsSL https://raw.githubusercontent.com/nobl9/sloctl/main/install.bash | bash")
	assert.NotContains(t, plainOutput, "curl -fsSL \\")
}

func TestNotifier_updateCommand(t *testing.T) {
	tests := map[string]struct {
		executable     string
		resolved       string
		env            map[string]string
		availableTools map[string]bool
		expected       string
	}{
		"homebrew cellar": {
			executable: "/opt/homebrew/bin/sloctl",
			resolved:   "/opt/homebrew/Cellar/sloctl/1.2.0/bin/sloctl",
			expected:   "brew upgrade sloctl",
		},
		"go bin from GOBIN": {
			executable: "/home/me/bin/sloctl",
			env: map[string]string{
				"GOBIN": "/home/me/bin",
			},
			expected: "go install github.com/nobl9/sloctl/cmd/sloctl@latest",
		},
		"go bin from GOPATH": {
			executable: "/home/me/go-work/bin/sloctl",
			env: map[string]string{
				"GOPATH": "/home/me/go-work",
			},
			expected: "go install github.com/nobl9/sloctl/cmd/sloctl@latest",
		},
		"go bin from default GOPATH": {
			executable: "/home/me/go/bin/sloctl",
			env: map[string]string{
				"HOME": "/home/me",
			},
			expected: "go install github.com/nobl9/sloctl/cmd/sloctl@latest",
		},
		"script prefers curl": {
			executable: "/usr/local/bin/sloctl",
			availableTools: map[string]bool{
				"curl": true,
				"wget": true,
			},
			expected: "curl -fsSL https://raw.githubusercontent.com/nobl9/sloctl/main/install.bash | bash",
		},
		"script falls back to wget": {
			executable: "/usr/local/bin/sloctl",
			availableTools: map[string]bool{
				"wget": true,
			},
			expected: "wget -O - -q https://raw.githubusercontent.com/nobl9/sloctl/main/install.bash | bash",
		},
		"script omits command without downloader": {
			executable: "/usr/local/bin/sloctl",
			expected:   "",
		},
		"executable failure uses script default": {
			availableTools: map[string]bool{
				"curl": true,
			},
			expected: "curl -fsSL https://raw.githubusercontent.com/nobl9/sloctl/main/install.bash | bash",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			config := Config{
				Getenv: func(key string) string {
					return tt.env[key]
				},
				Lookup: func(name string) (string, error) {
					if tt.availableTools[name] {
						return "/bin/" + name, nil
					}
					return "", errors.New("not found")
				},
				Executable: func() (string, error) {
					if tt.executable == "" {
						return "", errors.New("no executable")
					}
					return tt.executable, nil
				},
				EvalSymlinks: func(string) (string, error) {
					if tt.resolved == "" {
						return "", errors.New("not a symlink")
					}
					return tt.resolved, nil
				},
			}

			assert.Equal(t, tt.expected, newNotifier(config).updateCommand())
		})
	}
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

	assert.Contains(t, stripANSI(out.String()), "New sloctl updates in v1.2.0!")
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
