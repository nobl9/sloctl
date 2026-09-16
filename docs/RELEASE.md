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

The [synchronization script](../.github/actions/sync-sloctl-docs/sync-sloctl-docs.sh)
allows up to three attempts for each GitHub API read and for monitoring.
The job has a 55-minute timeout to stay within the App token's lifetime.
A rerun of `Sync sloctl docs` uses the release run ID to find the previous attempt:

- If the documentation run succeeded, synchronization finishes without another update.
- If the documentation run is active, synchronization resumes monitoring it.
- If the documentation run failed, synchronization dispatches a new update.
