#!/usr/bin/env bash
# bats file_tags=unit

# setup_file is run only once for the whole file.
setup_file() {
  export SLOCTL_CLIENT_ID=id
  export SLOCTL_CLIENT_SECRET=secret
}

# setup is run before each test.
setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"
}

@test "missing required flags" {
  run_sloctl move slo splunk-raw-rolling

  assert_failure
  assert_stderr 'Error: Either --to-project (for cross-project move) or --to-service (for same-project service move) must be provided.'
}

@test "validation error" {
  run_sloctl move slo splunk-raw-rolling --to-project=NewProject

  assert_failure
  assert_stderr - <<EOF
Error: Validation for Move SLOs request has failed for the following properties:
  - 'newProject' with value 'NewProject':
    - string must match regular expression: '^[a-z0-9]([a-z0-9-]*[a-z0-9])?$'; must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character
EOF
}
