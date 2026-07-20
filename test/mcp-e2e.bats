#!/usr/bin/env bash
# bats file_tags=e2e

# setup_file is run only once for the whole file.
setup_file() {
  load "test_helper/load"
  load_lib "bats-assert"

  generate_inputs "$BATS_FILE_TMPDIR"
  generate_outputs

  # Apply all resources once in setup
  run_sloctl apply -f "'$TEST_INPUTS/**'"
  assert_success_joined_output
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

@test "get Project after apply" {
  # Verify project was applied correctly
  assert_applied "$(read_files "$TEST_INPUTS/project.yaml")"

  # Get the project using MCP
  run_mcp_inspector \
    --method tools/call \
    --tool-name getProject \
    --tool-arg name="$TEST_PROJECT" \
    --tool-arg format=yaml
  assert_success

  # Verify MCP response contains the project
  json_output=$(echo "$output" | sed -n '/{/,$p')
  assert_equal "$(jq -r '.structuredContent.kind' <<<"$json_output")" "Project"
  assert_equal "$(jq -r '.structuredContent.metadata.name' <<<"$json_output")" "$TEST_PROJECT"
}

@test "get non-existent Project returns error" {
  run_mcp_inspector \
    --method tools/call \
    --tool-name getProject \
    --tool-arg name="non-existent-project-12345"
  assert_success
  assert_equal \
    "$(jq -r .content[0].text <<<"$output")" \
    "object was not found"
}

@test "get SLO after apply" {
  # Verify SLO was applied correctly
  assert_applied "$(read_files "$TEST_INPUTS/slo.yaml")"

  # Get the SLO using MCP
  run_mcp_inspector \
    --method tools/call \
    --tool-name getSLO \
    --tool-arg name=test-mcp-slo \
    --tool-arg project="$TEST_PROJECT" \
    --tool-arg format=json
  assert_success

  # Verify MCP response contains the SLO
  json_output=$(echo "$output" | sed -n '/{/,$p')
  assert_equal "$(jq -r '.structuredContent.kind' <<<"$json_output")" "SLO"
  assert_equal "$(jq -r '.structuredContent.metadata.name' <<<"$json_output")" "test-mcp-slo"
  assert_equal "$(jq -r '.structuredContent.metadata.project' <<<"$json_output")" "$TEST_PROJECT"
}
