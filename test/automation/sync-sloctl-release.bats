#!/usr/bin/env bats
# bats file_tags=unit

setup() {
  bats_require_minimum_version 1.7.0
  bats_load_library bats-support
  bats_load_library bats-assert
  export FAKE_STATE="$BATS_TEST_TMPDIR" FAKE_MODE=new
  export PATH="$BATS_TEST_DIRNAME/helpers:$PATH"
  export TARGET_REPOSITORY=nobl9/nobl9-action TARGET_WORKFLOW=update-sloctl.yml TARGET_REF=main
  export EXPECTED_RUN_NAME="sloctl action from release run 42"
  export SLOCTL_TAG=v0.27.0 SLOCTL_SHA=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa SOURCE_RUN_ID=42
  script="$BATS_TEST_DIRNAME/../../.github/actions/sync-sloctl-release/sync-sloctl-release.sh"
}

@test "dispatch passes the configured target and exact release identity" {
  run bash "$script"
  assert_success
  assert_equal "$(jq -c . "$FAKE_STATE/dispatched")" \
    '["workflow","run","update-sloctl.yml","--repo","nobl9/nobl9-action","--ref","main","--field","tag=v0.27.0","--field","sha=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","--field","source_run_id=42"]'
}

@test "the same dispatcher supports the documentation workflow and production branch" {
  export TARGET_REPOSITORY=nobl9/example-docs TARGET_WORKFLOW=update-sloctl-command-reference.yml TARGET_REF=production
  run bash "$script"
  assert_success
  assert_equal "$(jq -r '.[2], .[4], .[6]' "$FAKE_STATE/dispatched")" \
    $'update-sloctl-command-reference.yml\nnobl9/example-docs\nproduction'
}

@test "a completed downstream update is not dispatched again" {
  export FAKE_MODE=existing-success
  run bash "$script"
  assert_success
  refute [ -f "$FAKE_STATE/dispatched" ]
  assert_equal "$(cat "$FAKE_STATE/calls")" "run list"
}

@test "an active downstream update resumes monitoring" {
  export FAKE_MODE=existing-active
  run bash "$script"
  assert_success
  refute [ -f "$FAKE_STATE/dispatched" ]
  assert_equal "$(cat "$FAKE_STATE/calls")" $'run list\nrun watch'
}

@test "a failed downstream update dispatches a new attempt" {
  export FAKE_MODE=existing-failed
  run bash "$script"
  assert_success
  assert [ -f "$FAKE_STATE/dispatched" ]
}

@test "a failed downstream workflow fails synchronization" {
  export FAKE_MODE=watch-failed
  run bash "$script"
  assert_failure
  assert_output --partial "completed with failure"
}

@test "a lost watch response is reconciled with the completed run" {
  export FAKE_MODE=lost-watch-response
  run bash "$script"
  assert_success
  refute [ -f "$FAKE_STATE/dispatched" ]
}

@test "a matching run title on another branch cannot satisfy synchronization" {
  export FAKE_MODE=wrong-branch
  run bash "$script"
  assert_success
  assert_equal "$(cat "$FAKE_STATE/calls")" $'run list\nrun watch'
}

@test "the action receiver is left running when monitoring times out" {
  export FAKE_MODE=monitor-timeout CANCEL_ON_TIMEOUT=false
  run bash "$script"
  assert_failure
  refute [ -f "$FAKE_STATE/cancelled" ]
  assert_output --partial "Leaving downstream run 99 active"
}
