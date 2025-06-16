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
