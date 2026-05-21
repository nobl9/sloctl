#!/usr/bin/env bash
# bats file_tags=unit

setup_file() {
  export TEST_INPUTS="$BATS_TEST_DIRNAME/inputs/notifications"
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
  export XDG_CACHE_HOME="$BATS_TMPDIR/cache"
}

teardown() {
  if [ -n "${RELEASE_PROXY_PID:-}" ]; then
    kill "$RELEASE_PROXY_PID"
    wait "$RELEASE_PROXY_PID" 2> /dev/null || true
  fi
}

start_release_proxy() {
  local port_file="$BATS_TMPDIR/release-proxy-port"
  local cert_config="$BATS_TMPDIR/api-github-openssl.cnf"
  local cert_file="$BATS_TMPDIR/api-github.pem"
  local key_file="$BATS_TMPDIR/api-github-key.pem"

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

run_sloctl_with_stderr_pty() {
  bats_require_minimum_version 1.5.0
  run --separate-stderr python3 "$TEST_INPUTS/run_with_stderr_pty.py" sloctl "$@"
}

refute_stderr() {
  local actual="$stderr"
  output="$actual" refute_output "$@"
}

@test "sloctl shows a feature notification on TTY stderr and caches it" {
  start_release_proxy

  run_sloctl_with_stderr_pty version
  assert_success_joined_output
  assert_stderr --partial "╭"
  assert_stderr --partial "New sloctl features in"
  assert_stderr --partial "v1.1.0"
  assert_stderr --partial "Features"
  assert_stderr --partial "Add notification tests"
  assert_stderr --partial "https://github.com/nobl9/sloctl/releases/tag/v1.1.0"

  run_sloctl_with_stderr_pty version
  assert_success_joined_output
  assert_stderr ""
}

@test "sloctl does not show feature notification when opted out" {
  start_release_proxy
  export SLOCTL_NO_NOTIFICATIONS=1

  run_sloctl_with_stderr_pty version
  assert_success_joined_output
  assert_stderr ""
}

@test "sloctl does not show feature notification without TTY stderr" {
  start_release_proxy

  run_sloctl version
  assert_success_joined_output
  refute_stderr --partial "New sloctl feature"
}
