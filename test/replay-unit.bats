#!/usr/bin/env bash
# bats file_tags=unit

bats_require_minimum_version 1.5.0

setup_file() {
  export TEST_INPUTS="$TEST_SUITE_INPUTS/replay-unit"
  export REPLAY_REQUEST_LOG="$BATS_FILE_TMPDIR/requests.jsonl"
  local port_file="$BATS_FILE_TMPDIR/server-port"

  python3 "$TEST_INPUTS/server.py" "$port_file" "$REPLAY_REQUEST_LOG" &
  export REPLAY_SERVER_PID=$!

  for _ in {1..50}; do
    if [[ -s "$port_file" ]]; then
      break
    fi
    sleep 0.1
  done
  [[ -s "$port_file" ]]

  local port
  port="$(< "$port_file")"
  export SLOCTL_URL="http://127.0.0.1:${port}/api"
  export SLOCTL_ACCESS_TOKEN="test-access-token"
  export SLOCTL_DISABLE_OKTA=true
  export SLOCTL_ORGANIZATION="test-organization"
  export SLOCTL_PROJECT=default
}

setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"
}

teardown_file() {
  if kill -0 "$REPLAY_SERVER_PID" 2> /dev/null; then
    kill "$REPLAY_SERVER_PID"
    wait "$REPLAY_SERVER_PID" || true
  fi
}

@test "replay config preserves project scope and source SLO" {
  run_sloctl --no-config-file replay -f "$TEST_INPUTS/replay.yaml"

  assert_success_joined_output
  assert_stderr --partial "Successfully finished operations for all 2 SLOs."

  assert_equal \
    "$(jq -sc 'map({project, slo, sourceSlo})' "$REPLAY_REQUEST_LOG")" \
    '[{"project":"replay-project-a","slo":"replay-slo-a","sourceSlo":{"slo":"replay-source-slo","project":"replay-source-project","objectivesMap":[{"source":"acceptable","target":"objective-1"},{"source":"alarming","target":"objective-2"}]}},{"project":"replay-project-b","slo":"replay-slo-b","sourceSlo":null}]'
}
