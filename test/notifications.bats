#!/usr/bin/env bash
# bats file_tags=unit

setup_file() {
  load "test_helper/load"

  ensure_installed python3

  export TEST_INPUTS="$BATS_TEST_DIRNAME/inputs/notifications"
  export TEST_OUTPUTS="$BATS_TEST_DIRNAME/outputs/notifications"
}

setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"

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
  unset SLOCTL_TEST_TTY_JOIN_OUTPUT
  unset RELEASE_SERVER_BODY_FILE
  unset RELEASE_SERVER_HTML_URL
  unset RELEASE_SERVER_RAW_RESPONSE
  unset RELEASE_SERVER_STATUS
  unset RELEASE_SERVER_TAG

  export NO_COLOR=1
  export SLOCTL_ACCESSIBLE_MODE=1
  export XDG_CACHE_HOME="$BATS_TMPDIR/cache-$BATS_TEST_NUMBER"
  export LocalAppData="$BATS_TMPDIR/cache-$BATS_TEST_NUMBER"
  export RELEASE_SERVER_LOG="$BATS_TMPDIR/release-server-$BATS_TEST_NUMBER.log"
  RELEASE_SERVER_START_COUNT=0
  select_update_action skip
}

teardown() {
  stop_release_server
}

@test "sloctl shows a feature notification on TTY stderr and caches it" {
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr feature-prompt-skip
  assert_release_requests 1

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_stderr ""
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
  assert_stderr ""
  assert_release_requests 1

  expire_notification_cache
  stop_release_server
  use_release_body feature-without-author
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_stderr ""
  assert_release_requests 2
}

@test "sloctl defaults to update action and exits without running the command" {
  use_release_body maintenance
  select_default_update_action
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
  assert_stderr ""
  assert_release_requests 0
}

@test "sloctl does not show feature notification in CI" {
  start_release_server
  export CI=true

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_stderr ""
  assert_release_requests 0
}

