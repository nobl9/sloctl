#!/usr/bin/env bash
# bats file_tags=e2e

# setup_file is run only once for the whole file.
setup_file() {
  load "test_helper/load"
  load_lib "bats-assert"

  generate_inputs "$BATS_FILE_TMPDIR"
  run_sloctl apply -f "'$TEST_INPUTS/**'"
  assert_success_joined_output

  export TEST_OUTPUTS="$TEST_SUITE_OUTPUTS/move"
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

@test "move single slo (create Project and Service)" {
  run_sloctl move slo -p move-single-slo-new splunk-raw-rolling
  assert_success_joined_output
  assert_output - <<EOF
Applying 3 objects from the following sources:
 - $input
The resources were successfully applied.
EOF

  assert_applied "$(read_files "${TEST_OUTPUTS}/move-single-slo.yaml")"
}
