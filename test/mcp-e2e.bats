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

@test "get Project after apply" {
  input_file="$TEST_INPUTS/project.yaml"

  run_sloctl apply -f "$input_file"
  assert_success

  run_mcp_inspector \
    --method tools/call \
    --tool-name getProject \
    --tool-arg name="$TEST_PROJECT" \
    --tool-arg format=yaml
  assert_success

  actual=$(jq -r '.content[0].text' <<<"$output")
  expected=$(cat "$TEST_OUTPUTS/get-project.yaml")

  assert_equal "$actual" "$expected"
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
  service_file="$TEST_INPUTS/service.yaml"
  slo_file="$TEST_INPUTS/slo.yaml"

  run_sloctl apply -f "$service_file"
  assert_success_joined_output

  run_sloctl apply -f "$slo_file"
  assert_success_joined_output

  run_mcp_inspector \
    --method tools/call \
    --tool-name getSLO \
    --tool-arg name=test-mcp-slo \
    --tool-arg project="$TEST_PROJECT" \
    --tool-arg format=json
  assert_success

  actual=$(jq -S . <<<"$(jq -r '.structuredContent' <<<"$output")")
  expected=$(jq -S . "$TEST_OUTPUTS/get-slo.json")

  assert_equal "$actual" "$expected"
}
