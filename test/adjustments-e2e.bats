#!/usr/bin/env bash
# bats file_tags=e2e

# setup_file is run only once for the whole file.
setup_file() {
  load "test_helper/load"
  load_lib "bats-assert"

  generate_inputs "$BATS_FILE_TMPDIR"
}

# setup is run before each test.
setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"
}

@test "date to is before from" {
  from=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  run_sloctl budgetadjustments events get --adjustment-name=foo --to=2024-01-01T00:00:00Z --from=$from
  assert_failure
  output="$stderr"
  assert_output "Error: - 'to' date must be be after 'from' date (source: 'to', value: '{\"Adjustment\":\"foo\",\"From\":\"${from}\",\"To\":\"2024-01-01T00:00:00Z\",\"SloProject\":\"\",\"SloNa...')"
}

@test "adjustment not found" {
  run_sloctl budgetadjustments events get --adjustment-name=foo --from=2024-01-01T00:00:00Z --to=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  assert_failure
  output="$stderr"
  assert_output "Error: - adjustment 'foo' was not found"

  for action in delete update; do
    run_sloctl budgetadjustments events $action  --adjustment-name=foo -f "'$TEST_INPUTS/sample-events.yaml'"
    assert_failure
    output="$stderr"
    assert_output "Error: - adjustment 'foo' was not found"
  done
}
