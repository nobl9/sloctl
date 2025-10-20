#!/usr/bin/env bash
# bats file_tags=e2e

# setup_file is run only once for the whole file.
setup_file() {
  load "test_helper/load"
  load_lib "bats-assert"

  generate_inputs "$BATS_FILE_TMPDIR"

  run_sloctl apply -f "'$TEST_INPUTS/**'"
  assert_success_joined_output
}

# teardown_file is run only once for the whole file.
teardown_file() {
  run_sloctl delete -f "'$TEST_INPUTS/**'"
}

# setup is run before each test.
setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"
}

@test "review set-status with cycle" {
  run_sloctl get slo slo-with-cycle -p "$TEST_PROJECT" -o json
  assert_success_joined_output
  assert_slo_review_status "toReview"

  # Test reviewed status
  run_sloctl review set-status reviewed slo-with-cycle -p "$TEST_PROJECT"
  assert_success_joined_output
  # Verify the status was set
  run_sloctl get slo slo-with-cycle -p "$TEST_PROJECT" -o json
  assert_success_joined_output
  assert_slo_review_status "reviewed"

  # Test skipped status
  run_sloctl review set-status skipped slo-with-cycle -p "$TEST_PROJECT"
  assert_success_joined_output
  # Verify the status was set
  run_sloctl get slo slo-with-cycle -p "$TEST_PROJECT" -o json
  assert_success_joined_output
  assert_slo_review_status "skipped"

  # Test toReview status
  run_sloctl review set-status to-review slo-with-cycle -p "$TEST_PROJECT"
  assert_success_joined_output
  # Verify the status was set
  run_sloctl get slo slo-with-cycle -p "$TEST_PROJECT" -o json
  assert_success_joined_output
  assert_slo_review_status "toReview"
}

@test "review set-status without cycle" {
  run_sloctl get slo slo-without-review-cycle -p "$TEST_PROJECT" -o json
  assert_success_joined_output
  assert_slo_review_status "notStarted"

  # Test reviewed status
  run_sloctl review set-status reviewed slo-without-review-cycle -p "$TEST_PROJECT"
  assert_success_joined_output
  # Verify the status was set
  run_sloctl get slo slo-without-review-cycle -p "$TEST_PROJECT" -o json
  assert_success_joined_output
  assert_slo_review_status "reviewed"

  # Test notStarted status
  run_sloctl review set-status not-started slo-without-review-cycle -p "$TEST_PROJECT"
  assert_success_joined_output
  # Verify the status was set
  run_sloctl get slo slo-without-review-cycle -p "$TEST_PROJECT" -o json
  assert_success_joined_output
  assert_slo_review_status "notStarted"
}

assert_slo_review_status() {
  local expected_status="$1"
  local actual_status

  actual_status=$(echo "$output" | yq -r '.[0].status.review.status')

  if [[ "$actual_status" != "$expected_status" ]]; then
    fail "Expected review status '$expected_status' but got '$actual_status'"
  fi
}