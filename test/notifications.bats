#!/usr/bin/env bash
# bats file_tags=unit

setup_file() {
  load "test_helper/load"

  ensure_installed python3
  export NOTIFICATIONS_PYTHON
  NOTIFICATIONS_PYTHON="$(command -v python3)"
  if running_in_container; then
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
    SLOCTL_TEST_TTY_BACKGROUND \
    SLOCTL_TEST_UPGRADE_EXIT_CODE \
    SLOCTL_TEST_UPGRADE_MARKER \
    SLOCTL_TEST_UPGRADE_STDOUT \
    SLOCTL_TEST_BREW_PREFIX \
    SLOCTL_TEST_BREW_QUERY_EXIT_CODE \
    RELEASE_SERVER_BODY_FILE \
    RELEASE_SERVER_HTML_URL \
    RELEASE_SERVER_RAW_RESPONSE \
    RELEASE_SERVER_STATUS \
    RELEASE_SERVER_TAG

  export NO_COLOR=1
  export SLOCTL_ACCESSIBLE_MODE=1
  export HOME="$BATS_TEST_TMPDIR/home"
  export XDG_CACHE_HOME="$BATS_TEST_TMPDIR/cache"
  if has_bats_tag platform:windows; then
    export LOCALAPPDATA="$(cygpath -w "$XDG_CACHE_HOME")"
  fi
  export RELEASE_SERVER_LOG="$BATS_TEST_TMPDIR/release-server.log"
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
    'printf "%s" "${SLOCTL_TEST_UPGRADE_STDOUT:-}"' \
    'exit "${SLOCTL_TEST_UPGRADE_EXIT_CODE:-0}"' \
    > "$tools_dir/go"
  printf '%s\n' \
    '#!/usr/bin/env bash' \
    'if [[ "$*" == "--prefix --installed sloctl" ]]; then' \
    '  [[ -n "${SLOCTL_TEST_BREW_PREFIX:-}" ]] || exit 1' \
    '  printf "%s\n" "$SLOCTL_TEST_BREW_PREFIX"' \
    '  exit "${SLOCTL_TEST_BREW_QUERY_EXIT_CODE:-0}"' \
    'fi' \
    'if [[ -n "${SLOCTL_TEST_UPGRADE_MARKER:-}" ]]; then' \
    '  printf "%s\n" "$*" > "$SLOCTL_TEST_UPGRADE_MARKER"' \
    'fi' \
    'printf "%s" "${SLOCTL_TEST_UPGRADE_STDOUT:-}"' \
    'exit "${SLOCTL_TEST_UPGRADE_EXIT_CODE:-0}"' \
    > "$tools_dir/brew"
  chmod +x "$tools_dir/go" "$tools_dir/brew"
  export PATH="$tools_dir:$PATH"
  RELEASE_SERVER_START_COUNT=0
}

teardown() {
  stop_release_server
}

@test "sloctl shows a version notification on TTY stderr and caches it" {
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr version-notice
  assert_release_requests 1

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_stderr ""
  assert_release_requests 1
}

@test "sloctl shows the release notice before command validation" {
  start_release_server

  run_sloctl_with_tty_stderr config rename-context old
  assert_failure
  assert_notification_stderr failed-command-after-skip
  assert_release_requests 1
}

@test "sloctl includes feature highlights without release metadata" {
  export RELEASE_SERVER_BODY_FILE="$TEST_INPUTS/release-bodies/feature.md"
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr feature-notice
  assert_release_requests 1
}

@test "sloctl skips empty highlight sections" {
  export RELEASE_SERVER_BODY_FILE="$TEST_INPUTS/release-bodies/empty-features-then-bug-fixes.md"
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr bug-fix-notice
}

@test "sloctl includes breaking changes in release highlights" {
  export RELEASE_SERVER_BODY_FILE="$TEST_INPUTS/release-bodies/breaking.md"
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr breaking-notice
}

@test "sloctl includes security fixes in release highlights" {
  export RELEASE_SERVER_BODY_FILE="$TEST_INPUTS/release-bodies/fixed-vulnerabilities.md"
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr vulnerability-notice
}

