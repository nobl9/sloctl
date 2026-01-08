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

@test "date 'to' is before 'from'" {
  from="2024-01-02T00:00:00Z"
  run_sloctl budgetadjustments events get --adjustment-name=foo --from="$from" --to=2024-01-01T00:00:00Z
  assert_failure
  assert_stderr "Error: - 'to' date must be be after 'from' date (source: 'to', value: '{\"Adjustment\":\"foo\",\"From\":\"${from}\",\"To\":\"2024-01-01T00:00:00Z\",\"SloProject\":\"\",\"SloNa...')"
}

@test "get events for non-existent adjustment" {
  run_sloctl budgetadjustments events get --adjustment-name=foo --from=2024-01-01T00:00:00Z --to=2024-01-02T00:00:00Z
  assert_failure
  assert_stderr "Error: - adjustment 'foo' was not found"

  for action in delete update; do
    run_sloctl budgetadjustments events $action  --adjustment-name=foo -f "'$TEST_INPUTS/sample-events.yaml'"
    assert_failure
    assert_stderr "Error: - adjustment 'foo' was not found"
  done
}
