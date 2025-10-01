#!/usr/bin/env bash
# bats file_tags=unit

# setup_file is run only once for the whole file.
setup_file() {
  export SLOCTL_CLIENT_ID=id
  export SLOCTL_CLIENT_SECRET=secret
}

# setup is run before each test.
setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"
}

@test "list tools" {
  run_sloctl mcp list -o json --jq .tools
  assert_success_joined_output
  sloctl_output="$output"

  run_mcp_inspector --method tools/list
  assert_success
  mcp_output="$output"

  run diff <(echo "$mcp_output" | jq -S .tools) <(echo "$sloctl_output" | jq . -S)
}

@test "list resources" {
  run_sloctl mcp list -o json --jq .resources
  assert_success_joined_output
  sloctl_output="$output"

  run_mcp_inspector --method resources/list
  assert_success
  mcp_output="$output"

  run diff <(echo "$mcp_output" | jq -S .resources) <(echo "$sloctl_output" | jq . -S)
}
