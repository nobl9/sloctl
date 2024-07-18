#!/usr/bin/env bash
# bats file_tags=e2e

# setup_file is run only once for the whole file.
setup_file() {
  load "test_helper/load"
  load_lib "bats-assert"

  generate_inputs "$BATS_FILE_TMPDIR"
  run_sloctl apply -f "'$TEST_INPUTS/**'"
  assert_success
}

# teardown_file is run only once for the whole file.
teardown_file() {
  run_sloctl delete -f "'$TEST_INPUTS/**'"
}

# setup is run before each test.
setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"
}

# -------------------------------------------------------------
# | The order of execution matters here!                      |
# | To make the test faster, we're deleting objects ony by    |
# | one, but we cannot simply delete Project before resources |
# | tied to it are deleted.                                   |
# -------------------------------------------------------------

@test "alert silence" {
  test_delete_by_name "AlertSilence" "${TEST_INPUTS}/alertsilence.yaml"
}

@test "annotation" {
  test_delete_by_name "Annotation" "${TEST_INPUTS}/annotation.yaml"
}

@test "slo" {
  test_delete_by_name "SLO" "${TEST_INPUTS}/slo.yaml"
}

@test "alert policy" {
  test_delete_by_name "AlertPolicy" "${TEST_INPUTS}/alertpolicy.yaml"
}

@test "alert method" {
  test_delete_by_name "AlertMethod" "${TEST_INPUTS}/alertmethod.yaml"
}

@test "agent" {
  test_delete_by_name "Agent" "${TEST_INPUTS}/agent.yaml"
}

@test "direct" {
  test_delete_by_name "Direct" "${TEST_INPUTS}/direct.yaml"
}

@test "service" {
  test_delete_by_name "Service" "${TEST_INPUTS}/service.yaml"
}

@test "role binding" {
  test_delete_by_name "RoleBinding" "${TEST_INPUTS}/rolebinding.yaml"
}

@test "project" {
  test_delete_by_name "Project" "${TEST_INPUTS}/project.yaml"
}

@test "budget adjustment" {
  test_delete_by_name "BudgetAdjustment" "${TEST_INPUTS}/budgetadjustment.yaml"
}

# Currently we cannot apply user groups and DataExport has very strict
# org limits making it impossible to test with applied objects.
#
# @test "data export" {
# 	test_delete_by_name "DataExport" ""
# }
#
# @test "user group" {
# 	test_delete_by_name "UserGroup" ""
# }

test_delete_by_name() {
  local \
    kind="$1" \
    input="$2"
  object_name=$(yq -r .metadata.name "$input")

  # Ensure delete by name without a name doesn't pass.
  run_sloctl delete "$kind"
  assert_failure
  output="$stderr"
  assert_output "Error: requires at least 1 arg(s), only received 0"

  # Delete the object by name.
  args=(delete "$kind" "$object_name")
  if [[ $kind != "Project" ]]; then
    args+=("-p" "death-star")
  fi
  run_sloctl "${args[@]}"
  assert_success
  assert_deleted "$(read_files "$input")"
}
