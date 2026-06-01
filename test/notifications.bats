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

  ensure_installed openssl python3

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
  unset RELEASE_PROXY_BODY
  unset RELEASE_PROXY_BODY_FILE
  unset RELEASE_PROXY_HTML_URL
  unset RELEASE_PROXY_RAW_RESPONSE
  unset RELEASE_PROXY_STATUS
  unset RELEASE_PROXY_TAG
  unset SLOCTL_TEST_TTY_COLUMNS
  export XDG_CACHE_HOME="$BATS_TMPDIR/cache-$BATS_TEST_NUMBER"
  export RELEASE_PROXY_LOG="$BATS_TMPDIR/release-proxy-$BATS_TEST_NUMBER.log"
}

teardown() {
  if [ -n "${RELEASE_PROXY_PID:-}" ]; then
    kill "$RELEASE_PROXY_PID"
    wait "$RELEASE_PROXY_PID" 2> /dev/null || true
  fi
}

@test "sloctl shows a feature notification on TTY stderr and caches it" {
  start_release_proxy

  run_sloctl_with_stderr_pty version
  assert_success_joined_output
  assert_stderr_file "feature-notification.stderr"
  assert_release_proxy_requests 1

  run_sloctl_with_stderr_pty version
  assert_success_joined_output
  assert_stderr ""
  assert_release_proxy_requests 1
}

@test "sloctl shows a notification after successful commands" {
  start_release_proxy

  run_sloctl_with_stderr_pty version
  assert_success_joined_output
  assert_stderr_file "feature-notification.stderr"
  assert_release_proxy_requests 1
}

@test "sloctl skips notifications after failed commands" {
  start_release_proxy

  run_sloctl_with_stderr_pty config rename-context old
  assert_failure
  assert_stderr_file "failed-command.stderr"
  assert_release_proxy_requests 0
}

@test "sloctl does not show feature notification when opted out" {
  start_release_proxy
  export SLOCTL_NO_NOTIFICATIONS=1

  run_sloctl_with_stderr_pty version
  assert_success_joined_output
  assert_stderr ""
  assert_release_proxy_requests 0
}

@test "sloctl does not show feature notification in CI" {
  start_release_proxy
  export CI=true

  run_sloctl_with_stderr_pty version
  assert_success_joined_output
  assert_stderr ""
  assert_release_proxy_requests 0
}

@test "sloctl does not show feature notification without TTY stderr" {
  start_release_proxy

  run_sloctl version
  assert_success_joined_output
  assert_stderr ""
  assert_release_proxy_requests 0
}

@test "sloctl shows version notification when release has no feature notes" {
  export RELEASE_PROXY_BODY_FILE="$TEST_INPUTS/release-bodies/maintenance.md"
  start_release_proxy

  run_sloctl_with_stderr_pty version
  assert_success_joined_output
  assert_stderr_file "version-notification.stderr"
  assert_release_proxy_requests 1
}

@test "sloctl uses the first non-empty release notes section" {
  export RELEASE_PROXY_BODY_FILE="$TEST_INPUTS/release-bodies/empty-features-then-bug-fixes.md"
  start_release_proxy

  run_sloctl_with_stderr_pty version
  assert_success_joined_output
  assert_stderr_file "bug-fix-notification.stderr"
  assert_release_proxy_requests 1
}

@test "sloctl keeps nested release-note details from the selected section" {
  export RELEASE_PROXY_BODY_FILE="$TEST_INPUTS/release-bodies/features-with-details.md"
  start_release_proxy

  run_sloctl_with_stderr_pty version
  assert_success_joined_output
  assert_stderr_file "features-with-details.stderr"
  assert_release_proxy_requests 1
}

@test "sloctl shows release note without author metadata" {
  export RELEASE_PROXY_BODY_FILE="$TEST_INPUTS/release-bodies/feature-without-author.md"
  start_release_proxy

  run_sloctl_with_stderr_pty version
  assert_success_joined_output
  assert_stderr_file "feature-without-author.stderr"
  assert_release_proxy_requests 1
}

@test "sloctl does not show notification for current release" {
  export RELEASE_PROXY_TAG=v1.0.0
  export RELEASE_PROXY_HTML_URL=https://github.com/nobl9/sloctl/releases/tag/v1.0.0
  start_release_proxy

  run_sloctl_with_stderr_pty version
  assert_success_joined_output
  assert_stderr ""
  assert_release_proxy_requests 1
}

@test "sloctl suppresses fetch failures and caches the check" {
  export RELEASE_PROXY_STATUS=403
  export RELEASE_PROXY_RAW_RESPONSE="rate limited"
  start_release_proxy

  run_sloctl_with_stderr_pty version
  assert_success_joined_output
  assert_stderr ""

  run_sloctl_with_stderr_pty version
  assert_success_joined_output
  assert_stderr ""
  assert_release_proxy_requests 1
}

@test "sloctl suppresses malformed release responses" {
  export RELEASE_PROXY_RAW_RESPONSE="{"
  start_release_proxy

  run_sloctl_with_stderr_pty version
  assert_success_joined_output
  assert_stderr ""
  assert_release_proxy_requests 1
}

@test "sloctl still shows notification when cache cannot be written" {
  export XDG_CACHE_HOME="$BATS_TEST_TMPDIR/cache-file"
  touch "$XDG_CACHE_HOME"
  start_release_proxy

  run_sloctl_with_stderr_pty version
  assert_success_joined_output
  assert_stderr_file "feature-notification.stderr"
  assert_release_proxy_requests 1
}

