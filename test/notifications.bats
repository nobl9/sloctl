#!/usr/bin/env bash
# bats file_tags=unit

setup_file() {
  load "test_helper/load"

  ensure_installed python3
  if [ -f "/.dockerenv" ] || [ -f "/run/.containerenv" ]; then
    cp /usr/bin/sloctl /usr/local/bin/sloctl
  fi

  export TEST_INPUTS="$BATS_TEST_DIRNAME/inputs/notifications"
  export TEST_OUTPUTS="$BATS_TEST_DIRNAME/outputs/notifications"
}

setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"

  unset \
    CI \
    ALL_PROXY \
    HTTPS_PROXY \
    HTTP_PROXY \
    NO_PROXY \
    SSL_CERT_FILE \
    all_proxy \
    https_proxy \
    http_proxy \
    no_proxy \
    GOBIN \
    GOPATH \
    SLOCTL_NO_NOTIFICATIONS \
    SLOCTL_TEST_TTY_INPUT \
    SLOCTL_TEST_TTY_INPUT_WHEN_RAW \
    SLOCTL_TEST_UPGRADE_EXIT_CODE \
    SLOCTL_TEST_UPGRADE_MARKER \
    RELEASE_SERVER_BODY_FILE \
    RELEASE_SERVER_HTML_URL \
    RELEASE_SERVER_RAW_RESPONSE \
    RELEASE_SERVER_STATUS \
    RELEASE_SERVER_TAG

  export NO_COLOR=1
  export SLOCTL_ACCESSIBLE_MODE=1
  export HOME="$BATS_TEST_TMPDIR/home"
  export XDG_CACHE_HOME="$BATS_TMPDIR/cache-$BATS_TEST_NUMBER"
  export LocalAppData="$BATS_TMPDIR/cache-$BATS_TEST_NUMBER"
  export RELEASE_SERVER_LOG="$BATS_TMPDIR/release-server-$BATS_TEST_NUMBER.log"
  export SLOCTL_TEST_TTY_INPUT=$'1\n'
  local tools_dir="$BATS_TEST_TMPDIR/tools"
  mkdir -p "$tools_dir"
  printf '%s\n' \
    '#!/usr/bin/env bash' \
    'if [[ "${1:-}" == "env" ]]; then' \
    '  printf "{\"GOBIN\":\"%s\",\"GOPATH\":\"%s\"}\n" "${GOBIN:-}" "${GOPATH:-${HOME}/go}"' \
    '  exit 0' \
    'fi' \
    'if [[ -n "${SLOCTL_TEST_UPGRADE_MARKER:-}" ]]; then' \
    '  printf "%s\n" "$*" > "${SLOCTL_TEST_UPGRADE_MARKER}"' \
    'fi' \
    'exit "${SLOCTL_TEST_UPGRADE_EXIT_CODE:-0}"' \
    > "$tools_dir/go"
  chmod +x "$tools_dir/go"
  export PATH="$tools_dir:$PATH"
  RELEASE_SERVER_START_COUNT=0
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

@test "sloctl shows installation guidance before command validation" {
  start_release_server

  run_sloctl_with_tty_stderr config rename-context old
  assert_failure
  assert_notification_stderr failed-command-after-skip
  assert_release_requests 1
}

@test "sloctl skips the notification until the next version" {
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  select_update_action skip-until-next-version
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr feature-prompt-skip-until-next-version
  assert_release_requests 1

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_stderr ""
  assert_release_requests 1

  expire_notification_cache
  stop_release_server
  use_release_body feature-without-author
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_stderr ""
  assert_release_requests 2

  expire_notification_cache
  stop_release_server
  unset RELEASE_SERVER_BODY_FILE
  export RELEASE_SERVER_TAG=v1.2.0
  export RELEASE_SERVER_HTML_URL=https://github.com/nobl9/sloctl/releases/tag/v1.2.0
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_notification_stderr next-version-prompt-skip-until-next-version
  assert_release_requests 3
}

@test "sloctl defaults to Go update action and exits without running the command" {
  use_release_body maintenance
  select_default_update_action
  export SLOCTL_TEST_UPGRADE_MARKER="$BATS_TEST_TMPDIR/upgrade-ran"
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_output ""
  assert_notification_stderr version-prompt-run-upgrade
  assert [ -f "$SLOCTL_TEST_UPGRADE_MARKER" ]
  assert_equal \
    "$(< "$SLOCTL_TEST_UPGRADE_MARKER")" \
    "install github.com/nobl9/sloctl/cmd/sloctl@latest"
  assert_release_requests 1
}

