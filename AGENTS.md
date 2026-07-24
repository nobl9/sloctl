# AGENTS.md

This repository contains `sloctl`, the Go command-line interface for Nobl9.
Do not use this file as a replacement for the project documentation.

Read the existing docs before changing behavior, tests, or release logic:

- [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md) for development workflow,
  Makefile behavior, CI, Bats conventions, MCP development, and dependencies.
- [README.md](./README.md) for user-facing CLI purpose and usage.
- [docs/RELEASE.md](./docs/RELEASE.md) for release automation details.
- [docs/mcp.json](./docs/mcp.json) for the MCP inspector configuration.

If a workflow is documented there, follow the existing doc instead of adding
a second version here.

## Testing

Write end-to-end tests over unit tests by default.
For behavior visible through the `sloctl` CLI, add or update Bats e2e tests
under [test](./test/) unless there is a concrete reason not to.

Unit tests are acceptable for narrow internal logic, parser edge cases,
or failure branches that cannot be exercised through the CLI without brittle
setup or excessive external state.
If you choose unit-only coverage for a behavior change, state the reason in
the PR or handoff notes.

End-to-end tests talk to the Nobl9 platform API.
Do not run them without explicit user permission.

Before writing or modifying Bats tests, read:

- [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md#testing)
- [test/test_helper/load.bash](./test/test_helper/load.bash)
- [test/setup_suite.bash](./test/setup_suite.bash)
- sample existing tests to follow the established style and practices

When iterating over something and testing via e2e tests, use bats focus flags,
which help isolate specific tests or files:

```bash
#!/usr/bin/env bash
# bats file_tags=bats:focus   <-- This will only run this file

# bats test_tags=bats:focus   <-- This will only run this test
@test "sloctl does something" {}
```

## Code standards

Follow existing package layout, command patterns, and test style before adding
new abstractions.

Do not edit generated files directly.
If generated output is stale, update the source definitions and run
`make generate`, then verify with `make check/generate`.

### Shell

Use the Makefile targets instead of calling tools directly.
To inspect available targets, run: `make help`.
The CI workflows under [.github/workflows](./.github/workflows/) use the same
Makefile targets, so treat them as the local source of verification commands.

## Pull requests

When creating or updating a pull request description,
follow guidelines and template defined in
[.github/pull_request_template.md](./.github/pull_request_template.md).
Do not introduce new sections, subsections for the existing ones are acceptable.
Fill only the sections that apply, remove template instructions from the final
description, and remove the `## Release Notes` section entirely when the change
does not need release notes.
Always ask the use for for `## Motivation`,
unless you already know it from a spec or ticket.
Be vigilant of any breaking changes and document them in `## Breaking Changes` section.

## Verification

Always verify changes with project targets before claiming completion.
For Markdown-only changes, run `make check/markdown` at minimum.

If a command cannot be run locally, report the exact command and exact error.
Do not replace failed verification with assumptions.
