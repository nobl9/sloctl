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

@test "missing --to-project flag" {
  run_sloctl move slo splunk-raw-rolling

  assert_failure
  output="$stderr"
  assert_output 'Error: required flag(s) "to-project" not set'
}

@test "validation error" {
  run_sloctl move slo splunk-raw-rolling --to-project=NewProject

  assert_failure
  output="$stderr"
  assert_output - <<EOF
Error: Validation for Move SLOs request has failed for the following properties:
  - 'newProject' with value 'NewProject':
    - string must match regular expression: '^[a-z0-9]([-a-z0-9]*[a-z0-9])?$' (e.g. 'my-name', '123-abc'); an RFC-1123 compliant label name must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character
EOF
}