@test "sloctl reports a failed Go update and exits without running the command" {
  use_release_body maintenance
  select_default_update_action
  export SLOCTL_TEST_UPGRADE_EXIT_CODE=22
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_failure 1
  assert_output ""
  assert_notification_stderr version-prompt-failed-upgrade
  assert_release_requests 1
}

@test "sloctl exits without running the command when the update prompt is interrupted" {
  use_release_body maintenance
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  unset SLOCTL_ACCESSIBLE_MODE
  export SLOCTL_TEST_TTY_INPUT=$'\x03'
  export SLOCTL_TEST_TTY_INPUT_WHEN_RAW=1
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_failure 130
  assert_output ""
  assert_stderr --partial "New sloctl version v1.1.0 is available!"
  assert_release_requests 1

  unset SLOCTL_TEST_TTY_INPUT_WHEN_RAW
  export SLOCTL_ACCESSIBLE_MODE=1
  select_update_action skip
  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr version-prompt-run-upgrade
  assert_release_requests 2
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
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  select_update_action skip
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
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
  local go_binary="$HOME/go/bin/sloctl.exe"
  local native_path="${PATH#*:}"
  copy_sloctl_binary "$go_binary"
  export GOBIN="$(cygpath -w "$(dirname "$go_binary")")"
  start_release_server

  run_sloctl_binary_in_windows_console_with_path "$go_binary" "$native_path" version
  assert_success_joined_output
  assert_output --partial "New sloctl version v1.1.0 is available!"
  assert_output --partial "Update with: go install github.com/nobl9/sloctl/cmd/sloctl@latest"
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

@test "sloctl shows fixed vulnerability notification" {
  use_release_body fixed-vulnerabilities
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr fixed-vulnerabilities-prompt-skip
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

@test "sloctl does not show notification for an older release" {
  export RELEASE_SERVER_TAG=v0.9.0
  export RELEASE_SERVER_HTML_URL=https://github.com/nobl9/sloctl/releases/tag/v0.9.0
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

@test "sloctl warns when skip until next version cannot be saved" {
  use_release_body maintenance
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  select_update_action skip-until-next-version
  export XDG_CACHE_HOME="$BATS_TEST_TMPDIR/cache-file"
  touch "$XDG_CACHE_HOME"
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr version-prompt-skip-until-cache-error
  assert_release_requests 1

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr version-prompt-skip-until-cache-error
  assert_release_requests 2
}

@test "sloctl checks again when the cached timestamp is in the future" {
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr feature-prompt-skip
  assert_release_requests 1

  set_notification_cache_timestamp "2099-01-01T00:00:00Z"
  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr feature-prompt-skip
  assert_release_requests 2
}

# bats test_tags=platform,platform:macos
@test "sloctl runs Homebrew upgrade with the matching Homebrew executable" {
  if [ "$(uname -s)" != "Darwin" ]; then
    skip "native Homebrew compatibility is tested on macOS"
  fi

  use_release_body maintenance
  local cellar_binary="$BATS_TEST_TMPDIR/opt/homebrew/Cellar/sloctl/1.2.0/bin/sloctl"
  local linked_binary="$BATS_TEST_TMPDIR/opt/homebrew/bin/sloctl"
  local brew_binary="$BATS_TEST_TMPDIR/opt/homebrew/bin/brew"
  copy_sloctl_binary "$cellar_binary"
  mkdir -p "$(dirname "$linked_binary")"
  ln -s "$cellar_binary" "$linked_binary"
  export SLOCTL_TEST_UPGRADE_MARKER="$BATS_TEST_TMPDIR/upgrade-ran"
  printf '%s\n' \
    '#!/usr/bin/env bash' \
    'printf "%s\n" "$*" > "${SLOCTL_TEST_UPGRADE_MARKER}"' \
    > "$brew_binary"
  chmod +x "$brew_binary"
  select_default_update_action
  start_release_server

  run_sloctl_binary_with_tty_stderr "$linked_binary" version
  assert_success_joined_output
  assert_output ""
  assert_notification_stderr install-homebrew-prompt
  assert_equal "$(< "$SLOCTL_TEST_UPGRADE_MARKER")" "upgrade sloctl"
  assert_release_requests 1
}

@test "sloctl suggests go install for Go bin installs" {
  use_release_body maintenance
  export HOME="$BATS_TEST_TMPDIR/home"
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  select_update_action skip
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr install-go-prompt
}

@test "sloctl shows the installation guide when Go is unavailable" {
  use_release_body maintenance
  local go_binary="$HOME/go/bin/sloctl"
  local empty_path="$BATS_TEST_TMPDIR/empty-path"
  copy_sloctl_binary "$go_binary"
  mkdir -p "$empty_path"
  unset SLOCTL_TEST_TTY_INPUT
  start_release_server

  run_sloctl_binary_with_path "$go_binary" "$empty_path" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr version-prompt-skip
}

@test "sloctl shows the installation guide when the matching Homebrew is unavailable" {
  use_release_body maintenance
  local cellar_binary="$BATS_TEST_TMPDIR/opt/homebrew/Cellar/sloctl/1.2.0/bin/sloctl"
  copy_sloctl_binary "$cellar_binary"
  unset SLOCTL_TEST_TTY_INPUT
  start_release_server

  run_sloctl_binary_with_tty_stderr "$cellar_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr version-prompt-skip
}

@test "sloctl shows installation guide for unrecognized installs" {
  use_release_body maintenance
  local manual_binary="$BATS_TEST_TMPDIR/manual/sloctl"
  copy_sloctl_binary "$manual_binary"
  unset SLOCTL_TEST_TTY_INPUT
  start_release_server

  run_sloctl_binary_with_tty_stderr "$manual_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr version-prompt-skip
}

assert_notification_stderr() {
  local name="$1"
  local expected
  expected="$(normalize_tty_output < "$TEST_OUTPUTS/$name.stderr")"
  stderr="$(normalize_tty_output <<< "$stderr")"
  assert_stderr "$expected"
}

normalize_tty_output() {
  sed \
    -e 's/\r//g' \
    -e 's/[[:blank:]]$//' \
    -e "s#${BATS_TEST_TMPDIR}#<BATS_TEST_TMPDIR>#g"
}

assert_sloctl_version_output() {
  # The version prefix is fixed by the test target; suffix and build metadata vary by runner.
  assert_output --partial "sloctl/v1.0.0"
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

run_sloctl_binary_in_windows_console_with_path() {
  local binary="$1"
  local path="$2"
  shift 2
  bats_require_minimum_version 1.5.0

  local helper python
  helper="$(cygpath -w "$TEST_INPUTS/run_with_windows_pty.py")"
  python="$(cygpath -u "$pythonLocation")/python.exe"
  binary="$(cygpath -w "$binary")"

  run --separate-stderr env \
    PATH="$path" \
    "$python" "$helper" "$binary" "$@"
}

run_sloctl_binary_with_path() {
  local binary="$1"
  local path="$2"
  shift 2
  bats_require_minimum_version 1.5.0
  run --separate-stderr env PATH="$path" /usr/bin/python3 "$TEST_INPUTS/run_with_stderr_pty.py" "$binary" "$@"
}

copy_sloctl_binary() {
  local target="$1"
  local source="/usr/local/bin/sloctl"
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
  python3 "$TEST_INPUTS/release_server.py" "$port_file" "$RELEASE_SERVER_PORT" 2> "$error_file" &
  RELEASE_SERVER_PID="$!"

  for _ in {1..300}; do
    if [[ -s "$port_file" ]]; then
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
  set_notification_cache_timestamp "2000-01-01T00:00:00Z"
}

set_notification_cache_timestamp() {
  local timestamp="$1"
  local cache_file="$XDG_CACHE_HOME/nobl9/sloctl/notifications.json"
  sed -i 's/"lastCheckedAt": "[^"]*"/"lastCheckedAt": "'"$timestamp"'"/' "$cache_file"
  assert_equal "$(jq -r '.lastCheckedAt' "$cache_file")" "$timestamp"
}

assert_release_requests() {
  local expected="$1"
  local actual=0
  if [ -f "$RELEASE_SERVER_LOG" ]; then
    actual="$(wc -l < "$RELEASE_SERVER_LOG" | tr -d " ")"
  fi
  assert_equal "$actual" "$expected"
}
