#!/usr/bin/env bash
# bats file_tags=e2e

# setup_file is run only once for the whole file.
setup_file() {
  export TEST_OUTPUTS="$TEST_SUITE_OUTPUTS/config"
}

# setup is run before each test.
setup() {
  load "test_helper/load"
  load_lib "bats-assert"
  load_lib "bats-support"
}

@test "sloctl config current-user id" {
  run_sloctl config current-user
  assert_success_joined_output
  assert_regex "$output" "([a-z][A-Z][0-9])+"
}

@test "sloctl config current-user (verbose, default YAML)" {
  run_sloctl config current-user --verbose
  assert_success_joined_output
  assert_regex "$output" "$(cat "$TEST_OUTPUTS/get-current-user-regex.yaml")"
}

@test "sloctl config current-user (verbose)" {
  for format in yaml json csv; do
    run_sloctl config current-user -v -o "$format"
    assert_success_joined_output
    assert_regex "$output" "$(cat "$TEST_OUTPUTS/get-current-user-regex.$format")"
  done
}
