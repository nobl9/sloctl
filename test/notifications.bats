#!/usr/bin/env bash
# bats file_tags=unit

setup_file() {
  export TEST_INPUTS="$BATS_TEST_DIRNAME/inputs/notifications"
  export TEST_OUTPUTS="$BATS_TEST_DIRNAME/outputs/notifications"
}

setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"

  ensure_installed python3

  unset CI
  unset ALL_PROXY
  unset HTTPS_PROXY
  unset HTTP_PROXY
  unset NO_PROXY
  unset SSL_CERT_FILE
  unset all_proxy
  unset https_proxy
  unset http_proxy
  unset no_proxy
  unset SLOCTL_NO_NOTIFICATIONS
  unset SLOCTL_NOTIFICATIONS_RELEASE_URL
  unset SLOCTL_TEST_TTY_COLUMNS
  unset SLOCTL_TEST_TTY_INPUT
  unset RELEASE_SERVER_BODY
  unset RELEASE_SERVER_BODY_FILE
  unset RELEASE_SERVER_HTML_URL
  unset RELEASE_SERVER_RAW_RESPONSE
  unset RELEASE_SERVER_STATUS
  unset RELEASE_SERVER_TAG

  export NO_COLOR=1
  export SLOCTL_ACCESSIBLE_MODE=1
  export XDG_CACHE_HOME="$BATS_TMPDIR/cache-$BATS_TEST_NUMBER"
  export RELEASE_SERVER_LOG="$BATS_TMPDIR/release-server-$BATS_TEST_NUMBER.log"
  select_update_action skip
}

teardown() {
  if [ -n "${RELEASE_SERVER_PID:-}" ]; then
    kill "$RELEASE_SERVER_PID"
    wait "$RELEASE_SERVER_PID" 2> /dev/null || true
  fi
}

@test "sloctl shows a feature notification on TTY stderr and caches it" {
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr feature-prompt-skip
  assert_release_requests 1

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  refute_stderr
  assert_release_requests 1
}

@test "sloctl prompts before command validation and then runs the command after skip" {
  start_release_server

  run_sloctl_with_tty_stderr config rename-context old
  assert_failure
  assert_notification_stderr failed-command-after-skip
  assert_release_requests 1
}

@test "sloctl skips the notification until the next version" {
  select_update_action skip-until-next-version
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr feature-prompt-skip-until-next-version
  assert_release_requests 1

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  refute_stderr
  assert_release_requests 1
}

@test "sloctl runs upgrade and exits without running the command" {
  use_release_body maintenance
  select_update_action run-upgrade
  export SLOCTL_TEST_UPGRADE_MARKER="$BATS_TEST_TMPDIR/upgrade-ran"
  local tools_dir="$BATS_TEST_TMPDIR/tools"
  mkdir -p "$tools_dir"
  printf '%s\n' \
    '#!/usr/bin/env bash' \
    'printf "%s\n" "touch \"$SLOCTL_TEST_UPGRADE_MARKER\""' \
    > "$tools_dir/curl"
  chmod +x "$tools_dir/curl"
  start_release_server

  run_sloctl_binary_with_prefixed_path /usr/bin/sloctl "$tools_dir" version
  assert_success_joined_output
  assert_output ""
  assert_notification_stderr version-prompt-run-upgrade
  assert [ -f "$SLOCTL_TEST_UPGRADE_MARKER" ]
  assert_release_requests 1
}

@test "sloctl does not show feature notification when opted out" {
  start_release_server
  export SLOCTL_NO_NOTIFICATIONS=1

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  refute_stderr
  assert_release_requests 0
}

@test "sloctl does not show feature notification in CI" {
  start_release_server
  export CI=true

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  refute_stderr
  assert_release_requests 0
}

@test "sloctl does not show feature notification without TTY stderr" {
  start_release_server

  run_sloctl version
  assert_success_joined_output
  refute_stderr
  assert_release_requests 0
}

@test "sloctl shows version notification when release has no feature notes" {
  use_release_body maintenance
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr version-prompt-skip
  assert_release_requests 1
}

@test "sloctl uses the first non-empty release notes section" {
  use_release_body empty-features-then-bug-fixes
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr bug-fix-prompt-skip
  assert_release_requests 1
}

@test "sloctl keeps nested release-note details from the selected section" {
  use_release_body features-with-details
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr features-with-details-prompt-skip
  assert_release_requests 1
}

@test "sloctl shows release note without author metadata" {
  use_release_body feature-without-author
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr feature-without-author-prompt-skip
  assert_release_requests 1
}

@test "sloctl does not show notification for current release" {
  export RELEASE_SERVER_TAG=v1.0.0
  export RELEASE_SERVER_HTML_URL=https://github.com/nobl9/sloctl/releases/tag/v1.0.0
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  refute_stderr
  assert_release_requests 1
}

@test "sloctl suppresses fetch failures and caches the check" {
  export RELEASE_SERVER_STATUS=403
  export RELEASE_SERVER_RAW_RESPONSE="rate limited"
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  refute_stderr

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  refute_stderr
  assert_release_requests 1
}

@test "sloctl suppresses malformed release responses" {
  export RELEASE_SERVER_RAW_RESPONSE="{"
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  refute_stderr
  assert_release_requests 1
}

@test "sloctl still shows notification when cache cannot be written" {
  export XDG_CACHE_HOME="$BATS_TEST_TMPDIR/cache-file"
  touch "$XDG_CACHE_HOME"
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr feature-prompt-skip
  assert_release_requests 1
}

