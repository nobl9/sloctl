#!/usr/bin/env bash
# bats file_tags=e2e,bats:focus

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

@test "get SLO after apply" {
  service_file="$TEST_INPUTS/service.yaml"
  slo_file="$TEST_INPUTS/slo.yaml"

  # First apply the service and SLO using regular sloctl to ensure they exist.
  run_sloctl apply -f "$service_file"
  assert_success

  run_sloctl apply -f "$slo_file"
  assert_success

  # Now get the SLO using MCP getSLO tool.
  run_mcp_inspector \
    --method tools/call \
    --tool-name getSLO \
    --tool-arg name=test-mcp-slo \
    --tool-arg project="$TEST_PROJECT" \
    --tool-arg format=json
  assert_success

  # Verify the response contains the SLO
  slo_name=$(jq -r '.content[0].text | fromjson | .metadata.name' <<<"$output")
  assert_equal "$slo_name" "test-mcp-slo"

  slo_project=$(jq -r '.content[0].text | fromjson | .metadata.project' <<<"$output")
  assert_equal "$slo_project" "$TEST_PROJECT"
}