@test "sloctl preserves release highlights that have no author metadata" {
  export RELEASE_SERVER_BODY_FILE="$TEST_INPUTS/release-bodies/feature-without-author.md"
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr feature-without-author-notice
}

@test "sloctl shows a version notice for releases without highlights" {
  export RELEASE_SERVER_BODY_FILE="$TEST_INPUTS/release-bodies/maintenance.md"
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr version-notice
}

@test "sloctl appends update choices to the same release notice" {
  local manual_binary="$BATS_TEST_TMPDIR/manual/sloctl"
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$manual_binary"
  copy_sloctl_binary "$go_binary"
  export RELEASE_SERVER_BODY_FILE="$TEST_INPUTS/release-bodies/features-with-details.md"
  select_update_action skip
  start_release_server

  run_sloctl_binary_with_tty_stderr "$manual_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr features-with-details-notice

  expire_notification_cache
  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr features-with-details-notice go-update-options
  assert_release_requests 2
}

@test "sloctl skips the notification until the next version" {
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  select_update_action skip-until-next-version
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr go-update-prompt
  assert_release_requests 1

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_stderr ""
  assert_release_requests 1

  expire_notification_cache
  stop_release_server
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_stderr ""
  assert_release_requests 2

  expire_notification_cache
  stop_release_server
  export RELEASE_SERVER_TAG=v1.2.0
  export RELEASE_SERVER_HTML_URL=https://github.com/nobl9/sloctl/releases/tag/v1.2.0
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_notification_stderr next-version-update-prompt
  assert_release_requests 3
}

@test "sloctl runs the selected Go update and exits without running the command" {
  select_update_action run-upgrade
  export SLOCTL_TEST_UPGRADE_MARKER="$BATS_TEST_TMPDIR/upgrade-ran"
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_output ""
  assert_notification_stderr go-update-prompt
  assert [ -f "$SLOCTL_TEST_UPGRADE_MARKER" ]
  assert_equal \
    "$(< "$SLOCTL_TEST_UPGRADE_MARKER")" \
    "install github.com/nobl9/sloctl/cmd/sloctl@latest"
  assert_release_requests 1
}

# bats test_tags=platform,platform:unix
@test "sloctl preserves command JSON output after a failed Go update" {
  select_update_action run-upgrade
  export SLOCTL_TEST_UPGRADE_EXIT_CODE=22
  export SLOCTL_TEST_UPGRADE_STDOUT=$'Downloading update...\n'
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" \
    --config "$BATS_TEST_DIRNAME/inputs/config/single-context-config.toml" \
    config current-context -v -o json
  assert_success_joined_output
  assert_output - < "$BATS_TEST_DIRNAME/outputs/config/get-current-context-minimal.json"
  assert_notification_stderr failed-go-update
  assert_release_requests 1
}

@test "sloctl exits without running the command when the update prompt is interrupted" {
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
  assert_notification_stderr go-update-prompt
  assert_release_requests 2
}

@test "sloctl does not show version notification when opted out" {
  start_release_server
  export SLOCTL_NO_NOTIFICATIONS=1

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_stderr ""
  assert_release_requests 0
}

@test "sloctl does not show version notification in CI" {
  start_release_server
  export CI=true

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_stderr ""
  assert_release_requests 0
}

@test "sloctl does not show version notification without TTY stderr" {
  start_release_server

  run_sloctl version
  assert_success_joined_output
  assert_stderr ""
  assert_release_requests 0
}

