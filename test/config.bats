#!/usr/bin/env bash
# bats file_tags=unit

# setup_file is run only once for the whole file.
setup_file() {
	CONFIG_FILENAME="config.toml"
	export SLOCTL_CONFIG="$BATS_TMPDIR/$CONFIG_FILENAME"
	cp "$TEST_SUITE_INPUTS/config/$CONFIG_FILENAME" "$SLOCTL_CONFIG"
	export OUTPUTS="$TEST_SUITE_OUTPUTS/config"
}

# setup is run before each test.
setup() {
	load "test_helper/load"
	load_lib "bats-assert"
	load_lib "bats-support"
}

run_sloctl_config() {
  # Ensure env vars are not taken into consideration.
  export SLOCTL_DEFAULT_CONTEXT="fake"
  export SLOCTL_NO_CONFIG_FILE="true"
  # Run sloctl.
	run_sloctl --config "$SLOCTL_CONFIG" "$@"
}

@test "sloctl config current-context" {
	run_sloctl_config config current-context

  assert_success_joined_output
	assert_output 'minimal'
}

@test "sloctl config use-context" {
	run_sloctl_config config current-context

  assert_success_joined_output
	assert_output 'minimal'

	run_sloctl_config config use-context full

  assert_success_joined_output
	assert_output 'Switched to context "full"'

	run_sloctl_config config current-context

  assert_success_joined_output
	assert_output 'full'
}

@test "sloctl config get-contexts" {
	run_sloctl_config config get-contexts

  assert_success_joined_output
	assert_output '[full, minimal]'
}

@test "sloctl config get-contexts verbose" {
	run_sloctl_config config get-contexts -v

  assert_success_joined_output
	assert_output <"$OUTPUTS/get-contexts-verbose.txt"
}

@test "sloctl config rename-context" {
	run_sloctl_config config rename-context minimal mini

  assert_success_joined_output
	assert_output 'Renaming: "minimal" to "mini"'

	run_sloctl_config config get-contexts

  assert_success_joined_output
	assert_output '[full, mini]'
}

@test "sloctl config delete-context" {
	run_sloctl_config config delete-context mini

  assert_success_joined_output
	assert_output 'Context "mini" has been deleted.'
}
