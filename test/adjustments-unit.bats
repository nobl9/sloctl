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
  assert_stderr 'Error: required flag(s) "adjustment-name", "from", "to" not set'

  run_sloctl budgetadjustments events delete
  assert_failure
  assert_stderr 'Error: required flag(s) "adjustment-name", "file" not set'

  run_sloctl budgetadjustments events update
  assert_failure
  assert_stderr 'Error: required flag(s) "adjustment-name", "file" not set'
}

@test "invalid date format" {
  run_sloctl budgetadjustments events get --adjustment-name=foo --from=xyz --to=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  assert_failure
  assert_stderr "Error: invalid argument \"xyz\" for \"--from\" flag: invalid time format, expected RFC3339 layout (e.g. '2006-01-02T15:04:05Z' or '2006-01-02T08:04:05-07:00')"

  run_sloctl budgetadjustments events get --adjustment-name=foo --to=xyz --from=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  assert_failure
  assert_stderr "Error: invalid argument \"xyz\" for \"--to\" flag: invalid time format, expected RFC3339 layout (e.g. '2006-01-02T15:04:05Z' or '2006-01-02T08:04:05-07:00')"
}

@test "conditionally required arguments (slo-project and slo-name)" {
  run_sloctl budgetadjustments events get --adjustment-name=foo --from=2024-01-01T00:00:00Z --to=$(date -u +%Y-%m-%dT%H:%M:%SZ) --slo-project=baz
  assert_failure
  assert_stderr 'Error: required flag(s) "slo-name" not set'

  run_sloctl budgetadjustments events get --adjustment-name=foo --from=2024-01-01T00:00:00Z --to=$(date -u +%Y-%m-%dT%H:%M:%SZ) --slo-name=bar
  assert_failure
  assert_stderr 'Error: required flag(s) "slo-project" not set'
}

@test "check adjustment filtered by slo project and name flags - validation errors" {
  run_sloctl get budgetadjustments --project prometheus
  assert_failure
  output="$stderr"
  assert_output "Error: if any flags in the group [slo project] are set they must all be set; missing [slo]"

  run_sloctl get budgetadjustments --slo slo-2025-06-01-003
  assert_failure
  output="$stderr"
  assert_output "Error: if any flags in the group [slo project] are set they must all be set; missing [project]"
}
