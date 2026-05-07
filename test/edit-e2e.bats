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

test_edit_persists_description() {
  local editor_script
  editor_script="$(create_description_editor)"

  test_edit_persists "$1" "$2" "$3" "$editor_script" '.[0].spec.description' "Edited by sloctl edit e2e"
}

test_edit_persists_display_name() {
  local editor_script
  editor_script="$(create_display_name_editor)"

  test_edit_persists "$1" "$2" "$3" "$editor_script" '.[0].metadata.displayName' "Edited by sloctl edit e2e"
}

test_edit_persists_role_binding() {
  local editor_script
  editor_script="$(create_role_binding_editor)"

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

# create_description_editor returns an editor wrapper that changes the selected
# resource description without changing its identity.
create_description_editor() {
  create_editor_script "description" << 'EOF'
yq -Y -i '
  if type == "array" then
    .[0].spec.description = "Edited by sloctl edit e2e"
  else
    .spec.description = "Edited by sloctl edit e2e"
  end
' "$1"
EOF
}

# create_display_name_editor returns an editor wrapper that changes the selected
# resource display name without changing its identity.
create_display_name_editor() {
  create_editor_script "display-name" << 'EOF'
yq -Y -i '
  if type == "array" then
    .[0].metadata.displayName = "Edited by sloctl edit e2e"
  else
    .metadata.displayName = "Edited by sloctl edit e2e"
  end
' "$1"
EOF
}

# create_failing_editor returns an editor wrapper that writes to stderr and
# exits non-zero, allowing the test to verify editor failures preserve the
# edited file.
create_failing_editor() {
  create_editor_script "failing" << 'EOF'
printf "%s\n" "editor failed intentionally" >&2
exit 23
EOF
}

# create_role_binding_editor returns an editor wrapper that changes the selected
# RoleBinding role while preserving its kind, name, and project identity.
create_role_binding_editor() {
  create_editor_script "role-binding" << 'EOF'
yq -Y -i '
  if type == "array" then
    .[0].spec.roleRef = "project-editor" |
    del(.[0].spec.user)
  else
    .spec.roleRef = "project-editor" |
    del(.spec.user)
  end
' "$1"
EOF
}

# create_invalid_identity_editor returns an editor wrapper that changes the
# selected resource name only once. 'sloctl edit' reopens the editor after
# validation fails, and the second no-op run triggers the invalid-change
# cancellation path while leaving the preserved file available for assertions.
create_invalid_identity_editor() {
  create_editor_script "invalid-identity" << 'EOF'
state_file="${0}.state"

if [[ ! -e "$state_file" ]]; then
  touch "$state_file"
  yq -Y -i '
    if type == "array" then
      .[0].metadata.name = "renamed-edit-target"
    else
      .metadata.name = "renamed-edit-target"
    end
  ' "$1"
fi
EOF
}

create_services_selection_asserting_editor() {
  create_editor_script "services-selection" << 'EOF'
actual_names="$(yq -r '[if type == "array" then .[] else . end | select(.kind == "Service") | .metadata.name] | sort | join(" ")' "$1")"
expected_names="edit-target edit-target-secondary"

if [[ "$actual_names" != "$expected_names" ]]; then
  printf "expected edited services [%s], got [%s]\n" "$expected_names" "$actual_names" >&2
  exit 24
fi
EOF
}

# create_editor_script writes an executable wrapper that can be passed as
# SLOCTL_EDITOR. The edit command appends the edited file path to the editor
# command, so tests pass the wrapper body through stdin and use the returned path
# as the editor command.
create_editor_script() {
  local name="$1"

  local timestamp
  timestamp="$(date -u +%Y%m%dT%H%M%SZ)"

  local editor_script
  editor_script="$BATS_TEST_TMPDIR/sloctl-edit-$name-editor-$timestamp.sh"

  cat > "$editor_script" << 'EOF'
#!/usr/bin/env bash

set -euo pipefail

EOF
  cat >> "$editor_script"
  chmod +x "$editor_script"

  printf '%s\n' "$editor_script"
}

@test "sloctl edit services exits when editor leaves file unchanged" {
  SLOCTL_EDITOR=true run_sloctl edit services edit-target -p "$TEST_PROJECT"

  assert_success_joined_output
  assert_output "Edit canceled, no changes made."
}

@test "sloctl edit services without names exits when editor leaves file unchanged" {
  editor_script="$(create_services_selection_asserting_editor)"

  SLOCTL_EDITOR="$editor_script" run_sloctl edit services -p "$TEST_PROJECT"

  assert_success_joined_output
  assert_output "Edit canceled, no changes made."
}

@test "sloctl edit projects persists editor changes" {
  test_edit_persists_description "projects" "$TEST_PROJECT" ""
}

@test "sloctl edit services persists editor changes" {
  test_edit_persists_description "services" "edit-target" "$TEST_PROJECT"
}

@test "sloctl edit agents persists editor changes" {
  test_edit_persists_display_name "agents" "edit-agent" "$TEST_PROJECT"
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

@test "sloctl edit reports exits when editor leaves file unchanged" {
  SLOCTL_EDITOR=true run_sloctl edit reports edit-report

  assert_success_joined_output
  assert_output "Edit canceled, no changes made."
}

@test "sloctl edit dataexports subcommand reports no resources" {
  SLOCTL_EDITOR=true run_sloctl edit dataexports missing-data-export -p "$TEST_PROJECT"

  assert_success_joined_output
  assert_output "No resources found in '$TEST_PROJECT' project."
}

@test "sloctl edit services reports editor failure and preserves file" {
  editor_script="$(create_failing_editor)"

  SLOCTL_EDITOR="$editor_script" run_sloctl edit services edit-target -p "$TEST_PROJECT"

  assert_failure
  assert_stderr --partial "Error: failed to run editor \"$editor_script\": exit status 23"
  assert_stderr --partial "A copy of your changes has been stored to"
}

@test "sloctl edit services reports invalid resource identity and preserves file" {
  editor_script="$(create_invalid_identity_editor)"

  SLOCTL_EDITOR="$editor_script" run_sloctl edit services edit-target -p "$TEST_PROJECT"

  assert_failure
  assert_stderr --partial "A copy of your changes has been stored to"
  assert_stderr --partial "error: Edit cancelled, no valid changes were saved."

  edited_file="$(extract_preserved_edit_file_path "$stderr")"
  assert [ -n "$edited_file" ]
  assert [ -f "$edited_file" ]
  assert_file_contains "$edited_file" \
    "# The edited file had a syntax error: edited resources must match the selected resources; changing kind, name, or project is not supported"
}
