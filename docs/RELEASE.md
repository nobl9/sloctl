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
- `DOCS_AUTOMATION_CLIENT_ID` variable:
  Client ID of a dedicated GitHub App installed on the destination repository.

Create a `sloctl-docs-dispatch` environment in the sloctl repository.
Under **Deployment branches and tags**, choose **Selected branches and tags**
and add only the **branch** `main`.
Do not add tag patterns or other branches.
Leave required reviewers empty and set no wait timer.
The dispatcher uses `workflow_run`, whose workflow ref is the default branch,
even when the completed release ran from a tag.

Store `DOCS_AUTOMATION_APP_PRIVATE_KEY` only as an environment secret
in `sloctl-docs-dispatch`.
Do not expose this key through a repository secret
or an organization secret accessible to sloctl.
Keep `main` protected and require review of workflow changes.
The environment restriction prevents feature-branch workflows from reading the key.
If the key was previously available outside the environment,
rotate it and update both repositories before revoking the old key.

The App installation needs **Actions: write**,
**Checks: read**, **Contents: write**,
and **Pull requests: write** permissions.
The dispatcher requests only **Actions: write** for its short-lived token.
This limits that token, not the authority of the private key.
The documentation workflow separately requests the checks, content,
and pull-request permissions when publishing the generated update.
The App must also be listed under
**Allow specified actors to bypass required pull requests**
in the documentation repository's `production` branch protection rule.
This removes the human approval requirement for the automated update
without bypassing its required status checks.