@test "sloctl update prompts use the shared light and dark accents" {
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  export SLOCTL_ACCESSIBLE_MODE=0
  export SLOCTL_TEST_TTY_INPUT_WHEN_RAW=1
  export SLOCTL_TEST_TTY_INPUT=$'\r'
  export TERM=xterm-256color COLORTERM=truecolor
  unset NO_COLOR CLICOLOR CLICOLOR_FORCE
  start_release_server

  local background accent
  for background in light dark; do
    if [[ "$background" == light ]]; then
      accent='0;129;158'
    else
      accent='0;186;211'
    fi
    XDG_CACHE_HOME="$BATS_TEST_TMPDIR/cache-$background" \
      SLOCTL_TEST_TTY_BACKGROUND="$background" run_sloctl_binary_with_tty_stderr "$go_binary" version
    assert_success_joined_output
    assert_sloctl_version_output
    # The terminal renderer can redraw the form; check the title's color independently.
    assert_stderr --regexp $'\e''\[[0-9;]*38;2;'"$accent"'[0-9;]*mChoose update action'
  done
  assert_release_requests 2
}

# bats test_tags=platform,platform:unix
@test "sloctl shows the new version notification and update form on supported terminals" {
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
@test "sloctl in a legacy Windows terminal shows release highlights without waiting for input" {
  local go_binary="$HOME/go/bin/sloctl.exe"
  local native_path="${PATH#*:}"
  copy_sloctl_binary "$go_binary"
  export GOBIN="$(cygpath -w "$(dirname "$go_binary")")"
  export MSYS=disable_pcon
  export RELEASE_SERVER_BODY_FILE="$TEST_INPUTS/release-bodies/features-with-details.md"
  unset SLOCTL_TEST_TTY_INPUT
  start_release_server

  local accessible_mode
  for accessible_mode in 0 1; do
    export SLOCTL_ACCESSIBLE_MODE="$accessible_mode"
    rm -f "$XDG_CACHE_HOME/nobl9/sloctl/notifications.json"
    run_sloctl_binary_with_path "$go_binary" "$native_path" version
    assert_success_joined_output
    assert_sloctl_version_output
    assert_notification_stderr features-with-details-notice
  done
  assert_release_requests 2
}

# bats test_tags=platform,platform:windows
@test "sloctl in a native Windows console shows the notification without the update form" {
  if [[ "$(uname -s)" != MINGW* && "$(uname -s)" != CYGWIN* ]]; then
    skip "Windows-specific compatibility test"
  fi

  local go_binary="$HOME/go/bin/sloctl.exe"
  local native_path="${PATH#*:}"
  copy_sloctl_binary "$go_binary"
  export GOBIN="$(cygpath -w "$(dirname "$go_binary")")"
  start_release_server

  run_sloctl_binary_in_windows_console_with_path "$go_binary" "$native_path" version
  assert_success_joined_output
  assert_output --partial "New sloctl version v1.1.0 is available!"
  refute_output --partial "Update with:"
  refute_output --partial "Choose update action"
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
  assert_sloctl_version_output
  assert_stderr ""

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_stderr ""
  assert_release_requests 1
}

@test "sloctl suppresses malformed release responses" {
  export RELEASE_SERVER_RAW_RESPONSE="{"
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_stderr ""
  assert_release_requests 1
}

@test "sloctl still shows notification when cache cannot be written" {
  export XDG_CACHE_HOME="$BATS_TEST_TMPDIR/cache-file"
  touch "$XDG_CACHE_HOME"
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr version-notice
  assert_release_requests 1
}

@test "sloctl warns when skip until next version cannot be saved" {
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  select_update_action skip-until-next-version
  export XDG_CACHE_HOME="$BATS_TEST_TMPDIR/cache-file"
  touch "$XDG_CACHE_HOME"
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr skip-preference-error
  assert_release_requests 1

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr skip-preference-error
  assert_release_requests 2
}

@test "sloctl checks again when the cached timestamp is in the future" {
  start_release_server

  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr version-notice
  assert_release_requests 1

  set_notification_cache_timestamp "2099-01-01T00:00:00Z"
  run_sloctl_with_tty_stderr version
  assert_success_joined_output
  assert_notification_stderr version-notice
  assert_release_requests 2
}

# bats test_tags=platform,platform:unix,platform:macos
@test "sloctl runs Homebrew upgrade through the brew command in PATH" {
  local cellar_binary="$BATS_TEST_TMPDIR/homebrew/Cellar/sloctl/1.2.0/bin/sloctl"
  local linked_binary="$BATS_TEST_TMPDIR/homebrew/bin/sloctl"
  copy_sloctl_binary "$cellar_binary"
  mkdir -p "$(dirname "$linked_binary")"
  ln -s "$cellar_binary" "$linked_binary"
  export SLOCTL_TEST_BREW_PREFIX="$BATS_TEST_TMPDIR/homebrew/Cellar/sloctl/1.2.0"
  export SLOCTL_TEST_UPGRADE_MARKER="$BATS_TEST_TMPDIR/upgrade-ran"
  select_update_action run-upgrade
  start_release_server

  run_sloctl_binary_with_tty_stderr "$linked_binary" version
  assert_success_joined_output
  assert_output ""
  assert_notification_stderr homebrew-update-prompt
  assert_equal "$(< "$SLOCTL_TEST_UPGRADE_MARKER")" "upgrade sloctl"
  assert_release_requests 1
}

@test "sloctl suggests go install for Go bin installs" {
  export HOME="$BATS_TEST_TMPDIR/home"
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  select_update_action skip
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr go-update-prompt
}

@test "sloctl shows the release notice when Go is unavailable" {
  local go_binary="$HOME/go/bin/sloctl"
  local empty_path="$BATS_TEST_TMPDIR/empty-path"
  copy_sloctl_binary "$go_binary"
  mkdir -p "$empty_path"
  unset SLOCTL_TEST_TTY_INPUT
  start_release_server

  run_sloctl_binary_with_path "$go_binary" "$empty_path" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr version-notice
}

@test "sloctl shows the release notice when the matching Homebrew is unavailable" {
  local cellar_binary="$BATS_TEST_TMPDIR/opt/homebrew/Cellar/sloctl/1.2.0/bin/sloctl"
  copy_sloctl_binary "$cellar_binary"
  unset SLOCTL_TEST_TTY_INPUT
  start_release_server

  run_sloctl_binary_with_tty_stderr "$cellar_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr version-notice
}

@test "sloctl shows the release notice for unrecognized installs" {
  local manual_binary="$BATS_TEST_TMPDIR/manual/sloctl"
  copy_sloctl_binary "$manual_binary"
  unset SLOCTL_TEST_TTY_INPUT
  start_release_server

  run_sloctl_binary_with_tty_stderr "$manual_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr version-notice
}

# bats test_tags=platform,platform:unix
@test "sloctl defaults to Skip without running the updater" {
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  select_default_update_action
  export SLOCTL_TEST_UPGRADE_MARKER="$BATS_TEST_TMPDIR/upgrade-ran"
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr go-update-prompt
  assert [ ! -e "$SLOCTL_TEST_UPGRADE_MARKER" ]
  assert_release_requests 1
}

# bats test_tags=platform,platform:unix
@test "sloctl continues without updating when the accessible prompt reaches EOF" {
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  export SLOCTL_TEST_TTY_INPUT=$'\x04'
  export SLOCTL_TEST_UPGRADE_MARKER="$BATS_TEST_TMPDIR/upgrade-ran"
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr go-update-prompt
  assert [ ! -e "$SLOCTL_TEST_UPGRADE_MARKER" ]
  assert_release_requests 1
}

# bats test_tags=platform,platform:unix
@test "sloctl runs an accessible update choice followed by EOF" {
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  export SLOCTL_TEST_TTY_INPUT=$'1\x04\x04'
  export SLOCTL_TEST_UPGRADE_MARKER="$BATS_TEST_TMPDIR/upgrade-ran"
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_output ""
  assert_notification_stderr go-update-prompt
  assert_equal "$(< "$SLOCTL_TEST_UPGRADE_MARKER")" "install github.com/nobl9/sloctl/cmd/sloctl@latest"
}

# bats test_tags=platform,platform:unix
@test "sloctl retries invalid accessible input before updating" {
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  export SLOCTL_TEST_TTY_INPUT=$'x\n1\n'
  export SLOCTL_TEST_UPGRADE_MARKER="$BATS_TEST_TMPDIR/upgrade-ran"
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_output ""
  assert_notification_stderr update-prompt-invalid
  assert_equal "$(< "$SLOCTL_TEST_UPGRADE_MARKER")" "install github.com/nobl9/sloctl/cmd/sloctl@latest"
  assert_release_requests 1
}

# bats test_tags=platform,platform:unix
@test "sloctl continues after invalid accessible input followed by EOF" {
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  export SLOCTL_TEST_TTY_INPUT=$'x\n\x04'
  export SLOCTL_TEST_UPGRADE_MARKER="$BATS_TEST_TMPDIR/upgrade-ran"
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr update-prompt-invalid-eof
  assert [ ! -e "$SLOCTL_TEST_UPGRADE_MARKER" ]
  assert_release_requests 1
}

# bats test_tags=platform,platform:unix
@test "sloctl does not update an external hard link to a Go installation" {
  local go_binary="$HOME/go/bin/sloctl"
  local manual_binary="$BATS_TEST_TMPDIR/manual/sloctl"
  copy_sloctl_binary "$go_binary"
  mkdir -p "$(dirname "$manual_binary")"
  ln "$go_binary" "$manual_binary"
  export SLOCTL_TEST_UPGRADE_MARKER="$BATS_TEST_TMPDIR/upgrade-ran"
  unset SLOCTL_TEST_TTY_INPUT
  start_release_server

  run_sloctl_binary_with_tty_stderr "$manual_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr version-notice
  assert [ ! -e "$SLOCTL_TEST_UPGRADE_MARKER" ]
}

# bats test_tags=platform,platform:unix
@test "sloctl does not update a manual binary linked from GOBIN" {
  local manual_binary="$BATS_TEST_TMPDIR/manual/sloctl"
  copy_sloctl_binary "$manual_binary"
  mkdir -p "$HOME/go/bin"
  ln -s "$manual_binary" "$HOME/go/bin/sloctl"
  export SLOCTL_TEST_UPGRADE_MARKER="$BATS_TEST_TMPDIR/upgrade-ran"
  unset SLOCTL_TEST_TTY_INPUT
  start_release_server

  run_sloctl_binary_with_tty_stderr "$manual_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr version-notice
  assert [ ! -e "$SLOCTL_TEST_UPGRADE_MARKER" ]
}

# bats test_tags=platform,platform:unix,platform:macos
@test "sloctl does not update an unrelated installation through Homebrew" {
  local manual_binary="$BATS_TEST_TMPDIR/manual/sloctl"
  export SLOCTL_TEST_BREW_PREFIX="$BATS_TEST_TMPDIR/homebrew/sloctl"
  copy_sloctl_binary "$manual_binary"
  copy_sloctl_binary "$SLOCTL_TEST_BREW_PREFIX/bin/sloctl"
  export SLOCTL_TEST_UPGRADE_MARKER="$BATS_TEST_TMPDIR/upgrade-ran"
  unset SLOCTL_TEST_TTY_INPUT
  start_release_server

  run_sloctl_binary_with_tty_stderr "$manual_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr version-notice
  assert [ ! -e "$SLOCTL_TEST_UPGRADE_MARKER" ]
}

# bats test_tags=platform,platform:unix,platform:macos
@test "sloctl does not update a renamed Homebrew executable" {
  export SLOCTL_TEST_BREW_PREFIX="$BATS_TEST_TMPDIR/homebrew/Cellar/sloctl/1.2.0"
  local backup_binary="$SLOCTL_TEST_BREW_PREFIX/bin/sloctl.backup"
  copy_sloctl_binary "$SLOCTL_TEST_BREW_PREFIX/bin/sloctl"
  copy_sloctl_binary "$backup_binary"
  export SLOCTL_TEST_UPGRADE_MARKER="$BATS_TEST_TMPDIR/upgrade-ran"
  unset SLOCTL_TEST_TTY_INPUT
  start_release_server

  run_sloctl_binary_with_tty_stderr "$backup_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr version-notice
  assert [ ! -e "$SLOCTL_TEST_UPGRADE_MARKER" ]
}

# bats test_tags=platform,platform:unix,platform:macos
@test "sloctl continues when the Homebrew installation query fails" {
  export SLOCTL_TEST_BREW_PREFIX="$BATS_TEST_TMPDIR/homebrew/sloctl"
  local brew_binary="$SLOCTL_TEST_BREW_PREFIX/bin/sloctl"
  copy_sloctl_binary "$brew_binary"
  export SLOCTL_TEST_BREW_QUERY_EXIT_CODE=22
  export SLOCTL_TEST_UPGRADE_MARKER="$BATS_TEST_TMPDIR/upgrade-ran"
  unset SLOCTL_TEST_TTY_INPUT
  start_release_server

  run_sloctl_binary_with_tty_stderr "$brew_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  assert_notification_stderr version-notice
  assert [ ! -e "$SLOCTL_TEST_UPGRADE_MARKER" ]
}

# bats test_tags=platform,platform:unix
@test "sloctl loads Bash completions without a notification or input" {
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  unset SLOCTL_TEST_TTY_INPUT
  start_release_server

  run_sloctl_binary_with_tty_stderr bash --noprofile --norc -e -c '
    eval "$("$1" --no-config-file completion bash)"
    complete -p sloctl > /dev/null
    declare -F __start_sloctl
  ' bash "$go_binary"
  assert_success_joined_output
  assert_output "__start_sloctl"
  assert_stderr ""
  assert_release_requests 0
}

# bats test_tags=platform,platform:unix
@test "sloctl serves dynamic shell completions without notifications" {
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  unset SLOCTL_TEST_TTY_INPUT
  start_release_server

  local request
  for request in __complete __completeNoDesc; do
    run_sloctl_binary_with_tty_stderr "$go_binary" "$request" --no-config-file version ""
    assert_success_joined_output
    assert_output ":0"
    assert_stderr "Completion ended with directive: ShellCompDirectiveDefault"
    assert_release_requests 0
  done
}

# bats test_tags=platform,platform:unix
@test "sloctl defaults to Skip in the normal update form" {
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  unset SLOCTL_ACCESSIBLE_MODE
  export SLOCTL_TEST_TTY_INPUT=$'\r'
  export SLOCTL_TEST_TTY_INPUT_WHEN_RAW=1
  export SLOCTL_TEST_UPGRADE_MARKER="$BATS_TEST_TMPDIR/upgrade-ran"
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_sloctl_version_output
  # Normal form output includes terminal cursor movements.
  assert_stderr --partial "New sloctl version v1.1.0 is available!"
  assert [ ! -e "$SLOCTL_TEST_UPGRADE_MARKER" ]
}

# bats test_tags=platform,platform:unix
@test "sloctl runs an explicitly selected update in the normal form" {
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  unset SLOCTL_ACCESSIBLE_MODE
  export SLOCTL_TEST_TTY_INPUT=$'\x1b[A\r'
  export SLOCTL_TEST_TTY_INPUT_WHEN_RAW=1
  export SLOCTL_TEST_UPGRADE_MARKER="$BATS_TEST_TMPDIR/upgrade-ran"
  start_release_server

  run_sloctl_binary_with_tty_stderr "$go_binary" version
  assert_success_joined_output
  assert_output ""
  assert_equal "$(< "$SLOCTL_TEST_UPGRADE_MARKER")" "install github.com/nobl9/sloctl/cmd/sloctl@latest"
}

assert_notification_stderr() {
  local name
  local files=()
  for name in "$@"; do
    files+=("$TEST_OUTPUTS/$name.stderr")
  done
  local expected
  expected="$(cat "${files[@]}" | normalize_tty_output)"
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
  if ! running_in_container; then
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
  run --separate-stderr env PATH="$path" "$NOTIFICATIONS_PYTHON" "$TEST_INPUTS/run_with_stderr_pty.py" "$binary" "$@"
}

copy_sloctl_binary() {
  local target="$1"
  local source="/usr/local/bin/sloctl"
  if ! running_in_container; then
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

running_in_container() {
  [ -f "/.dockerenv" ] || [ -f "/run/.containerenv" ]
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
