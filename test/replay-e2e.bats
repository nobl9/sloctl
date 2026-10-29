#!/usr/bin/env bash
# bats file_tags=e2e

bats_require_minimum_version 1.5.0

setup_file() {
  load "test_helper/load"
  load_lib "bats-assert"

  generate_inputs "$BATS_FILE_TMPDIR"

  run_sloctl apply -f "$TEST_INPUTS/resources.yaml"
  assert_success_joined_output
  export REPLAY_SETUP_COMPLETE=true
}

setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"
}

teardown_file() {
  run_sloctl delete -f "$TEST_INPUTS/resources.yaml"
  if [[ "${REPLAY_SETUP_COMPLETE:-}" == true ]]; then
    assert_success_joined_output
  fi
}

@test "replay config resolves SLOs in separate Projects" {
  run_sloctl replay -f "$TEST_INPUTS/replay.yaml"

  assert_failure
  assert_stderr --partial "The following SLOs are not available for Replay:"
  assert_stderr --partial "['replay-slo-a' SLO in '$TEST_PROJECT' Project]"
  assert_stderr --partial "['replay-slo-b' SLO in '${TEST_PROJECT}-b' Project]"

  if [[ "$stderr" == *"Some of the SLOs marked for Replay were not found"* ]]; then
    fail "Replay treated existing platform SLOs as missing"
  fi
}

@test "replay rejects a missing source SLO before checking availability" {
  run_sloctl replay -f "$TEST_INPUTS/missing-source.yaml"

  assert_failure
  assert_stderr --partial "Some of the SLOs marked for Replay were not found"
  assert_stderr --partial "'replay-missing-source' SLO in '${TEST_PROJECT}-source' Project"
}

@test "replay does not match a source SLO in another Project" {
  run_sloctl replay -f "$TEST_INPUTS/source-wrong-project.yaml"

  assert_failure
  assert_stderr --partial "Some of the SLOs marked for Replay were not found"
  assert_stderr --partial "'replay-slo-a' SLO in '${TEST_PROJECT}-source' Project"
}

@test "replay reports both missing target and source SLOs" {
  run_sloctl replay -f "$TEST_INPUTS/missing-target-and-source.yaml"

  assert_failure
  assert_stderr --partial "Some of the SLOs marked for Replay were not found"
  assert_stderr --partial "'replay-missing-target' SLO in '$TEST_PROJECT' Project"
  assert_stderr --partial "'replay-missing-source' SLO in '${TEST_PROJECT}-source' Project"
}

@test "replay list returns the platform queue state" {
  run_sloctl replay list -o json

  assert_success
  assert_stderr --partial "Listing all Replays"

  # The shared queue can be empty when the release tests run.
  if [[ -z "$output" ]]; then
    assert_stderr --partial "Replay not found"
    return
  fi

  run jq -e '
    type == "array" and
    all(.[];
      (.slo | type == "string") and
      (.project | type == "string") and
      (.createdAt | type == "string" and ((try fromdateiso8601 catch null) != null)) and
      (.isComposite == null or (.isComposite | type == "boolean")) and
      (.status as $status |
        ["unknown", "queued", "in progress", "completed", "failed", "canceled"] |
        index($status) != null)
    )
  ' <<< "$output"
  assert_success
}

@test "replay cancel uses the platform endpoint" {
  run_sloctl replay cancel replay-missing-slo -p "$TEST_PROJECT"

  assert_failure
  assert_stderr --partial "endpoint: POST https://"
  assert_stderr --partial "/api/timetravel/cancel"
}

@test "replay delete uses the platform endpoint" {
  run_sloctl replay delete replay-missing-slo -p "$TEST_PROJECT"

  assert_failure
  assert_stderr --partial "endpoint: DELETE https://"
  assert_stderr --partial "/api/timetravel"
}
