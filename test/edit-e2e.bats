#!/usr/bin/env bash
# bats file_tags=e2e

# setup_file is run only once for the whole file.
setup_file() {
  load "test_helper/load"
  load_lib "bats-assert"

  generate_inputs "$BATS_FILE_TMPDIR"

  run_sloctl apply -f "'$TEST_INPUTS/**'"
  assert_success_joined_output
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

@test "sloctl edit services exits when editor leaves file unchanged" {
  SLOCTL_EDITOR=true run_sloctl edit services edit-target -p "$TEST_PROJECT"

  assert_success_joined_output
  assert_output "Edit canceled, no changes made."
}

@test "sloctl edit services without names exits when editor leaves file unchanged" {
  editor_script="$(copy_editor_script "services-selection.sh")"

  SLOCTL_EDITOR="$editor_script" run_sloctl edit services -p "$TEST_PROJECT"

  assert_success_joined_output
  assert_output "Edit canceled, no changes made."
}

@test "sloctl edit projects persists editor changes" {
  test_edit_persists_description "projects" "$TEST_PROJECT" ""
}

@test "sloctl edit projects reopens editor after apply validation error" {
  editor_script="$(copy_editor_script "apply-error-recovering.sh")"

  SLOCTL_EDITOR="$editor_script" run_sloctl edit projects "$TEST_PROJECT"

  assert_success_joined_output
  assert_output "The resources were successfully applied."

  run_sloctl get projects "$TEST_PROJECT" -o json
  assert_success_joined_output

  actual="$(yq -r '.[0].spec.description' <<< "$output")"
  assert_equal "$actual" "Recovered after apply error"
}

@test "sloctl edit projects cancels when apply error is reverted" {
  run_sloctl get projects "$TEST_PROJECT" -o json
  assert_success_joined_output
  before="$(yq -r '.[0].spec.description // ""' <<< "$output")"

  editor_script="$(copy_editor_script "apply-error-reverting.sh")"

  SLOCTL_EDITOR="$editor_script" run_sloctl edit projects "$TEST_PROJECT"

  assert_success_joined_output
  assert_output "Edit canceled, no changes made."

  run_sloctl get projects "$TEST_PROJECT" -o json
  assert_success_joined_output
  after="$(yq -r '.[0].spec.description // ""' <<< "$output")"

  assert_equal "$after" "$before"
}

@test "sloctl edit services persists editor changes" {
  test_edit_persists_description "services" "edit-target" "$TEST_PROJECT"
}

@test "sloctl edit agents persists editor changes" {
  test_edit_persists_display_name "agents" "edit-agent" "$TEST_PROJECT"
}

@test "sloctl edit agents rejects multiple names" {
  SLOCTL_EDITOR=true run_sloctl edit agents edit-agent edit-agent-secondary -p "$TEST_PROJECT"

  assert_failure
  assert_stderr "Error: edit agents command accepts only a single Agent"
}

@test "sloctl edit alertpolicies persists editor changes" {
  test_edit_persists_description "alertpolicies" "edit-alert-policy" "$TEST_PROJECT"
}

@test "sloctl edit alertsilences persists editor changes" {
  test_edit_persists_description "alertsilences" "edit-alert-silence" "$TEST_PROJECT"
}

@test "sloctl edit alertmethods persists editor changes" {
  test_edit_persists_description "alertmethods" "edit-alert-method" "$TEST_PROJECT"
}

@test "sloctl edit directs persists editor changes" {
  test_edit_persists_description "directs" "edit-direct" "$TEST_PROJECT"
}

@test "sloctl edit slos persists editor changes" {
  test_edit_persists_description "slos" "edit-slo" "$TEST_PROJECT"
}

@test "sloctl edit rolebindings persists editor changes" {
  test_edit_persists_role_binding "rolebindings" "edit-role-binding" "$TEST_PROJECT"
}

@test "sloctl edit annotations persists editor changes" {
  test_edit_persists_description "annotations" "edit-annotation" "$TEST_PROJECT"
}

@test "sloctl edit budgetadjustments exits when editor leaves file unchanged" {
  SLOCTL_EDITOR=true run_sloctl edit budgetadjustments edit-budget-adjustment

  assert_success_joined_output
  assert_output "Edit canceled, no changes made."
}

@test "sloctl edit budgetadjustments persists editor changes" {
  test_edit_persists_description "budgetadjustments" "edit-budget-adjustment" ""
}

@test "sloctl edit reports exits when editor leaves file unchanged" {
  SLOCTL_EDITOR=true run_sloctl edit reports edit-report

  assert_success_joined_output
  assert_output "Edit canceled, no changes made."
}

@test "sloctl edit reports persists editor changes" {
  test_edit_persists_display_name "reports" "edit-report" ""
}

@test "sloctl edit dataexports subcommand reports no resources" {
  SLOCTL_EDITOR=true run_sloctl edit dataexports missing-data-export -p "$TEST_PROJECT"

  assert_success_joined_output
  assert_output "No resources found in '$TEST_PROJECT' project."
}

@test "sloctl edit services supports all-projects shorthand" {
  SLOCTL_EDITOR=true run_sloctl edit services "missing-service-$TEST_PROJECT" -A

  assert_success_joined_output
  assert_output "No resources found in '*' project."
}

@test "sloctl edit services supports label filter" {
  SLOCTL_EDITOR=true run_sloctl edit services -l edit-filter=primary -p "$TEST_PROJECT"

  assert_success_joined_output
  assert_output "Edit canceled, no changes made."
}

@test "sloctl edit slos supports service filter" {
  SLOCTL_EDITOR=true run_sloctl edit slos -s edit-target -p "$TEST_PROJECT"

  assert_success_joined_output
  assert_output "Edit canceled, no changes made."
}

@test "sloctl edit budgetadjustments supports slo and project filters" {
  SLOCTL_EDITOR=true run_sloctl edit budgetadjustments --slo edit-slo -p "$TEST_PROJECT"

  assert_success_joined_output
  assert_output "Edit canceled, no changes made."
}

@test "sloctl edit services reports editor failure and preserves file" {
  editor_script="$(copy_editor_script "failing.sh")"

  SLOCTL_EDITOR="$editor_script" run_sloctl edit services edit-target -p "$TEST_PROJECT"

  assert_failure
  assert_stderr --partial "Error: failed to run editor \"$editor_script\": exit status 23"
  assert_stderr --partial "A copy of your changes has been stored to"
}

@test "sloctl edit services reports invalid resource identity and preserves file" {
  editor_script="$(copy_editor_script "invalid-identity.sh")"

  SLOCTL_EDITOR="$editor_script" run_sloctl edit services edit-target -p "$TEST_PROJECT"

  assert_failure
  assert_stderr --partial "A copy of your changes has been stored to"
  assert_stderr --partial "error: Edit cancelled, no valid changes were saved."

  edited_file="$(extract_preserved_edit_file_path "$stderr")"
  assert [ -n "$edited_file" ]
  assert [ -f "$edited_file" ]
  assert_file_contains "$edited_file" \
    "# The edited file had an error: edited resources must match the selected resources; changing kind, name, or project is not supported"
}

test_edit_persists_description() {
  local editor_script
  editor_script="$(copy_editor_script "description.sh")"

  test_edit_persists "$1" "$2" "$3" "$editor_script" '.[0].spec.description' "Edited by sloctl edit e2e"
}

test_edit_persists_display_name() {
  local editor_script
  editor_script="$(copy_editor_script "display-name.sh")"

  test_edit_persists "$1" "$2" "$3" "$editor_script" '.[0].metadata.displayName' "Edited by sloctl edit e2e"
}

test_edit_persists_role_binding() {
  local editor_script
  editor_script="$(copy_editor_script "role-binding.sh")"

  test_edit_persists "$1" "$2" "$3" "$editor_script" '.[0].spec.roleRef' "project-editor"
}

test_edit_persists() {
  local \
    subcommand="$1" \
    name="$2" \
    project="$3" \
    editor_script="$4" \
    assertion_filter="$5" \
    expected="$6"
  local args=(edit "$subcommand" "$name")
  if [[ -n "$project" ]]; then
    args+=(-p "$project")
  fi

  SLOCTL_EDITOR="$editor_script" run_sloctl "${args[@]}"
  assert_success_joined_output
  assert_output "The resources were successfully applied."

  args=(get "$subcommand" "$name" -o json)
  if [[ -n "$project" ]]; then
    args+=(-p "$project")
  fi

  run_sloctl "${args[@]}"
  assert_success_joined_output

  actual="$(yq -r "$assertion_filter" <<< "$output")"
  assert_equal "$actual" "$expected"
}

extract_preserved_edit_file_path() {
  sed -n 's/.*A copy of your changes has been stored to "\([^"]*\)".*/\1/p' <<< "$1"
}

assert_file_contains() {
  local file="$1"
  local expected="$2"
  local contents
  contents="$(< "$file")"

  if [[ "$contents" != *"$expected"* ]]; then
    fail "Expected $file to contain: $expected"
  fi
}

# copy_editor_script copies a fixture editor into the per-test temp directory so
# stateful editor scripts can persist retry state without writing into the repo.
copy_editor_script() {
  local fixture_name="$1"

  local timestamp
  timestamp="$(date -u +%Y%m%dT%H%M%SZ)"

  local editor_script
  editor_script="$BATS_TEST_TMPDIR/sloctl-edit-${fixture_name%.sh}-editor-$timestamp.sh"

  cp "test/inputs/edit-e2e/editors/$fixture_name" "$editor_script"
  chmod +x "$editor_script"

  printf '%s\n' "$editor_script"
}
