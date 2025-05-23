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

@test "sloctl config current-user (default YAML)" {
	run_sloctl config current-user

	assert_regex "$output" "$(cat "$TEST_OUTPUTS/get-current-user-regex.yaml")"
}

@test "sloctl config current-user (YAML)" {
	run_sloctl config current-user -o yaml

	assert_regex "$output" "$(cat "$TEST_OUTPUTS/get-current-user-regex.yaml")"
}

@test "sloctl config current-user (JSON)" {
	run_sloctl config current-user -o json

	assert_regex "$output" "$(cat "$TEST_OUTPUTS/get-current-user-regex.json")"
}

@test "sloctl config current-user (CSV)" {
	run_sloctl config current-user -o csv

	assert_regex "$output" "$(cat "$TEST_OUTPUTS/get-current-user-regex.csv")"
}
