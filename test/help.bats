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

@test "sloctl distinguishes headings, flag names, and comments in both themes" {
  local background accent comment
  for background in light dark; do
    if [[ "$background" == light ]]; then
      accent='0;129;158'
      comment='103;104;104'
    else
      accent='0;186;211'
      comment='186;187;187'
    fi

    run --separate-stderr python3 "$TEST_INPUTS/run_help.py" \
      --width 100 --expect styled --background "$background" --raw -- sloctl get alerts --help
    assert_success
    assert_stderr ""
    # Match semantic colors without fixing unrelated layout and syntax tokens.
    assert_output --regexp $'\e''\[[0-9;]*38;2;'"$accent"'[0-9;]*mFlags'
    assert_output --regexp $'\e''\[1m--alert-policy stringArray'$'\e''\[m'
    assert_output --regexp $'\e''\[[0-9;]*38;2;'"$comment"'m# Get active and resolved alerts from all projects\.'
  done
}

@test "sloctl renders help when the terminal cannot report its background" {
  local background
  for background in unknown unresponsive; do
    run --separate-stderr python3 "$TEST_INPUTS/run_help.py" \
      --width 80 --expect styled --background "$background" -- sloctl mcp --help
    assert_success
    assert_output - < "$TEST_OUTPUTS/mcp.stdout"
    assert_stderr ""
  done
}

@test "sloctl supports read-only input and write-only help terminals" {
  run --separate-stderr python3 "$TEST_INPUTS/run_help.py" \
    --width 80 --expect styled --background light --write-only --raw -- sloctl mcp --help
  assert_success
  assert_output --regexp $'\e''\[[0-9;]*38;2;0;129;158[0-9;]*mUsage'
  assert_stderr ""

  run --separate-stderr python3 "$TEST_INPUTS/run_help.py" \
    --stream stderr --width 80 --expect styled --background light --write-only --raw -- sloctl aws-iam-ids direct
  assert_failure 1
  assert_output ""
  assert_stderr --regexp $'\e''\[[0-9;]*38;2;0;129;158[0-9;]*mUsage'
}

@test "sloctl does not query the terminal or read redirected input for help" {
  run --separate-stderr python3 "$TEST_INPUTS/run_help.py" \
    --width 80 --expect styled --redirect-stdin --write-only -- sloctl mcp --help
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

@test "sloctl preserves heredoc delimiters in help examples" {
  local action example
  for action in delete update; do
    run_help_tty 60 styled budgetadjustments events "$action" --help
    assert_success
    assert_line 'Examples'
    assert_stderr ""

    example="${output#*$'Examples\n\n'}"
    example="${example%%$'\nFlags\n'*}"
    printf '%s\n' "$example" > "$BATS_TEST_TMPDIR/example.sh"

    run --separate-stderr bash -n "$BATS_TEST_TMPDIR/example.sh"
    assert_success
    assert_output ""
    assert_stderr ""
  done
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
