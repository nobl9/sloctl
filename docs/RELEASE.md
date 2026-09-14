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

After the complete `Release` workflow succeeds,
the `Sync sloctl docs` workflow requests an update of the published
command reference and waits for the documentation workflow to finish.
If the downstream update fails,
the synchronization workflow fails without changing the completed release.
The release-candidate workflow does not trigger this synchronization.
The documentation workflow validates the successful release run,
official tag,
and commit,
reads the generated command-reference JSON from that exact release,
renders the MDX in the documentation repository,
opens a pull request,
waits for the exact required checks,
automatically squash-merges the update,
and verifies that the exact merge commit's production `build-deploy` run succeeds.
The dispatcher retries a failed documentation run up to three times
to recover from concurrent changes to the documentation repository.

The dispatcher requires the following repository configuration:

- `DOCS_AUTOMATION_REPOSITORY` variable:
  destination repository name without the owner.
- `DOCS_AUTOMATION_APP_ID` variable:
  App ID of a dedicated GitHub App installed on the destination repository.
- `DOCS_AUTOMATION_APP_PRIVATE_KEY` secret:
  private key for the same GitHub App.

The App installation needs **Actions: write**,
**Checks: read**, **Contents: write**,
and **Pull requests: write** permissions.
The dispatcher requests only **Actions: write** for its short-lived token;
the documentation workflow separately requests the checks, content,
and pull-request permissions when publishing the generated update.
The App must also be listed under
**Allow specified actors to bypass required pull requests**
in the documentation repository's `production` branch protection rule.
This removes the human approval requirement for the automated update
without bypassing its required status checks.
