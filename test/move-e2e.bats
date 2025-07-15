#!/usr/bin/env bash
# bats file_tags=e2e

# setup_file is run only once for the whole file.
setup_file() {
  load "test_helper/load"
  load_lib "bats-assert"

  generate_inputs "$BATS_FILE_TMPDIR"

  export TEST_OUTPUTS="$TEST_SUITE_OUTPUTS/move-e2e"

  # Use generated project name in the outputs too.
  for file in "$TEST_OUTPUTS"/*; do
    run sed -i "s/<PROJECT>/$TEST_PROJECT/g" "$file"
  done

  run_sloctl apply -f "'$TEST_INPUTS/**'"
  assert_success_joined_output
}

# teardown_file is run only once for the whole file.
teardown_file() {
  run_sloctl delete -f "'$TEST_INPUTS/**'" -f "'$TEST_OUTPUTS/**'"
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
  output="$stderr"
  assert_output - <<EOF
Moving 'default-project' SLO from '$TEST_PROJECT' Project to '${TEST_PROJECT}-new' Project.
If the target Service in the new Project does not exist, it will be created.

The SLOs were successfully moved.
EOF

  assert_applied "$(read_files "${TEST_OUTPUTS}/default-project.yaml")"
}

@test "custom source Project" {
  run_sloctl move slo custom-project -p "$TEST_PROJECT" --to-project="${TEST_PROJECT}-new"

  assert_success_joined_output
  output="$stderr"
  assert_output - <<EOF
Moving 'custom-project' SLO from '$TEST_PROJECT' Project to '${TEST_PROJECT}-new' Project.
If the target Service in the new Project does not exist, it will be created.

The SLOs were successfully moved.
EOF

  assert_applied "$(read_files "${TEST_OUTPUTS}/custom-project.yaml")"
}

@test "move multiple slos" {
  run_sloctl move slo move-multiple-slos-1 move-multiple-slos-2 -p "$TEST_PROJECT" --to-project="${TEST_PROJECT}-new"

  assert_success_joined_output
  output="$stderr"
  assert_output - <<EOF
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
  output="$stderr"
  assert_output - <<EOF
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
  output="$stderr"
  assert_output - <<EOF
Moving 'custom-target-service' SLO from '$TEST_PROJECT' Project to '${TEST_PROJECT}-new' Project.
'custom-target-service' Service in '${TEST_PROJECT}-new' Project will be assigned to all the moved SLOs.
If the target Service in the new Project does not exist, it will be created.

The SLOs were successfully moved.
EOF

  assert_applied "$(read_files "${TEST_OUTPUTS}/custom-target-service.yaml")"
}

@test "error for attached Alert Policies" {
  run_sloctl move slo detach-alert-policies -p "$TEST_PROJECT" --to-project="${TEST_PROJECT}-new"

  assert_failure
  output="$stderr"
  assert_output - <<EOF
Moving 'detach-alert-policies' SLO from '$TEST_PROJECT' Project to '${TEST_PROJECT}-new' Project.
If the target Service in the new Project does not exist, it will be created.

Error: Cannot move SLOs with attached Alert Policies.
Detach them manually or use the '--detach-alert-policies' flag to detach them automatically.
EOF
}

@test "detach Alert Policies" {
  run_sloctl move slo detach-alert-policies -p "$TEST_PROJECT" --detach-alert-policies --to-project="${TEST_PROJECT}-new"

  assert_success_joined_output
  output="$stderr"
  assert_output - <<EOF
Moving 'detach-alert-policies' SLO from '$TEST_PROJECT' Project to '${TEST_PROJECT}-new' Project.
If the target Service in the new Project does not exist, it will be created.
Attached Alert Policies will be detached from all the moved SLOs.

The SLOs were successfully moved.
EOF

  assert_applied "$(read_files "${TEST_OUTPUTS}/detach-alert-policies.yaml")"
}
