# Release process

The internal release process is described in great detail
[here](http://go/sloctl-release).

## Release automation details

We're using [Release Drafter](https://github.com/release-drafter/release-drafter)
to automate release notes creation. Drafter also does its best to propose
the next release version based on commit messages from `main` branch.

Release Drafter is also responsible for auto-labeling pull requests.
It checks both title and body of the pull request and adds appropriate labels. \
**NOTE:** The auto-labeling mechanism will not remove labels once they're
created. For example, If you end up changing PR title from `sec:` to `fix:`
you'll have to manually remove `security` label.

On each commit to `main` branch, Release Drafter will update the next release
draft.

In addition to Release Drafter, we're also running a script which extracts
explicitly listed release notes and breaking changes which are optionally
defined in `## Release Notes` and `## Breaking Changes` headers.
It also performs a cleanup of the PR draft mitigating Release Drafter
shortcomings.

## Command reference synchronization

The [Sync sloctl docs](../.github/workflows/sync-sloctl-docs.yml) workflow
runs after each successful official `Release` run.
It updates the published command reference.
Release candidates do not trigger synchronization.

The workflow uses a short-lived GitHub App token to dispatch the documentation workflow.
It passes the release tag, commit SHA, and release run ID.
The documentation workflow verifies the release
and reads the command-reference JSON from that exact commit.
It renders the MDX and opens a pull request against `production`.
After the required checks pass, it squash-merges the update.
It then waits for the production deployment of that merge commit.

Synchronization succeeds only after the documentation workflow finishes successfully,
including deployment.
A failure in the update or deployment fails synchronization
without changing the completed CLI release.

The [synchronization script](../.github/actions/sync-sloctl-release/sync-sloctl-release.sh)
allows up to three attempts for each GitHub API read and for monitoring.
The job has a 55-minute timeout to stay within the App token's lifetime.
A rerun of `Sync sloctl docs` uses the release run ID to find the previous attempt:

- If the documentation run succeeded, synchronization finishes without another update.
- If the documentation run is active, synchronization resumes monitoring it.
- If the documentation run failed, synchronization dispatches a new update.

## GitHub Action synchronization

[Sync nobl9-action](../.github/workflows/sync-nobl9-action.yml) also runs after
each successful official `Release` run.
It uses the same dispatcher and monitors the update
independently of documentation synchronization.
The completed CLI release is unaffected if either downstream update fails.

The sender passes the release tag, commit SHA, and source run ID to
`update-sloctl.yml` on `nobl9/nobl9-action`'s `main` branch.
That workflow verifies the release, updates the bundled image and README examples,
and opens a PR.
After the action checks pass, it merges the PR and checks the merge commit.
It then publishes the next patch release of `nobl9-action`.
The action's version is independent of the sloctl version.
Synchronization succeeds only after publication succeeds.

Install the GitHub App on `nobl9-action`.
The existing documentation automation App can serve both workflows,
with each token scoped to its target repository.
Create the `nobl9-action-dispatch` environment in this repository with:

| Name | Type | Value |
| --- | --- | --- |
| `SLOCTL_AUTOMATION_CLIENT_ID` | Variable | GitHub App client ID |
| `SLOCTL_AUTOMATION_APP_PRIVATE_KEY` | Secret | GitHub App private key |

The sender requests Actions write and Checks read permissions.
Restrict the environment to the default branch.
Before enabling the sender, deploy the receiver and configure its App access,
environment, review bypass, and required checks.
See the [action's development guide](https://github.com/nobl9/nobl9-action/blob/main/docs/DEVELOPMENT.md#automation-setup).

The dispatcher resumes an active run, skips a successful run,
and dispatches another attempt after a failed run.
The receiver skips older sloctl versions and resumes unfinished publication.
If monitoring times out, the sender fails without cancelling
the queued or active action update.
Rerun synchronization to resume monitoring.
The action receiver owns the sloctl image update,
which is excluded from its Renovate configuration.

Run `make test/automation` for the offline dispatcher tests.
These tests need Bats, bats-support, bats-assert, jq, and GNU coreutils.
Set `BATS_LIB_PATH` if the assertion libraries are outside the Bats search path.
The containerized `make test/bats/unit` target includes the same tests.
Run `make check/automation` for workflow and shell validation.
This check needs actionlint, ShellCheck, and shfmt.