@test "sloctl does not show feature notification without TTY stderr" {
  start_release_server

  run_sloctl version
  assert_success_joined_output
  assert_stderr ""
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

# bats test_tags=platform,platform:unix
@test "sloctl shows the new version notification and update form on supported terminals" {
  use_release_body maintenance
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  # Exact prompt rendering is covered by unit cases; this test isolates platform form support.
  assert_stderr --partial "New sloctl version v1.1.0 is available!"
  assert_stderr --partial "Choose update action"
  assert_release_requests 1
}

# bats test_tags=platform,platform:windows
@test "sloctl in a native Windows console shows the notification without the update form" {
  if [[ "$(uname -s)" != MINGW* && "$(uname -s)" != CYGWIN* ]]; then
    skip "Windows-specific compatibility test"
  fi

  use_release_body maintenance
  local tools_dir="$BATS_TEST_TMPDIR/tools-without-uname"
  mkdir -p "$tools_dir"
  start_release_server

  run_sloctl_binary_in_windows_console_with_path "$(native_sloctl_binary)" "$tools_dir" version
  assert_success_joined_output
  # winpty combines native console streams, so assertions use its joined terminal output.
  assert_output --partial "New sloctl version v1.1.0 is available!"
  refute_output --partial "Choose update action"
  assert_release_requests 1
}

@test "sloctl skips empty release notes sections" {
  use_release_body empty-features-then-bug-fixes
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr bug-fix-prompt-skip
  assert_release_requests 1
}

@test "sloctl shows breaking change notification" {
  use_release_body breaking
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr breaking-prompt-skip
  assert_release_requests 1
}

@test "sloctl keeps nested details and additional release-note sections" {
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
  assert_stderr ""
  assert_release_requests 1
}

@test "sloctl suppresses fetch failures and caches the check" {
  export RELEASE_SERVER_STATUS=403
  export RELEASE_SERVER_RAW_RESPONSE="rate limited"
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_stderr ""

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_stderr ""
  assert_release_requests 1
}

@test "sloctl suppresses malformed release responses" {
  export RELEASE_SERVER_RAW_RESPONSE="{"
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_stderr ""
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

# bats test_tags=platform,platform:macos
@test "sloctl suggests Homebrew upgrade for Homebrew installs" {
  if [ "$(uname -s)" != "Darwin" ]; then
    skip "native Homebrew compatibility is tested on macOS"
  fi

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
  stderr="$(normalize_tty_output <<< "$stderr")"
  assert_stderr "$expected"
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

select_default_update_action() {
  export SLOCTL_TEST_TTY_INPUT=$'\n'
}

run_sloctl_with_tty_stderr() {
  local binary="sloctl"
  if has_bats_tag platform; then
    binary="$(native_sloctl_binary)"
  fi
  run_sloctl_binary_with_tty_stderr "$binary" "$@"
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

run_sloctl_binary_in_windows_console_with_path() {
  local binary="$1"
  local path="$2"
  shift 2
  bats_require_minimum_version 1.5.0
  run --separate-stderr env \
    PATH="$path" \
    SLOCTL_TEST_TTY_JOIN_OUTPUT=1 \
    /usr/bin/python3 "$TEST_INPUTS/run_with_stderr_pty.py" /usr/bin/winpty "$binary" "$@"
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
  local source="/usr/bin/sloctl"
  if has_bats_tag platform; then
    source="$(native_sloctl_binary)"
  fi
  mkdir -p "$(dirname "$target")"
  cp "$source" "$target"
  chmod +x "$target"
}

has_bats_tag() {
  local expected="$1"
  [[ " ${BATS_TEST_TAGS[*]} " == *" $expected "* ]]
}

native_sloctl_binary() {
  local binary="$BATS_TEST_DIRNAME/../bin/sloctl"
  case "$(uname -s)" in
    CYGWIN* | MINGW* | MSYS*) binary+=".exe" ;;
  esac
  printf '%s\n' "$binary"
}

start_release_server() {
  RELEASE_SERVER_START_COUNT=$((RELEASE_SERVER_START_COUNT + 1))
  local port_file="$BATS_TEST_TMPDIR/release-server-$RELEASE_SERVER_START_COUNT.port"
  local error_file="$BATS_TEST_TMPDIR/release-server-$RELEASE_SERVER_START_COUNT.stderr"
  python3 "$TEST_INPUTS/release_server.py" "$port_file" 2> "$error_file" &
  RELEASE_SERVER_PID="$!"

  for _ in {1..300}; do
    if [[ -s "$port_file" ]]; then
      local port
      port="$(cat "$port_file")"
      export SLOCTL_NOTIFICATIONS_RELEASE_URL="http://127.0.0.1:$port/repos/nobl9/sloctl/releases/latest"
      return 0
    fi
    if ! kill -0 "$RELEASE_SERVER_PID" 2> /dev/null; then
      wait "$RELEASE_SERVER_PID" 2> /dev/null || true
      unset RELEASE_SERVER_PID
      local server_error
      server_error="$(< "$error_file")"
      fail "release server exited before startup: ${server_error:-no error output}"
    fi
    sleep 0.1
  done

  fail "release server did not start within 30 seconds"
}

stop_release_server() {
  if [ -n "${RELEASE_SERVER_PID:-}" ]; then
    kill "$RELEASE_SERVER_PID"
    wait "$RELEASE_SERVER_PID" 2> /dev/null || true
    unset RELEASE_SERVER_PID
  fi
}

expire_notification_cache() {
  local cache_file="$XDG_CACHE_HOME/nobl9/sloctl/notifications.json"
  sed -i 's/"lastCheckedAt": "[^"]*"/"lastCheckedAt": "2000-01-01T00:00:00Z"/' "$cache_file"
}

assert_release_requests() {
  local expected="$1"
  local actual=0
  if [ -f "$RELEASE_SERVER_LOG" ]; then
    actual="$(wc -l < "$RELEASE_SERVER_LOG" | tr -d " ")"
  fi
  assert_equal "$actual" "$expected"
}
