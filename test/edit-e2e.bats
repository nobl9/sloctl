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

@test "sloctl edit service exits when editor leaves file unchanged" {
  SLOCTL_EDITOR=true run_sloctl edit service edit-target -p "$TEST_PROJECT"

  assert_success_joined_output
  assert_output "Edit canceled, no changes made."
}

@test "sloctl edit service persists editor changes" {
  local editor_script
  editor_script="$(create_edit_service_editor)"

  SLOCTL_EDITOR="$editor_script" run_sloctl edit service edit-target -p "$TEST_PROJECT"

  assert_success_joined_output
  assert_output "The resources were successfully applied."

  run_sloctl get service edit-target -p "$TEST_PROJECT" -o json
  assert_success_joined_output

  local edited_label
  edited_label="$(yq -r '.[0].metadata.labels.edited[0]' <<<"$output")"
  assert_equal "$edited_label" "true"
}

@test "sloctl edit service reports editor failure and preserves file" {
  local editor_script
  editor_script="$(create_failing_editor)"

  SLOCTL_EDITOR="$editor_script" run_sloctl edit service edit-target -p "$TEST_PROJECT"

  assert_failure
  assert_stderr --partial "Error: failed to run editor \"$editor_script\": exit status 23"
  assert_stderr --partial "A copy of your changes has been stored to"
}

@test "sloctl edit service reports invalid resource identity and preserves file" {
  local editor_script
  editor_script="$(create_invalid_identity_editor)"

  SLOCTL_EDITOR="$editor_script" run_sloctl edit service edit-target -p "$TEST_PROJECT"

  assert_failure
  assert_stderr --partial "A copy of your changes has been stored to"
  assert_stderr --partial "error: Edit cancelled, no valid changes were saved."

  local edited_file
  edited_file="$(extract_preserved_edit_file_path "$stderr")"
  assert [ -n "$edited_file" ]
  assert [ -f "$edited_file" ]
  assert_file_contains "$edited_file" \
    "# The edited file had a syntax error: edited resources must match the selected resources; changing kind, name, or project is not supported"
}

create_edit_service_editor() {
  create_editor_script "service" \
    "yq -Y -i 'if type == \"array\" then .[0].metadata.labels.edited = [\"true\"] else .metadata.labels.edited = [\"true\"] end' \"\$1\""
}

create_failing_editor() {
  create_editor_script "failing" \
    'printf "%s\n" "editor failed intentionally" >&2' \
    'exit 23'
}

create_invalid_identity_editor() {
  create_editor_script "invalid-identity" \
    "state_file=\"\${0}.state\"" \
    "if [[ ! -e \"\$state_file\" ]]; then" \
    "  touch \"\$state_file\"" \
    "  yq -Y -i 'if type == \"array\" then .[0].metadata.name = \"renamed-edit-target\" else .metadata.name = \"renamed-edit-target\" end' \"\$1\"" \
    'fi'
}

# create_editor_script writes an executable wrapper that can be passed as
# SLOCTL_EDITOR. The edit command appends the edited file path to the editor
# command, so tests provide only the wrapper body here and use the returned
# path as the editor command.
create_editor_script() {
  local name="$1"
  shift

  local timestamp
  timestamp="$(date -u +%Y%m%dT%H%M%SZ)"

  local editor_script
  editor_script="$BATS_TEST_TMPDIR/sloctl-edit-$name-editor-$timestamp.sh"

  printf '%s\n' \
    '#!/usr/bin/env bash' \
    '' \
    'set -euo pipefail' \
    '' \
    "$@" \
    > "$editor_script"
  chmod +x "$editor_script"

  printf '%s\n' "$editor_script"
}

extract_preserved_edit_file_path() {
  sed -n 's/.*A copy of your changes has been stored to "\([^"]*\)".*/\1/p' <<<"$1"
}

assert_file_contains() {
  local file="$1"
  local expected="$2"
  local contents
  contents="$(<"$file")"

  if [[ "$contents" != *"$expected"* ]]; then
    fail "Expected $file to contain: $expected"
  fi
}
