# Development

This document describes the intricacies of sloctl development workflow.

## Makefile

Run `make help` to display short description for each target.
The provided Makefile will automatically install dev dependencies if they're
missing and place them under `bin`
(this does not apply to `yarn` managed dependencies).
However, it does not detect if the binary you have is up to date with the
versions declaration located in Makefile.
If you see any discrepancies between CI and your local runs, remove the
binaries from `bin` and let Makefile reinstall them with the latest version.

## Testing

In addition to standard unit tests, sloctl is tested with
[bats](https://bats-core.readthedocs.io/en/stable/) framework.
Bats is a testing framework for Bash, it provides a simple way to verify
that shell programs behave as expected.
Bats tests are located under `test` directory.
Each test file ends with `.bats` suffix.
In addition to helper test utilities which are part of the framework we also
provide custom helpers which are located in `test/test_helper` directory.

Bats tests are currently divided into 2 categories, end-to-end and unit tests.
The categorization is done through Bats tags. In order to categorize a whole
file as a unit test, add this comment: `# bats file_tags=unit` anywhere in the
file, preferably just below shebang.

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

Bats tests are fully containerized, refer to Makefile for more details on
how they're executed.

## Releases

We're using [Release Drafter](https://github.com/release-drafter/release-drafter)
to automate release notes creation. Drafter also does its best to propose
the next release version based on commit messages from `main` branch.

Release Drafter is also responsible for auto-labeling of pull requests.
It checks both title and body of pull request and adds appropriate labels. \
**NOTE:** The auto-labeling mechanism will not remove labels once they're
created. For example, If you end up changing PR title from `sec:` to `fix:`
you'll have to manually remove `security` label.

On each commit to `main` branch, Release Drafter will update the next release
draft. Once you're ready to create new version, simply publish the draft.

In addition to Release Drafter, we're also running a script which extracts
explicitly listed release notes and breaking changes which are optionally
defined in `## Release Notes` and `## Breaking Changes` headers.

## Dependencies

Renovate is configured to automatically merge minor and patch updates.
For major versions, which sadly includes GitHub Actions, manual approval
is required.
