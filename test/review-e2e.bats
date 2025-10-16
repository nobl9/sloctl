#!/usr/bin/env bash
# bats file_tags=e2e

# setup_file is run only once for the whole file.
setup_file() {
  export SLOCTL_CLIENT_ID="$SLOCTL_E2E_CLIENT_ID"
  export SLOCTL_CLIENT_SECRET="$SLOCTL_E2E_CLIENT_SECRET"

  generate_inputs "$BATS_TMPDIR"
  generate_outputs
}

# setup is run before each test.
setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"
}

# teardown_file is run only once for the whole file.
teardown_file() {
  # Clean up any created SLOs
  run_sloctl delete slo test-slo-for-review -p "$TEST_PROJECT" --ignore-not-found
}

@test "review set with all valid statuses" {
  # First apply the test SLO
  run_sloctl apply -f "$TEST_INPUTS/test-slo.yaml"
  assert_success

  # Test each valid status
  statuses=("reviewed" "skipped" "pending" "overdue" "notStarted")

  for status in "${statuses[@]}"; do
    run_sloctl review set test-slo-for-review --status "$status" -p "$TEST_PROJECT"
    assert_success
    assert_output --partial "Successfully set review status to '$status' for SLO 'test-slo-for-review' in project '$TEST_PROJECT'"
  done
}

@test "review set with note for reviewed status" {
  # Apply the test SLO if not already present
  run_sloctl apply -f "$TEST_INPUTS/test-slo.yaml"
  assert_success

  run_sloctl review set test-slo-for-review --status reviewed -p "$TEST_PROJECT" --note "Test review note"
  assert_success
  assert_output --partial "Successfully set review status to 'reviewed' for SLO 'test-slo-for-review' in project '$TEST_PROJECT'"
}

@test "review set with note for skipped status" {
  # Apply the test SLO if not already present
  run_sloctl apply -f "$TEST_INPUTS/test-slo.yaml"
  assert_success

  run_sloctl review set test-slo-for-review --status skipped -p "$TEST_PROJECT" --note "Skipped due to insufficient data"
  assert_success
  assert_output --partial "Successfully set review status to 'skipped' for SLO 'test-slo-for-review' in project '$TEST_PROJECT'"
}