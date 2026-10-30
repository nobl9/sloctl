#!/usr/bin/env bash
# bats file_tags=unit

bats_require_minimum_version 1.5.0

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

@test "sloctl config form keeps stderr and cancellation during background detection" {
  ensure_installed python3 cmp
  cp "$SLOCTL_DEFAULT_CONFIG" "$BATS_TEST_TMPDIR/before.toml"
  export SLOCTL_NO_NOTIFICATIONS=1
  export SLOCTL_ACCESSIBLE_MODE=0
  export SLOCTL_TEST_TTY_INPUT_WHEN_RAW=1
  export SLOCTL_TEST_TTY_INPUT_AFTER_QUERY=1
  export SLOCTL_TEST_TTY_INPUT=$'\x03'
  export SLOCTL_TEST_TTY_BACKGROUND=unresponsive
  export TERM=xterm-256color COLORTERM=truecolor
  unset NO_COLOR CLICOLOR CLICOLOR_FORCE

  run --separate-stderr python3 "$TEST_SUITE_INPUTS/notifications/run_with_stderr_pty.py" sloctl config add-context
  assert_failure 1
  assert_output ""
  assert_stderr --partial "failed to run context addition form"

  run cmp "$BATS_TEST_TMPDIR/before.toml" "$SLOCTL_DEFAULT_CONFIG"
  assert_success
}

@test "sloctl config form keyboard hints follow the terminal background" {
  ensure_installed python3
  export SLOCTL_NO_NOTIFICATIONS=1
  export SLOCTL_ACCESSIBLE_MODE=0
  export SLOCTL_TEST_TTY_INPUT_WHEN_RAW=1
  export SLOCTL_TEST_TTY_INPUT=$'\x03'
  export SLOCTL_TEST_TTY_PROMPT_TITLE='Provide context name'
  export TERM=xterm-256color COLORTERM=truecolor
  unset NO_COLOR CLICOLOR CLICOLOR_FORCE

  local background accent muted
  for background in light dark; do
    if [[ "$background" == light ]]; then
      accent='0;129;158'
      muted='103;104;104'
    else
      accent='0;186;211'
      muted='186;187;187'
    fi
    SLOCTL_TEST_TTY_BACKGROUND="$background" \
      run --separate-stderr python3 "$TEST_SUITE_INPUTS/notifications/run_with_stderr_pty.py" sloctl config add-context
    assert_failure 1
    assert_output ""
    assert_stderr --regexp $'\e''\[[0-9;]*38;2;'"$accent"'[0-9;]*menter'
    assert_stderr --regexp $'\e''\[[0-9;]*38;2;'"$muted"'[0-9;]*mnext'
  done
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

@test "sloctl --context completion lists config contexts" {
  run --separate-stderr sloctl __complete --config "$SLOCTL_DEFAULT_CONFIG" --context ""
  assert_success
  assert_output 'full
minimal
:4'
}

@test "sloctl --context completion filters config contexts" {
  run --separate-stderr sloctl __complete --config "$SLOCTL_DEFAULT_CONFIG" --context m
  assert_success
  assert_output 'minimal
:4'
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

@test "sloctl config current-context, show-secret flag without verbose" {
  run_sloctl config current-context --show-secret
  assert_failure
  assert_stderr 'Error: --show-secret flag can only be set if --verbose flag is also provided'
}

@test "sloctl config current-context (verbose with show-secret)" {
  run_sloctl config use-context full
  assert_success

  run_sloctl config current-context -v --show-secret -o yaml
  assert_success_joined_output
  assert_output <"$TEST_OUTPUTS/get-current-context-full-show-secret.yaml"
}
