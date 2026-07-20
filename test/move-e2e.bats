#!/usr/bin/env bash
# bats file_tags=e2e

# setup_file is run only once for the whole file.
setup_file() {
  load "test_helper/load"
  load_lib "bats-assert"

  generate_inputs "$BATS_FILE_TMPDIR"
  generate_outputs

  run_sloctl apply -f "'$TEST_INPUTS/**'"
  assert_success_joined_output
}

# teardown_file is run only once for the whole file.
teardown_file() {
  run_sloctl delete -f "'$TEST_INPUTS/**'"
  run_sloctl delete -f "'$TEST_OUTPUTS/**'"
  assert_success_joined_output
}

# setup is run before each test.
setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"
}

@test "default source Project" {
  SLOCTL_PROJECT="$TEST_PROJECT" run_sloctl move slo default-project --to-project="${TEST_PROJECT}-new"

  assert_success_joined_output
  assert_stderr - <<EOF
Moving 'default-project' SLO from '$TEST_PROJECT' Project to '${TEST_PROJECT}-new' Project.
If the target Service in the new Project does not exist, it will be created.

The SLOs were successfully moved.
EOF

  assert_applied "$(read_files "${TEST_OUTPUTS}/default-project.yaml")"
}

@test "custom source Project" {
  run_sloctl move slo custom-project -p "$TEST_PROJECT" --to-project="${TEST_PROJECT}-new"

  assert_success_joined_output
  assert_stderr - <<EOF
Moving 'custom-project' SLO from '$TEST_PROJECT' Project to '${TEST_PROJECT}-new' Project.
If the target Service in the new Project does not exist, it will be created.

The SLOs were successfully moved.
EOF

  assert_applied "$(read_files "${TEST_OUTPUTS}/custom-project.yaml")"
}

@test "move multiple slos" {
  run_sloctl move slo move-multiple-slos-1 move-multiple-slos-2 -p "$TEST_PROJECT" --to-project="${TEST_PROJECT}-new"

  assert_success_joined_output
  assert_stderr - <<EOF
Moving the following SLOs from '$TEST_PROJECT' Project to '${TEST_PROJECT}-new' Project:
 - move-multiple-slos-1
 - move-multiple-slos-2
If the target Service in the new Project does not exist, it will be created.

The SLOs were successfully moved.
EOF

  assert_applied "$(read_files "${TEST_OUTPUTS}/move-multiple-slos.yaml")"
}

@test "move all slos from Project" {
  run_sloctl move slo -p "${TEST_PROJECT}-all" --to-project="${TEST_PROJECT}-new"

  assert_success_joined_output
  assert_stderr - <<EOF
Fetching all SLOs from '${TEST_PROJECT}-all' Project...
Moving the following SLOs from '${TEST_PROJECT}-all' Project to '${TEST_PROJECT}-new' Project:
 - move-all-slos-1
 - move-all-slos-2
If the target Service in the new Project does not exist, it will be created.

The SLOs were successfully moved.
EOF

  assert_applied "$(read_files "${TEST_OUTPUTS}/move-all-slos.yaml")"
}

@test "custom target Service" {
  run_sloctl move slo custom-target-service -p "$TEST_PROJECT" --to-service="custom-target-service" --to-project="${TEST_PROJECT}-new"

  assert_success_joined_output
  assert_stderr - <<EOF
Moving 'custom-target-service' SLO from '$TEST_PROJECT' Project to '${TEST_PROJECT}-new' Project.
'custom-target-service' Service in '${TEST_PROJECT}-new' Project will be assigned to all the moved SLOs.
If the target Service in the new Project does not exist, it will be created.

The SLOs were successfully moved.
EOF

  assert_applied "$(read_files "${TEST_OUTPUTS}/custom-target-service.yaml")"
}

@test "same-project service move" {
  run_sloctl move slo same-project-service-move -p "$TEST_PROJECT" --to-service="new-service"

  assert_success_joined_output
  assert_stderr - <<EOF
Moving 'same-project-service-move' SLO to a different Service within '$TEST_PROJECT' Project.
'new-service' Service in '$TEST_PROJECT' Project will be assigned to all the moved SLOs.
If the target Service does not exist in this Project, it will be created.

The SLOs were successfully moved.
EOF

  assert_applied "$(read_files "${TEST_OUTPUTS}/same-project-service-move.yaml")"
}

@test "same-project service move with multiple slos" {
  run_sloctl move slo same-project-multi-1 same-project-multi-2 -p "$TEST_PROJECT" --to-service="new-service"

  assert_success_joined_output
  assert_stderr - <<EOF
Moving the following SLOs to a different Service within '$TEST_PROJECT' Project:
 - same-project-multi-1
 - same-project-multi-2
'new-service' Service in '$TEST_PROJECT' Project will be assigned to all the moved SLOs.
If the target Service does not exist in this Project, it will be created.

The SLOs were successfully moved.
EOF

  assert_applied "$(read_files "${TEST_OUTPUTS}/same-project-multi.yaml")"
}

@test "error for attached Alert Policies" {
  run_sloctl move slo detach-alert-policies -p "$TEST_PROJECT" --to-project="${TEST_PROJECT}-new"

  assert_failure
  assert_stderr - <<EOF
Moving 'detach-alert-policies' SLO from '$TEST_PROJECT' Project to '${TEST_PROJECT}-new' Project.
If the target Service in the new Project does not exist, it will be created.

Error: Cannot move SLOs with attached Alert Policies.
Detach them manually or use the '--detach-alert-policies' flag to detach them automatically.
EOF
}

@test "detach Alert Policies" {
  run_sloctl move slo detach-alert-policies -p "$TEST_PROJECT" --detach-alert-policies --to-project="${TEST_PROJECT}-new"

  assert_success_joined_output
  assert_stderr - <<EOF
Moving 'detach-alert-policies' SLO from '$TEST_PROJECT' Project to '${TEST_PROJECT}-new' Project.
If the target Service in the new Project does not exist, it will be created.
Attached Alert Policies will be detached from all the moved SLOs.

The SLOs were successfully moved.
EOF

  assert_applied "$(read_files "${TEST_OUTPUTS}/detach-alert-policies.yaml")"
}

@test "no SLOs in source Project" {
  FAKE_PROJECT="made-up-project-x-y-z"
  SLOCTL_PROJECT="$TEST_PROJECT" run_sloctl move slo -p "$FAKE_PROJECT" --to-project="${TEST_PROJECT}-new"

  assert_failure
  assert_stderr - <<EOF
Fetching all SLOs from '$FAKE_PROJECT' Project...
Error: Found no SLOs in '$FAKE_PROJECT' Project.
EOF
}