@test "sloctl keeps install command on one line when terminal is wide" {
  export RELEASE_PROXY_BODY_FILE="$TEST_INPUTS/release-bodies/maintenance.md"
  export SLOCTL_TEST_TTY_COLUMNS=140
  local tools_dir="$BATS_TEST_TMPDIR/tools"
  mkdir -p "$tools_dir"
  touch "$tools_dir/curl"
  chmod +x "$tools_dir/curl"
  start_release_proxy

  run_sloctl_binary_with_path /usr/bin/sloctl "$tools_dir" version
  assert_success_joined_output
  assert_stderr_file "install-curl-wide.stderr"
}

@test "sloctl suggests Homebrew upgrade for Homebrew installs" {
  export RELEASE_PROXY_BODY_FILE="$TEST_INPUTS/release-bodies/maintenance.md"
  local cellar_binary="$BATS_TEST_TMPDIR/opt/homebrew/Cellar/sloctl/1.2.0/bin/sloctl"
  local linked_binary="$BATS_TEST_TMPDIR/opt/homebrew/bin/sloctl"
  copy_sloctl_binary "$cellar_binary"
  mkdir -p "$(dirname "$linked_binary")"
  ln -s "$cellar_binary" "$linked_binary"
  start_release_proxy

  run_sloctl_binary_with_stderr_pty "$linked_binary" version
  assert_success_joined_output
  assert_stderr_file "install-homebrew.stderr"
}

@test "sloctl suggests go install for Go bin installs" {
  export RELEASE_PROXY_BODY_FILE="$TEST_INPUTS/release-bodies/maintenance.md"
  export HOME="$BATS_TEST_TMPDIR/home"
  local go_binary="$HOME/go/bin/sloctl"
  copy_sloctl_binary "$go_binary"
  start_release_proxy

  run_sloctl_binary_with_stderr_pty "$go_binary" version
  assert_success_joined_output
  assert_stderr_file "install-go.stderr"
}

@test "sloctl falls back to wget when curl is unavailable" {
  export RELEASE_PROXY_BODY_FILE="$TEST_INPUTS/release-bodies/maintenance.md"
  local tools_dir="$BATS_TEST_TMPDIR/tools"
  mkdir -p "$tools_dir"
  touch "$tools_dir/wget"
  chmod +x "$tools_dir/wget"
  start_release_proxy

  run_sloctl_binary_with_path /usr/bin/sloctl "$tools_dir" version
  assert_success_joined_output
  assert_stderr_file "install-wget.stderr"
}

@test "sloctl omits update command when no downloader is available" {
  export RELEASE_PROXY_BODY_FILE="$TEST_INPUTS/release-bodies/maintenance.md"
  local tools_dir="$BATS_TEST_TMPDIR/tools"
  mkdir -p "$tools_dir"
  start_release_proxy

  run_sloctl_binary_with_path /usr/bin/sloctl "$tools_dir" version
  assert_success_joined_output
  assert_stderr_file "no-install-command.stderr"
}

assert_stderr_file() {
  local file="$1"
  assert_stderr - < "$TEST_OUTPUTS/$file"
}

run_sloctl_with_stderr_pty() {
  run_sloctl_binary_with_stderr_pty sloctl "$@"
}

run_sloctl_binary_with_stderr_pty() {
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

copy_sloctl_binary() {
  local target="$1"
  mkdir -p "$(dirname "$target")"
  cp /usr/bin/sloctl "$target"
  chmod +x "$target"
}

start_release_proxy() {
  local port_file="$BATS_TMPDIR/release-proxy-$BATS_TEST_NUMBER.port"
  local cert_config="$BATS_TMPDIR/api-github-$BATS_TEST_NUMBER-openssl.cnf"
  local cert_file="$BATS_TMPDIR/api-github-$BATS_TEST_NUMBER.pem"
  local key_file="$BATS_TMPDIR/api-github-$BATS_TEST_NUMBER-key.pem"

  cat > "$cert_config" << EOF
[req]
distinguished_name = dn
x509_extensions = extensions
prompt = no

[dn]
CN = api.github.com

[extensions]
subjectAltName = DNS:api.github.com
EOF

  openssl req \
    -x509 \
    -newkey rsa:2048 \
    -nodes \
    -keyout "$key_file" \
    -out "$cert_file" \
    -days 1 \
    -config "$cert_config" \
    2> /dev/null

  python3 "$TEST_INPUTS/release_proxy.py" "$port_file" "$cert_file" "$key_file" &
  RELEASE_PROXY_PID="$!"

  for _ in {1..50}; do
    if [ -s "$port_file" ]; then
      local port
      port="$(cat "$port_file")"
      export HTTPS_PROXY="http://127.0.0.1:$port"
      export SSL_CERT_FILE="$cert_file"
      return 0
    fi
    sleep 0.1
  done

  fail "release proxy did not start"
}

assert_release_proxy_requests() {
  local expected="$1"
  local actual=0
  if [ -f "$RELEASE_PROXY_LOG" ]; then
    actual="$(wc -l < "$RELEASE_PROXY_LOG" | tr -d " ")"
  fi
  assert_equal "$actual" "$expected"
}
