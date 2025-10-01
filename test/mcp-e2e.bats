#!/usr/bin/env bash
# bats file_tags=e2e

# setup_file is run only once for the whole file.
setup_file() {
  load "test_helper/load"
  load_lib "bats-assert"

  generate_inputs "$BATS_FILE_TMPDIR"
  generate_outputs
}

# teardown_file is run only once for the whole file.
teardown_file() {
  # Clean up any applied resources from the tests.
  if [[ -n "$TEST_INPUTS" ]]; then
    run_sloctl delete -f "'$TEST_INPUTS/**'" 2>/dev/null || true
  fi
}

# setup is run before each test.
setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"
}

@test "apply Project" {
  input_file="$TEST_INPUTS/project.yaml"

  # First apply the project using MCP apply tool.
  run_mcp_inspector \
    --method tools/call \
    --tool-name apply \
    --tool-arg file_name="$input_file"
  assert_success
  assert_output --partial "successfully applied"

  # Verify the project was created by getting it directly with sloctl.
  run_sloctl get project "$TEST_PROJECT"
  assert_success_joined_output
  assert_applied "$(read_files "${TEST_OUTPUTS}/project.yaml")"
}

@test "get Project after apply" {
  input_file="$TEST_INPUTS/project.yaml"

  # First apply the project using regular sloctl to ensure it exists.
  run_sloctl apply -f "$input_file"
  assert_success

  # Now get the project using MCP get tool.
  run_mcp_inspector \
    --method tools/call \
    --tool-name get_projects \
    --tool-arg name="$TEST_PROJECT" \
    --tool-arg format=yaml
  assert_success
  assert_output --regexp "Retrieved 1 Projects. Output written to: .*.yaml"
  assert_output --partial "$TEST_PROJECT"
}

@test "missing argument to apply" {
  # Try to apply a non-existent file.
  run_mcp_inspector \
    --method tools/call \
    --tool-name apply
  assert_failure
  assert_output --partial "'file_name' argument is required"
}

@test "apply non-existent file returns error" {
  # Try to apply a non-existent file.
  run_mcp_inspector \
    --method tools/call \
    --tool-name apply \
    --tool-arg file_name="/tmp/non-existent-file.yaml"
  assert_failure
  assert_output --partial "/tmp/non-existent-file.yaml: no such file or directory"
}

@test "get non-existent Project returns error" {
  # Try to get a non-existent project.
  run_mcp_inspector \
    --method tools/call \
    --tool-name get_projects \
    --tool-arg name="non-existent-project-12345"
  assert_success
  assert_equal \
    "$(jq -r .content[0].text <<<"$output")" \
    "Found no Projects"
}
