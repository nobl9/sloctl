#!/usr/bin/env bash
# bats file_tags=e2e

# setup is run before each test.
setup() {
  load "test_helper/load"
  load_lib "bats-assert"

  generate_inputs "$BATS_TEST_TMPDIR"
}

@test "read separate documents from file" {
  input="${TEST_INPUTS}/separate-documents.yaml"

  # apply
  run_sloctl apply -f "$input"
  assert_success_joined_output
  assert_output - <<EOF
Applying 3 objects from the following sources:
 - $input
The resources were successfully applied.
EOF
  assert_applied "$(read_files "$input")"

  # delete
  run_sloctl delete -f "$input"
  assert_success_joined_output
  assert_output - <<EOF
Deleting 3 objects from the following sources:
 - $input
The resources were successfully deleted.
EOF
  assert_deleted "$(read_files "$input")"
}

@test "read list of objects from file" {
  input="${TEST_INPUTS}/list-of-objects.yaml"

  # apply
  run_sloctl apply -f "$input"
  assert_success_joined_output
  assert_output - <<EOF
Applying 3 objects from the following sources:
 - $input
The resources were successfully applied.
EOF
  assert_applied "$(read_files "$input")"

  # delete
  run_sloctl delete -f "$input"
  assert_success_joined_output
  assert_output - <<EOF
Deleting 3 objects from the following sources:
 - $input
The resources were successfully deleted.
EOF
  assert_deleted "$(read_files "$input")"
}

@test "read single object from file" {
  input="${TEST_INPUTS}/single-object.yaml"

  # apply
  run_sloctl apply -f "$input"
  assert_success_joined_output
  assert_output - <<EOF
Applying 1 objects from the following sources:
 - $input
The resources were successfully applied.
EOF
  assert_applied "$(read_files "$input")"

  # delete
  run_sloctl delete -f "$input"
  assert_success_joined_output
  assert_output - <<EOF
Deleting 1 objects from the following sources:
 - $input
The resources were successfully deleted.
EOF
  assert_deleted "$(read_files "$input")"
}

@test "read from stdin" {
  input="${TEST_INPUTS}/single-object.yaml"

  # apply
  run_sloctl apply -f - <"$input"
  assert_success_joined_output
  assert_output "The resources were successfully applied."
  assert_applied "$(read_files "$input")"

  # delete
  run_sloctl delete -f - <"$input"
  assert_success_joined_output
  assert_output "The resources were successfully deleted."
  assert_deleted "$(read_files "$input")"
}

@test "project flag differs from file definition -> error" {
  # These changes won't (or at least shouldn't) take any effect.
  # To make it easier to test the output we use the static names, without the generated hash.
  input="${TEST_SUITE_INPUTS}/$(basename "$BATS_TEST_FILENAME" .bats)/project-flag-differs.yaml"

  project_flag_mismatch_project="kamino"
  # Prefer multiline string over heredoc since this is a one liner, this way we keep it somewhat readable.
  project_flag_mismatch_error="Error: \
The death-star project from the provided object destroyer.death-star \
does not match the project '$project_flag_mismatch_project'. \
You must pass '--project=death-star' to perform this operation or \
allow the Project to be inferred from the object definition."

  # apply
  run_sloctl apply -f "$input" -p "$project_flag_mismatch_project"
  assert_failure
  output="$stderr"
  assert_output "$project_flag_mismatch_error"

  # delete
  run_sloctl apply -f "$input" -p "$project_flag_mismatch_project"
  assert_failure
  output="$stderr"
  assert_output "$project_flag_mismatch_error"
}

@test "read from json file" {
  input="${TEST_INPUTS}/single-object.json"

  # apply
  run_sloctl apply -f "$input"
  assert_success_joined_output
  assert_output - <<EOF
Applying 1 objects from the following sources:
 - $input
The resources were successfully applied.
EOF
  assert_applied "$(read_files "$input")"

  # delete
  run_sloctl delete -f "$input"
  assert_success_joined_output
  assert_output - <<EOF
Deleting 1 objects from the following sources:
 - $input
The resources were successfully deleted.
EOF
  assert_deleted "$(read_files "$input")"
}

@test "read from multiple sources" {
  inputs_base="$TEST_INPUTS/recursive"
  inputs=(
    "$inputs_base/first-level.yaml"
    "$inputs_base/nested/nested/third-level.json"
    "$inputs_base/nested/second-level.yml"
  )

  # apply
  run_sloctl apply -f "${inputs[0]}" -f "${inputs[1]}" -f "${inputs[2]}"
  assert_success_joined_output
  assert_output - <<EOF
Applying 4 objects from the following sources:
 - ${inputs[0]}
 - ${inputs[1]}
 - ${inputs[2]}
The resources were successfully applied.
EOF
  assert_applied "$(read_files "${inputs[@]}")"

  # delete
  run_sloctl delete -f "${inputs[0]}" -f "${inputs[1]}" -f "${inputs[2]}"
  assert_success_joined_output
  assert_output - <<EOF
Deleting 4 objects from the following sources:
 - ${inputs[0]}
 - ${inputs[1]}
 - ${inputs[2]}
The resources were successfully deleted.
EOF
  assert_deleted "$(read_files "${inputs[@]}")"
}

@test "recursive directory read with **" {
  inputs_base="$TEST_INPUTS/recursive"
  inputs=(
    "$inputs_base/first-level.yaml"
    "$inputs_base/nested/nested/third-level.json"
    "$inputs_base/nested/second-level.yml"
  )

  # apply
  run_sloctl apply -f "'$inputs_base/**'"
  assert_success_joined_output
  assert_output - <<EOF
Applying 4 objects from the following sources:
 - ${inputs[0]}
 - ${inputs[1]}
 - ${inputs[2]}
The resources were successfully applied.
EOF
  assert_applied "$(read_files "${inputs[@]}")"

  # delete
  run_sloctl delete -f "'$inputs_base/**'"
  assert_success_joined_output
  assert_output - <<EOF
Deleting 4 objects from the following sources:
 - ${inputs[0]}
 - ${inputs[1]}
 - ${inputs[2]}
The resources were successfully deleted.
EOF
  assert_deleted "$(read_files "${inputs[@]}")"
}
