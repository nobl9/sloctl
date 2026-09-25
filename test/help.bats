#!/usr/bin/env bats
# bats file_tags=unit

setup_file() {
  export TEST_INPUTS="$BATS_TEST_DIRNAME/inputs/help"
  export TEST_OUTPUTS="$BATS_TEST_DIRNAME/outputs/help"
}

setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"
  bats_require_minimum_version 1.5.0
  ensure_installed python3

  export SLOCTL_NO_NOTIFICATIONS=1
  export SLOCTL_CONFIG_FILE_PATH="$BATS_TEST_TMPDIR/missing-config.toml"
  export TERM=xterm-256color
  export COLORTERM=truecolor
  unset NO_COLOR CLICOLOR CLICOLOR_FORCE
}

@test "sloctl renders root help for every help entrypoint" {
  local argument
  for argument in --help -h help; do
    run_help_tty 80 styled "$argument"
    assert_success
    assert_output - < "$TEST_OUTPUTS/root.stdout"
    assert_stderr ""
  done

  run_help_tty 80 styled
  assert_success
  assert_output - < "$TEST_OUTPUTS/root.stdout"
  assert_stderr ""
}

@test "sloctl renders nested help with aliases and flag defaults" {
  run_help_tty 80 styled get slo --help
  assert_success
  assert_output - < "$TEST_OUTPUTS/get-slo.stdout"
  assert_stderr ""

  run_help_tty 80 styled help get slo
  assert_success
  assert_output - < "$TEST_OUTPUTS/get-slo.stdout"
  assert_stderr ""
}

@test "sloctl renders links in help" {
  run_help_tty 80 styled mcp --help
  assert_success
  assert_output - < "$TEST_OUTPUTS/mcp.stdout"
  assert_stderr ""
}

@test "sloctl wraps completion prose without changing code examples" {
  run_help_tty 60 styled completion bash --help
  assert_success
  assert_output - < "$TEST_OUTPUTS/completion-bash-60.stdout"
  assert_stderr ""
}

@test "sloctl renders usage on stderr for invalid arguments" {
  run --separate-stderr python3 "$TEST_INPUTS/run_help.py" \
    --stream stderr --width 80 --expect styled -- sloctl aws-iam-ids direct
  assert_failure 1
  assert_output ""
  assert_stderr - < "$TEST_OUTPUTS/aws-iam-ids-usage.stderr"
}

@test "sloctl keeps redirected help plain even when stderr is a terminal" {
  run --separate-stderr sloctl mcp --help
  assert_success
  assert_output - < "$TEST_OUTPUTS/mcp-plain.stdout"
  assert_stderr ""

  run --separate-stderr python3 "$TEST_INPUTS/run_help.py" \
    --stream stderr --width 80 --expect plain -- sloctl mcp --help
  assert_success
  assert_output - < "$TEST_OUTPUTS/mcp-plain.stdout"
  assert_stderr ""
}

@test "sloctl keeps terminal help plain with NO_COLOR or TERM=dumb" {
  NO_COLOR=1 run_help_tty 80 plain mcp --help
  assert_success
  assert_output - < "$TEST_OUTPUTS/mcp-plain.stdout"
  assert_stderr ""

  TERM=dumb run_help_tty 80 plain mcp --help
  assert_success
  assert_output - < "$TEST_OUTPUTS/mcp-plain.stdout"
  assert_stderr ""
}

run_help_tty() {
  local width="$1" expected_style="$2"
  shift 2
  run --separate-stderr python3 "$TEST_INPUTS/run_help.py" \
    --width "$width" --expect "$expected_style" -- sloctl "$@"
}
