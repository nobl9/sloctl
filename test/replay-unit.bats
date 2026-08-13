#!/usr/bin/env bash
# bats file_tags=unit

bats_require_minimum_version 1.5.0

setup_file() {
  export TEST_INPUTS="$TEST_SUITE_INPUTS/replay-unit"
  export REPLAY_REQUEST_LOG="$BATS_FILE_TMPDIR/requests.jsonl"
  export REPLAY_AVAILABILITY_LOG="$BATS_FILE_TMPDIR/availability.jsonl"
  export REPLAY_CONTROL_LOG="$BATS_FILE_TMPDIR/control.jsonl"
  local port_file="$BATS_FILE_TMPDIR/server-port"

  python3 \
    "$TEST_INPUTS/server.py" \
    "$port_file" \
    "$REPLAY_REQUEST_LOG" \
    "$REPLAY_AVAILABILITY_LOG" \
    "$REPLAY_CONTROL_LOG" &
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

  assert_equal \
    "$(jq -Ssc 'map({project, query: (.query | del(.durationValue))}) | sort_by(.project)' "$REPLAY_AVAILABILITY_LOG")" \
    '[{"project":"replay-project-a","query":{"durationUnit":"Minute","sloName":"replay-slo-a","type":"reimport_and_recalculation"}},{"project":"replay-project-b","query":{"durationUnit":"Minute","sloName":"replay-slo-b","type":"reimport_and_recalculation"}}]'
  assert_equal \
    "$(jq -sc 'all(.[]; (.query.durationValue | tonumber) > 0)' "$REPLAY_AVAILABILITY_LOG")" \
    "true"
}

@test "replay list preserves createdAt and coarse status" {
  run_sloctl --no-config-file replay list -o json

  assert_success
  assert_equal \
    "$(jq -c . <<< "$output")" \
    '[{"slo":"replay-slo-a","project":"replay-project-a","createdAt":"2026-08-06T12:34:56Z","status":"in progress"}]'
}

@test "replay cancel and delete commands use the SDK request shapes" {
  run_sloctl --no-config-file replay cancel replay-slo-a -p replay-project-a
  assert_success_joined_output

  run_sloctl --no-config-file replay delete replay-slo-a -p replay-project-a
  assert_success_joined_output

  run_sloctl --no-config-file replay delete --all
  assert_success_joined_output

  assert_equal \
    "$(jq -sc . "$REPLAY_CONTROL_LOG")" \
    '[{"method":"POST","path":"/api/timetravel/cancel","project":"replay-project-a","body":{"project":"replay-project-a","slo":"replay-slo-a"}},{"method":"DELETE","path":"/api/timetravel","project":"replay-project-a","body":{"project":"replay-project-a","slo":"replay-slo-a"}},{"method":"DELETE","path":"/api/timetravel","project":"default","body":{"all":true}}]'
}