@test "sloctl keeps install command on one line when terminal is wide" {
  use_release_body maintenance
  export SLOCTL_TEST_TTY_COLUMNS=140
  local tools_dir="$BATS_TEST_TMPDIR/tools"
  mkdir -p "$tools_dir"
  touch "$tools_dir/curl"
  chmod +x "$tools_dir/curl"
  start_release_server

  run_sloctl_binary_with_path /usr/bin/sloctl "$tools_dir" version
  assert_success_joined_output
  assert_notification_stderr install-curl-wide-prompt
}

@test "sloctl suggests Homebrew upgrade for Homebrew installs" {
  use_release_body maintenance
  local cellar_binary="$BATS_TEST_TMPDIR/opt/homebrew/Cellar/sloctl/1.2.0/bin/sloctl"
  local linked_binary="$BATS_TEST_TMPDIR/opt/homebrew/bin/sloctl"
  copy_sloctl_binary "$cellar_binary"
  mkdir -p "$(dirname "$linked_binary")"
  ln -s "$cellar_binary" "$linked_binary"
  start_release_server

  run_sloctl_binary_with_tty_stderr "$linked_binary" version
  assert_success_joined_output
  assert_notification_stderr install-homebrew-prompt
}

@test "sloctl suggests go install for Go bin installs" {
  use_release_body maintenance
  export HOME="$BATS_TEST_TMPDIR/home"
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_notification_stderr install-go-prompt
}

@test "sloctl falls back to wget when curl is unavailable" {
  use_release_body maintenance
  local tools_dir="$BATS_TEST_TMPDIR/tools"
  mkdir -p "$tools_dir"
  touch "$tools_dir/wget"
  chmod +x "$tools_dir/wget"
  start_release_server

  run_sloctl_binary_with_path /usr/bin/sloctl "$tools_dir" version
  assert_success_joined_output
  assert_notification_stderr install-wget-prompt
}

@test "sloctl omits update command when no downloader is available" {
  use_release_body maintenance
  select_update_action_without_update skip
  local tools_dir="$BATS_TEST_TMPDIR/tools"
  mkdir -p "$tools_dir"
  start_release_server

  run_sloctl_binary_with_path /usr/bin/sloctl "$tools_dir" version
  assert_success_joined_output
  assert_notification_stderr no-install-command-prompt
}

assert_notification_stderr() {
  local name="$1"
  local expected
  expected="$(normalize_tty_output < "$TEST_OUTPUTS/$name.stderr")"
  output="$(normalize_tty_output <<< "$stderr")"
  assert_output "$expected"
}

normalize_tty_output() {
  sed -e 's/\r//g' -e 's/[[:blank:]]$//'
}

use_release_body() {
  local name="$1"
  export RELEASE_SERVER_BODY_FILE="$TEST_INPUTS/release-bodies/$name.md"
}

select_update_action() {
  case "$1" in
    run-upgrade)
      export SLOCTL_TEST_TTY_INPUT=$'1\n'
      ;;
    skip)
      export SLOCTL_TEST_TTY_INPUT=$'2\n'
      ;;
    skip-until-next-version)
      export SLOCTL_TEST_TTY_INPUT=$'3\n'
      ;;
    *)
      fail "unknown update action: $1"
      ;;
  esac
}

select_update_action_without_update() {
  case "$1" in
    skip)
      export SLOCTL_TEST_TTY_INPUT=$'1\n'
      ;;
    skip-until-next-version)
      export SLOCTL_TEST_TTY_INPUT=$'2\n'
      ;;
    *)
      fail "unknown update action without update command: $1"
      ;;
  esac
}

run_sloctl_with_tty_stderr() {
  run_sloctl_binary_with_tty_stderr sloctl "$@"
}

run_sloctl_binary_with_tty_stderr() {
  local binary="$1"
  shift
  bats_require_minimum_version 1.5.0
  run --separate-stderr python3 "$TEST_INPUTS/run_with_stderr_pty.py" "$binary" "$@"
}

run_sloctl_binary_with_path() {
  local binary="$1"
  local path="$2"
  shift 2
  bats_require_minimum_version 1.5.0
  run --separate-stderr env PATH="$path" /usr/bin/python3 "$TEST_INPUTS/run_with_stderr_pty.py" "$binary" "$@"
}

run_sloctl_binary_with_prefixed_path() {
  local binary="$1"
  local path="$2"
  shift 2
  bats_require_minimum_version 1.5.0
  run --separate-stderr env PATH="$path:$PATH" /usr/bin/python3 "$TEST_INPUTS/run_with_stderr_pty.py" "$binary" "$@"
}

copy_sloctl_binary() {
  local target="$1"
  mkdir -p "$(dirname "$target")"
  cp /usr/bin/sloctl "$target"
  chmod +x "$target"
}

start_release_server() {
  local port_file="$BATS_TMPDIR/release-server-$BATS_TEST_NUMBER.port"
  python3 "$TEST_INPUTS/release_server.py" "$port_file" &
  RELEASE_SERVER_PID="$!"

  for _ in {1..50}; do
    if [ -s "$port_file" ]; then
      local port
      port="$(cat "$port_file")"
      export SLOCTL_NOTIFICATIONS_RELEASE_URL="http://127.0.0.1:$port/repos/nobl9/sloctl/releases/latest"
      return 0
    fi
    sleep 0.1
  done

  fail "release server did not start"
}

assert_release_requests() {
  local expected="$1"
  local actual=0
  if [ -f "$RELEASE_SERVER_LOG" ]; then
    actual="$(wc -l < "$RELEASE_SERVER_LOG" | tr -d " ")"
  fi
  assert_equal "$actual" "$expected"
}
