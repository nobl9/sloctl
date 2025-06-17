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
  run_sloctl budgetadjustments events get
  assert_failure
  output="$stderr"
  assert_output 'Error: required flag(s) "adjustment-name", "from", "to" not set'

  run_sloctl budgetadjustments events delete
  assert_failure
  output="$stderr"
  assert_output 'Error: required flag(s) "adjustment-name", "file" not set'

  run_sloctl budgetadjustments events update
  assert_failure
  output="$stderr"
  assert_output 'Error: required flag(s) "adjustment-name", "file" not set'
}

@test "invalid date format" {
  run_sloctl budgetadjustments events get --adjustment-name=foo --from=xyz --to=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  assert_failure
  output="$stderr"
  assert_output "Error: invalid argument \"xyz\" for \"--from\" flag: date does not match '2006-01-02T15:04:05Z07:00' layout (RFC3339)"

  run_sloctl budgetadjustments events get --adjustment-name=foo --to=xyz --from=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  assert_failure
  output="$stderr"
  assert_output "Error: invalid argument \"xyz\" for \"--to\" flag: date does not match '2006-01-02T15:04:05Z07:00' layout (RFC3339)"
}

@test "conditionally required arguments (slo-project and slo-name)" {
  run_sloctl budgetadjustments events get --adjustment-name=foo --from=2024-01-01T00:00:00Z --to=$(date -u +%Y-%m-%dT%H:%M:%SZ) --slo-project=baz
  assert_failure
  output="$stderr"
  assert_output 'Error: required flag(s) "slo-name" not set'

  run_sloctl budgetadjustments events get --adjustment-name=foo --from=2024-01-01T00:00:00Z --to=$(date -u +%Y-%m-%dT%H:%M:%SZ) --slo-name=bar
  assert_failure
  output="$stderr"
  assert_output 'Error: required flag(s) "slo-project" not set'
}
