#!/usr/bin/env bash
# bats file_tags=unit

# setup_file is run only once for the whole file.
setup_file() {
  CONFIG_FILENAME="config.toml"
  export SLOCTL_DEFAULT_CONFIG="$BATS_TMPDIR/$CONFIG_FILENAME"
  export TEST_INPUTS="$TEST_SUITE_INPUTS/config"
  cp "$TEST_INPUTS/$CONFIG_FILENAME" "$SLOCTL_DEFAULT_CONFIG"
  export TEST_OUTPUTS="$TEST_SUITE_OUTPUTS/config"
}

# setup is run before each test.
setup() {
  load "test_helper/load"
  load_lib "bats-assert"
  load_lib "bats-support"

  # Ensure env vars are not taken into consideration.
  export SLOCTL_DEFAULT_CONTEXT="fake"
  export SLOCTL_NO_CONFIG_FILE="true"
  # Use minimal TUI.
  export SLOCTL_ACCESSIBLE_MODE=1
  export NO_COLOR=1
  # Always reset the default config.
  export SLOCTL_CONFIG_FILE_PATH="$SLOCTL_DEFAULT_CONFIG"
}

# teardown is run after each test.
teardown() {
  run_sloctl config use-context minimal
  assert_success_joined_output
}

@test "sloctl config current-context" {
  run_sloctl config current-context
  assert_success_joined_output
  assert_output 'minimal'
}

@test "sloctl config current-context (override config path with flag)" {
  run_sloctl config --config="$TEST_INPUTS/delete-config.toml" current-context
  assert_success_joined_output
  assert_output '3'
}

@test "sloctl config current-context (verbose, minimal, default YAML)" {
  run_sloctl config current-context -v
  assert_success_joined_output
  assert_output <"$TEST_OUTPUTS/get-current-context-minimal.yaml"
}

@test "sloctl config current-context (verbose, minimal)" {
  for format in yaml json toml csv; do
    run_sloctl config current-context -v -o "$format"
    assert_success_joined_output
    assert_output <"$TEST_OUTPUTS/get-current-context-minimal.$format"
  done
}

@test "sloctl config current-context (verbose, full)" {
  run_sloctl config use-context full
  assert_success

  for format in yaml json toml csv; do
    run_sloctl config current-context -v -o "$format"
    assert_success_joined_output
    assert_output <"$TEST_OUTPUTS/get-current-context-full.$format"
  done
}

@test "sloctl config current-context, output flag without verbose" {
  run_sloctl config current-context -o json
  assert_failure
  assert_stderr 'Error: --output flag can only be set if --verbose flag is also provided'
}

@test "sloctl config use-context" {
  run_sloctl config use-context full
  assert_success_joined_output
  assert_output 'Switched to context "full".'

  run_sloctl config current-context
  assert_success_joined_output
  assert_output 'full'
}

@test "sloctl config use-context (interactive)" {
  run_sloctl config use-context <<<"2"
  assert_success
  assert_output --partial 'Switched to context "minimal".'

  run_sloctl config current-context
  assert_success_joined_output
  assert_output 'minimal'
}

@test "sloctl config get-contexts" {
  run_sloctl config get-contexts
  assert_success_joined_output
  assert_output 'full
minimal'
}

@test "sloctl config get-contexts (verbose, default YAML)" {
  run_sloctl config get-contexts -v
  assert_success_joined_output
  assert_output <"$TEST_OUTPUTS/get-contexts-verbose.yaml"
}

@test "sloctl config get-contexts (verbose)" {
  for format in yaml json toml csv; do
    run_sloctl config get-contexts -v -o "$format"
    assert_success_joined_output
    assert_output <"$TEST_OUTPUTS/get-contexts-verbose.$format"
  done
}

@test "sloctl config get-contexts, output flag without verbose" {
  run_sloctl config get-contexts -o json
  assert_failure
  assert_stderr 'Error: --output flag can only be set if --verbose flag is also provided'
}

@test "sloctl config rename-context" {
  run_sloctl config rename-context minimal mini
  assert_success_joined_output
  assert_output 'Renamed context was set as default. Changing default context to "mini".
Renamed context "minimal" to "mini".'

  run_sloctl config get-contexts
  assert_success_joined_output
  assert_output 'full
mini'

  run_sloctl config rename-context mini minimal
  assert_success_joined_output
  assert_output --partial 'Renamed context "mini" to "minimal".'
}

@test "sloctl config rename-context (interactive)" {
  run bash -c '
  set -eo pipefail
  (
    echo "1"
    sleep 0.1
    echo "fullish"
  ) |
    sloctl --config "$SLOCTL_CONFIG" config rename-context
  '
  assert_success
  assert_output --partial 'Renamed context "full" to "fullish".'

  run_sloctl config get-contexts
  assert_success_joined_output
  assert_output 'fullish
minimal'

  run_sloctl config rename-context fullish full
  assert_success_joined_output
  assert_output 'Renamed context "fullish" to "full".'
}

@test "sloctl config rename-context, no contexts" {
  run_sloctl config --config="$TEST_INPUTS/empty-config.toml" rename-context
  assert_failure
  assert_stderr 'Error: there are no contexts defined in your configuration file'
}

@test "sloctl config rename-context, invalid args" {
  run_sloctl config rename-context mini
  assert_failure
  assert_stderr 'Error: either provide new and old context names or no arguments at all, received 1 arguments'
}

@test "sloctl config rename-context, new context is empty" {
  run_sloctl config rename-context minimal "' '"
  assert_failure
  assert_stderr 'Error: new context cannot be empty'
}

@test "sloctl config rename-context, old context doesn't exist" {
  run_sloctl config rename-context mini minimal
  assert_failure
  assert_stderr 'Error: selected context "mini" does not exists'
}

@test "sloctl config rename-context, new context already exists" {
  run_sloctl config rename-context minimal full
  assert_failure
  assert_stderr 'Error: selected context name "full" is already in use'
}

@test "sloctl config delete-context, non-existing context" {
  run_sloctl config delete-context fake
  assert_failure
  assert_stderr 'Error: selected context "fake" does not exists'
}

@test "sloctl config delete-context, cannot delete default context" {
  run_sloctl config delete-context minimal
  assert_failure
  assert_stderr 'Error: cannot remove context currently set as default'
}

@test "sloctl config delete-context, no contexts" {
  run_sloctl config --config="$TEST_INPUTS/empty-config.toml" delete-context
  assert_failure
  assert_stderr 'Error: there are no contexts defined in your configuration file'
}

@test "sloctl config delete-context, single context set as default" {
  run_sloctl config --config="$TEST_INPUTS/single-context-config.toml" delete-context
  assert_failure
  assert_stderr 'Error: cannot remove context currently set as default; there'"'"'s only a single context set in your configuration file and it is marked as default'
}

@test "sloctl config delete-context" {
  SLOCTL_CONFIG="$TEST_INPUTS/delete-config.toml"

  run_sloctl config --config="$SLOCTL_CONFIG" delete-context "1"
  assert_success_joined_output
  assert_output 'Context "1" has been deleted.'

  run_sloctl config --config="$SLOCTL_CONFIG" get-contexts
  assert_success_joined_output
  assert_output '2
3'
}

@test "sloctl config delete-context (interactive)" {
  SLOCTL_CONFIG="$TEST_INPUTS/delete-config.toml"

  run_sloctl config --config="$SLOCTL_CONFIG" delete-context <<<"1"
  assert_success_joined_output
  assert_output --partial 'Context "2" has been deleted.'

  run_sloctl config --config="$SLOCTL_CONFIG" get-contexts
  assert_success_joined_output
  assert_output '3'
}
