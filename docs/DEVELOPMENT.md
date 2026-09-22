# Development

This document describes the intricacies of sloctl development workflow.
If you see anything missing, feel free to contribute :)

## Pull requests

[Pull request template](../.github/pull_request_template.md)
is provided when you create new PR.
Section worth noting and getting familiar with is located under
`## Release Notes` header.

## Makefile

Run `make help` to display short description for each target.
The provided Makefile will automatically install dev dependencies if they're
missing.
Binaries, like `golangci-lint`, and Bats libraries are placed under `bin`,
and `yarn` managed dependencies are installed into `node_modules`.
However, it does not detect if the binary you have is up to date with the
versions declaration located in Makefile.
If you see any discrepancies between CI and your local runs, remove the
binaries from `bin` and let Makefile reinstall them with the latest version.

### Prerequisites

The Makefile does not install the tools listed below.
Make sure they are available in your `PATH` before running the targets
which need them, or the aggregate targets which include them:

- [Go](https://go.dev/doc/install) and git, for most targets.
- [Node.js](https://nodejs.org) and [Yarn](https://classic.yarnpkg.com) 1.x,
  for `check/spell`, `check/trailing`, `check/markdown`, `check/format`
  and `format/cspell`.
  The Node.js version must satisfy the `engines` requirement of the
  dependencies in [package.json](../package.json),
  otherwise `yarn install` fails and reports the required version.
- [Docker](https://docs.docker.com/get-started/get-docker/), for `docker`,
  `test/bats/unit`, `test/bats/e2e` and `test/go/e2e-docker`.
- [jq](https://github.com/jqlang/jq), for `test/bats/e2e`.
- [bats-core](https://github.com/bats-core/bats-core) and Python 3,
  for `test/bats/platform`.
  See [Platform compatibility tests](#platform-compatibility-tests).

## CI

Continuous integration pipelines utilize the same Makefile commands which
you run locally. This ensures consistent behavior of the executed checks
and makes local debugging easier.

## Object model caveat

Sloctl configures the v1alpha parser to use [v1alpha.GenericObject].
This is intentional: sloctl is designed to be object-version agnostic.
Anything returned by the API should be proxied to the user, even if the local
sloctl binary was built before the API gained a new field or object shape.

You MUST NOT rely on v1alpha object types when handling API objects.
Older sloctl versions would not know about recent schema changes, and concrete
types could drop unknown fields or break when the API evolves.

Since [v1alpha.GenericObject] is just a generic `map[string]any`,
it renders type assertions like the below useless:

```go
_, ok := object.(manifest.ProjectScopedObject) // `ok` will always be false
```

Prefer explicit kind-based checks, generic metadata accessors, or existing
helpers that already account for this caveat.

## Testing

In addition to standard unit tests, sloctl is tested with
[bats](https://bats-core.readthedocs.io/en/stable/) framework.
Bats is a testing framework for Bash, it provides a simple way to verify
that shell programs behave as expected.
Bats tests are located under `test` directory.
Each test file ends with `.bats` suffix.
In addition to helper test utilities which are part of the framework we also
provide custom helpers which are located in `test/test_helper` directory.

Bats tests are primarily divided into two categories: end-to-end and unit tests.
The categorization is done through Bats tags.
Platform compatibility tags are orthogonal to those categories and select tests
for the native `make test/bats/platform` target.
To categorize a whole file as a unit test, add
`# bats file_tags=unit` anywhere in the file, preferably just below the shebang.

The end-to-end tests are only run automatically for releases, be it official
version or pre-release (release candidate).
The tests are executed against the production application.
If you want to run the tests manually against a different environment, you can
run the following command:

```shell
SLOCTL_CLIENT_ID=<client_id> \
SLOCTL_CLIENT_SECRET=<client_secret> \
SLOCTL_OKTA_ORG_URL=https://accounts.nobl9.dev \
SLOCTL_OKTA_AUTH_SERVER=<dev_auth_server> \ # Runs against dev Okta.
make test/e2e
```

When any of these variables is not set, `make test/bats/e2e` reads the
missing values from the current context of your sloctl configuration
(`~/.config/nobl9/config.toml` by default,
override it with `SLOCTL_CONFIG_FILE_PATH`).
Variables set in the environment take precedence.
The Go Docker tests (`make test/go/e2e-docker`) do not use this fallback.

Bats unit and end-to-end tests run in containers.
Platform compatibility tests run natively with `make test/bats/platform`.
Refer to the Makefile for the exact commands.

### Platform compatibility tests

`make test/bats/platform` runs Bats directly on your machine,
so bats-core and Python 3 must be installed locally.
For example, on macOS run `brew install bats-core`.

When `BATS_LIB_PATH` is not set, the target downloads the bats-support
and bats-assert versions pinned in the Makefile into `bin/bats-lib`
and loads them from there.
If you set `BATS_LIB_PATH` yourself, it must point at a directory which
contains `bats-support` and `bats-assert` 2.2.0 or newer,
because older bats-assert versions do not provide `assert_stderr`.

The `notification-platforms` and `notification-windows` jobs in
[unit-tests.yml](../.github/workflows/unit-tests.yml) show the full setup
for each platform, including the Windows-only `pywinpty` dependency.

### Bats output assertions

Prefer exact stdout and stderr assertions for complete CLI messages.
Store input fixtures, such as request payloads and release bodies, under
[test/inputs](../test/inputs/), and store expected output fixtures under
[test/outputs](../test/outputs/).
When a test file needs a narrower fixture root, set `TEST_INPUTS` or
`TEST_OUTPUTS` in `setup_file` and compare against files from there.

Use file-backed assertions for expected output:

```bash
assert_output - < "$TEST_OUTPUTS/result.stdout"
assert_stderr - < "$TEST_OUTPUTS/error.stderr"
```

Use `--partial` only as a last resort when exact output would be unstable for
reasons unrelated to the behavior under test, such as nondeterministic fields
that cannot be normalized.
If `--partial` is necessary, keep the assertion narrow and leave nearby context
explaining why a full output fixture would be brittle.

Interactive terminal tests should prefer deterministic plain-text fixtures.
Set `NO_COLOR=1` and use accessible form mode when the command supports it.
Keep source data in [test/inputs](../test/inputs/),
and compare complete stdout or stderr messages against files in
[test/outputs](../test/outputs/).
For notification tests, use the local release fixture server
and `assert_stderr ""` when stderr must be empty.

### End-to-end tests

When creating new e2e tests make sure you adhere to the existing patterns
and use [test helper utility functions](../test/test_helper/load.bash).
The helper functions are documented inline in that file; read them before
adding a new test, especially if you need fixture generation or output
assertion helpers.

Input fixtures for e2e tests live under [test/inputs](../test/inputs/).
The fixture directory name must match the test filename without the `.bats`
suffix.
For example, [test/edit-e2e.bats](../test/edit-e2e.bats) reads input fixtures
from [test/inputs/edit-e2e](../test/inputs/edit-e2e/).

Use `generate_inputs "$BATS_FILE_TMPDIR"` in `setup_file` to prepare the
fixtures for a test file.
This helper copies the matching fixture directory into a temporary directory
and exports `TEST_INPUTS` with the copied path.
Tests should apply from `TEST_INPUTS`, not directly from
[test/inputs](../test/inputs/), because generated fixtures are isolated per
test run.

Use `<PROJECT>` as a placeholder for Project names and project references in
e2e input fixtures.
During `generate_inputs`, every `<PROJECT>` occurrence is replaced with a
unique project name and exported as `TEST_PROJECT`.
The generated name includes the Bats test number, current timestamp, and git
revision, which avoids collisions between local runs and CI retries.

Define project-scoped resources with `project: <PROJECT>` and create a
matching Project fixture with `metadata.name: <PROJECT>`.
For example:

```yaml
- apiVersion: n9/v1alpha
  kind: Project
  metadata:
    name: <PROJECT>
  spec:
    description: Project for e2e tests
- apiVersion: n9/v1alpha
  kind: Service
  metadata:
    name: example-service
    project: <PROJECT>
  spec:
    description: Service for e2e tests
```

After calling `generate_inputs`, use `"$TEST_PROJECT"` in commands that need
the generated project name:

```bash
run_sloctl get service example-service -p "$TEST_PROJECT"
```

#### Debugging end-to-end tests

When developing new tests or debugging existing ones, you can utilize
`bats` tags, specifically the `bats:focus` tag.
This tag can be added both on file and test levels:

```bash
#!/usr/bin/env bash
# bats file_tags=bats:focus   <-- This will only run this file

# bats test_tags=bats:focus   <-- This will only run this test
@test "sloctl does something" {}
```

You can read more about it in
[bats docs](https://bats-core.readthedocs.io/en/stable/writing-tests.html#focusing-on-tests-with-bats-focus-tag).

## MCP

Sloctl proxies to an [MCP server](https://modelcontextprotocol.io/quickstart/server)
which can be used by LLMs to interact with Nobl9 platform and its resources.

In order to help develop the server's capabilities and test them you can use
a tool called [MCP inspector](https://modelcontextprotocol.io/docs/tools/inspector).

Simply run:

```bash
make install # So that sloctl is available in the path.
npx @modelcontextprotocol/inspector@latest --config ./docs/mcp.json --server nobl9
```

## Releases

Refer to [RELEASE.md](./RELEASE.md) for more information on release process.

## Dependencies

Renovate is configured to automatically merge minor and patch updates.
For major versions, which sadly includes GitHub Actions, manual approval
is required.

---

[v1alpha.GenericObject]: https://pkg.go.dev/github.com/nobl9/nobl9-go@v0.127.0/manifest/v1alpha#GenericObject
