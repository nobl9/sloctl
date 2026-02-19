#!/usr/bin/env bash
# bats file_tags=unit

# setup_file is run only once for the whole file.
setup_file() {
  export TEST_INPUTS="$TEST_SUITE_INPUTS/recipes"
  export TEST_OUTPUTS="$TEST_SUITE_OUTPUTS/recipes"
  export SLOCTL_RECIPES_PATH="$BATS_FILE_TMPDIR/sloctl-recipes.yaml"
  cp "$TEST_INPUTS/sloctl-recipes.yaml" "$SLOCTL_RECIPES_PATH"
}

# setup is run before each test.
setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"

  # Minimal TUI mode
  export SLOCTL_ACCESSIBLE_MODE=1
  export NO_COLOR=1
}

@test "sloctl recipes list (yaml)" {
  run_sloctl recipes list
  assert_success_joined_output
  assert_output <"$TEST_OUTPUTS/list.yaml"
}

@test "sloctl recipes list (json)" {
  run_sloctl recipes list -o json
  assert_success_joined_output
  assert_output <"$TEST_OUTPUTS/list.json"
}

@test "sloctl recipes remove" {
  run_sloctl recipes remove find-by-label
  assert_success

  run_sloctl recipes list
  assert_success_joined_output
  assert_output <"$TEST_OUTPUTS/list-after-remove.yaml"

  # Restore for other tests
  cp "$TEST_INPUTS/sloctl-recipes.yaml" "$SLOCTL_RECIPES_PATH"
}

@test "sloctl recipes remove (multiple)" {
  run_sloctl recipes remove find-by-label count-services
  assert_success

  run_sloctl recipes list -o json
  assert_success_joined_output
  assert_output '{"list-all-slos":{"args":["get","slo","-A"],"description":"List all SLOs"}}'

  # Restore for other tests
  cp "$TEST_INPUTS/sloctl-recipes.yaml" "$SLOCTL_RECIPES_PATH"
}

@test "sloctl recipes remove (non-existent)" {
  run_sloctl recipes remove non-existent-recipe
  assert_success

  # Verify nothing was removed
  run_sloctl recipes list -o json
  assert_success_joined_output
  assert_output <"$TEST_OUTPUTS/list.json"
}

@test "sloctl recipes find-by-label (missing required args)" {
  run_sloctl recipes find-by-label
  assert_failure
  assert_stderr --partial "Expected at least 2 arg(s), received 0"
  assert_stderr --partial "required arg(s): [kind label]"
}

@test "sloctl recipes find-by-label (partial args)" {
  run_sloctl recipes find-by-label slo
  assert_failure
  assert_stderr --partial "Expected at least 2 arg(s), received 1"
  assert_stderr --partial "required arg(s): [kind label]"
}

@test "sloctl recipes with empty config" {
  echo "{}" > "$SLOCTL_RECIPES_PATH"

  run_sloctl recipes list -o json
  assert_success_joined_output
  assert_output "{}"

  # Restore for other tests
  cp "$TEST_INPUTS/sloctl-recipes.yaml" "$SLOCTL_RECIPES_PATH"
}

@test "sloctl recipes with invalid yaml" {
  echo "invalid: yaml: content:" > "$SLOCTL_RECIPES_PATH"

  run_sloctl recipes list
  assert_failure
  assert_stderr --partial "failed to decode sloctl recipes config"

  # Restore for other tests
  cp "$TEST_INPUTS/sloctl-recipes.yaml" "$SLOCTL_RECIPES_PATH"
}

@test "sloctl recipes with missing config file" {
  export SLOCTL_RECIPES_PATH="$BATS_FILE_TMPDIR/nonexistent.yaml"

  run_sloctl recipes list
  assert_failure
  assert_stderr --partial "failed to read sloctl recipes"
}

@test "sloctl recipes unknown-recipe" {
  run_sloctl recipes unknown-recipe
  assert_failure
  assert_stderr --partial "unknown recipe: unknown-recipe"
}
