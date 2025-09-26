#!/usr/bin/env bash
# bats file_tags=e2e

# setup_file is run only once for the whole file.
setup_file() {
  load "test_helper/load"
  load_lib "bats-assert"
  #
  # generate_inputs "$BATS_FILE_TMPDIR"
  # run_sloctl apply -f "'$TEST_INPUTS/**'"
  # assert_success_joined_output
  #
  # export TEST_OUTPUTS="$TEST_SUITE_OUTPUTS/get"
}

# # teardown_file is run only once for the whole file.
# teardown_file() {
  # run_sloctl delete -f "'$TEST_INPUTS/**'"
# }

# setup is run before each test.
setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"
}

@test "get" {
  run_mcp_inspector --method resources/list
}
