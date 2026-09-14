#!/usr/bin/env bash
# bats file_tags=e2e

# setup_file is run only once for the whole file.
setup_file() {
  load "test_helper/load"
  load_lib "bats-assert"

  generate_inputs "$BATS_FILE_TMPDIR"
  TEST_SUITE_OUTPUTS="$BATS_FILE_TMPDIR/outputs"
  mkdir "$TEST_SUITE_OUTPUTS"
  cp -R "$BATS_TEST_DIRNAME/outputs/get" "$TEST_SUITE_OUTPUTS"
  generate_outputs

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

@test "alert methods" {
  aliases="alertmethod alertmethods AlertMethods"
  test_get "AlertMethod" "$aliases" "${TEST_INPUTS}/alertmethods.yaml" "$output"
}

@test "alert policies" {
  aliases="alertpolicy alertpolicies AlertPolicies"
  test_get "AlertPolicy" "$aliases" "${TEST_INPUTS}/alertpolicies.yaml" "$output"
}

@test "alert silences" {
  aliases="alertsilence alertsilences AlertSilences"
  test_get "AlertSilence" "$aliases" "${TEST_INPUTS}/alertsilences.yaml" "$output"
}

@test "annotations" {
  aliases="annotation annotations Annotations"
  test_get "Annotation" "$aliases" "${TEST_OUTPUTS}/annotations-death-star.yaml" "$output"
}

@test "annotations filtered by slo-name" {
  want=$(read_files "${TEST_OUTPUTS}/annotations-for-slo.yaml")

  run_sloctl get annotation -p "$TEST_PROJECT" --slo=splunk-raw-rolling
  verify_get_success "$output" "$want"
}

@test "annotations filtered by category Comment" {
  want=$(read_files "${TEST_OUTPUTS}/annotations-by-category-comment.yaml")

  run_sloctl get annotation -p "$TEST_PROJECT" --category=Comment
  verify_get_success "$output" "$want"
}

@test "annotations filtered by category ReviewNote" {
  want=$(read_files "${TEST_OUTPUTS}/annotations-by-category-reviewnote.yaml")

  run_sloctl get annotation -p "$TEST_PROJECT" --category=ReviewNote
  verify_get_success "$output" "$want"
}

@test "annotations filtered by multiple categories" {
  want=$(read_files "${TEST_OUTPUTS}/annotations-death-star.yaml")

  run_sloctl get annotation -p "$TEST_PROJECT" --category=Comment --category=ReviewNote
  verify_get_success "$output" "$want"
}

@test "annotations filtered by --user flag" {
  want=$(read_files "${TEST_OUTPUTS}/annotations-death-star.yaml")

  run_sloctl get annotation -p "$TEST_PROJECT" --user
  verify_get_success "$output" "$want"
}

@test "annotations filtered by --system flag" {
  run_sloctl get annotation -p "${TEST_PROJECT}-custom" --system
  assert_success_joined_output
  assert_output "No resources found in '${TEST_PROJECT}-custom' project."
}

@test "annotations filtered by --from flag" {
  want=$(read_files "${TEST_OUTPUTS}/annotations-death-star.yaml")
  run_sloctl get annotation -p "$TEST_PROJECT" --from=2023-01-01T00:00:00Z
  verify_get_success "$output" "$want"
}

@test "annotations filtered by --to flag" {
  want=$(read_files "${TEST_OUTPUTS}/annotations-by-time-january.yaml")
  run_sloctl get annotation -p "$TEST_PROJECT" --to=2023-01-31T23:59:59Z
  verify_get_success "$output" "$want"
}

@test "annotations filtered by --from and --to combined" {
  want=$(read_files "${TEST_OUTPUTS}/annotations-by-time-january.yaml")
  run_sloctl get annotation -p "$TEST_PROJECT" --from=2023-01-01T00:00:00Z --to=2023-01-31T23:59:59Z
  verify_get_success "$output" "$want"
}

@test "annotations with no results in time range" {
  run_sloctl get annotation -p "$TEST_PROJECT" --from=2020-01-01T00:00:00Z --to=2020-12-31T23:59:59Z
  assert_success_joined_output
  assert_output "No resources found in '$TEST_PROJECT' project."
}

@test "annotations filtered by --slo and --category" {
  want=$(read_files "${TEST_OUTPUTS}/annotations-by-category-reviewnote.yaml")
  run_sloctl get annotation -p "$TEST_PROJECT" --slo=splunk-raw-rolling --category=ReviewNote
  verify_get_success "$output" "$want"
}

@test "annotations filtered by --slo and --from" {
  want=$(read_files "${TEST_OUTPUTS}/annotations-for-slo.yaml")
  run_sloctl get annotation -p "$TEST_PROJECT" --slo=splunk-raw-rolling --from=2023-01-01T00:00:00Z
  verify_get_success "$output" "$want"
}

@test "invalid annotation category" {
  run_sloctl get annotation --category Invalid
  assert_failure
  assert_stderr "Error: invalid 'category' flag value: Invalid is not a valid Category"
}

@test "data exports" {
  aliases="dataexport dataexports DataExports"
  test_get "DataExport" "$aliases" "" "$output"
}

@test "directs" {
  aliases="direct directs Directs"
  test_get "Direct" "$aliases" "${TEST_INPUTS}/directs.yaml" "$output"
}

@test "user groups" {
  aliases="usergroup usergroups UserGroups"
  test_get "UserGroup" "$aliases" "" "$output"
}

@test "projects" {
  aliases="projects project Projects"
  test_get "Project" "$aliases" "${TEST_INPUTS}/projects.yaml" "$output"
}

@test "role bindings" {
  aliases="rolebinding rolebindings RoleBindings"
  test_get "RoleBinding" "$aliases" "${TEST_INPUTS}/rolebindings.yaml" "$output"
}

@test "services" {
  aliases="services svc svcs service Services"
  test_get "Service" "$aliases" "${TEST_OUTPUTS}/services-death-star.yaml" "$output"
}

@test "slos" {
  aliases="slo slos SLOs"
  test_get "SLO" "$aliases" "${TEST_OUTPUTS}/slos-death-star.yaml" "$output"
}

@test "slos with limit and offset" {
  local pages=()
  local offset have want
  for offset in 0 1 2; do
    run_sloctl get slo -p "$TEST_PROJECT" --limit 1 --offset "$offset" -o json
    assert_success_joined_output
    assert_equal "$(jq length <<< "$output")" 1
    pages+=("$output")
  done

  have=$(printf '%s\n' "${pages[@]}" | jq -s add)
  want=$(read_files "${TEST_OUTPUTS}/slos-death-star.yaml")
  verify_get_success "$have" "$want"

  run_sloctl get slo -p "$TEST_PROJECT" --limit 1 --offset 3
  assert_success_joined_output
  assert_output "No resources found in '$TEST_PROJECT' project."
}

@test "slo pagination accepts the maximum limit and zero offset" {
  local alias want
  want=$(read_files "${TEST_OUTPUTS}/slos-death-star.yaml")
  for alias in slo slos SLO SLOs; do
    run_sloctl get "$alias" -p "$TEST_PROJECT" --limit 1000 --offset 0
    verify_get_success "$output" "$want"
  done
}

@test "slo pagination rejects invalid limits" {
  local limit
  for limit in -1 0 1001; do
    run_sloctl get slo --limit "$limit"
    assert_failure
    assert_output ""
    assert_stderr "Error: --limit must be between 1 and 1000"
  done
}

@test "slo pagination rejects negative offsets" {
  run_sloctl get slo --limit 50 --offset -1
  assert_failure
  assert_output ""
  assert_stderr "Error: --offset must be nonnegative"
}

@test "slo pagination requires a limit with an offset" {
  local offset
  for offset in 0 1; do
    run_sloctl get slo --offset "$offset"
    assert_failure
    assert_output ""
    assert_stderr "Error: --offset requires --limit"
  done
}

@test "slo pagination flags are unavailable for services" {
  local flag
  for flag in --limit --offset; do
    run_sloctl get service "$flag" 1
    assert_failure
    assert_output ""
    assert_stderr "Error: unknown flag: $flag"
  done
}

@test "slo pagination preserves name and service filters" {
  local want
  want=$(read_files "${TEST_OUTPUTS}/slo-by-service-name.yaml")

  run_sloctl get slo -p "$TEST_PROJECT" -s deputy-office \
    newrelic-rolling-timeslices-threshold-deputy-office --limit 1
  verify_get_success "$output" "$want"

  run_sloctl get slo -p "$TEST_PROJECT" -s deputy-office \
    newrelic-rolling-timeslices-threshold-deputy-office --limit 1 --offset 1
  assert_success_joined_output
  assert_output "No resources found in '$TEST_PROJECT' project."
}

@test "slos filtered by service name" {
  # Default project, no matches.
  run_sloctl get slo -s deputy-office
  assert_success_joined_output
  assert_output "No resources found in 'default' project."

  # Wrong name, no matches.
  run_sloctl get slo -s deputy-office -p "$TEST_PROJECT" newrelic-rolling-timeslices-threshold-deputy-home
  assert_success_joined_output
  assert_output "No resources found in '$TEST_PROJECT' project."

  want=$(read_files "${TEST_OUTPUTS}/slo-by-service-name.yaml")
  for flag_alias in "-s" "--service"; do
    run_sloctl get slo "$flag_alias" deputy-office -p "$TEST_PROJECT"
    verify_get_success "$output" "$want"
  done

  # Combine all filters.
  run_sloctl get slo -s deputy-office -p "$TEST_PROJECT" newrelic-rolling-timeslices-threshold-deputy-office
  verify_get_success "$output" "$want"

  # Multiple services.
  want=$(read_files "${TEST_OUTPUTS}/slos-death-star.yaml")
  run_sloctl get slo -s deputy-office -s destroyer -p "$TEST_PROJECT"
  verify_get_success "$output" "$want"
}

@test "budget adjustments" {
  aliases="budgetadjustment budgetadjustments BudgetAdjustments"
  test_get "BudgetAdjustment" "$aliases" "${TEST_INPUTS}/budgetadjustments.yaml" "$output"
}

@test "reports" {
  aliases="report reports Reports"
  test_get "Report" "$aliases" "${TEST_INPUTS}/reports.yaml" "$output"
}

@test "agent" {
  aliases="agent agents Agents"
  test_get "Agent" "$aliases" "${TEST_INPUTS}/agent.yaml" "$output"
}

@test "agent with keys" {
  for flag in -k --with-keys; do
    run_sloctl get agent -p "$TEST_PROJECT" "$flag"
    assert_success_joined_output
    # Assert length of client_id and regex of client_secret, as the latter may vary.
    client_id="$(yq -r .[].metadata.client_id <<< "$output")"
    client_secret="$(yq -r .[].metadata.client_secret <<< "$output")"

    # Assert that client_id length is either 16 or 20
    assert [ "${#client_id}" -eq 16 ] || [ "${#client_id}" -eq 20 ]

    assert_regex "${#client_secret}" "[a-zA-Z0-9_-]+"
    # Finally make sure the whole Agent definition is being presented.
    verify_get_success "$output" "$(read_files "${TEST_INPUTS}/agent.yaml")"
  done
}

@test "user by ID" {
  run_sloctl config current-user
  assert_success_joined_output
  user_id="$output"

  run_sloctl get user "$user_id" -o json
  assert_success_joined_output
  assert_equal "$(jq -r .[0].userId <<< "$output")" "$user_id"
}

@test "users" {
  run_sloctl get user -o json
  assert_success_joined_output
  assert [ "$(jq length <<< "$output")" -gt 1 ]
}

@test "users with limit" {
  run_sloctl get user --limit 1 -o json
  assert_success_joined_output
  assert [ "$(jq length <<< "$output")" -eq 1 ]
}

@test "projects, multiple names" {
  run_sloctl get project "$TEST_PROJECT" "${TEST_PROJECT}-hoth"
  verify_get_success "$output" "$(read_files "${TEST_INPUTS}/projects.yaml")"
}

@test "projects, names from stdin and positional args" {
  run --separate-stderr bash -o pipefail -c \
    'printf "%s\n" "$1" | sloctl get project "$2"' bash "$TEST_PROJECT" "${TEST_PROJECT}-hoth"
  verify_get_success "$output" "$(read_files "${TEST_INPUTS}/projects.yaml")"
}

@test "projects, names from stdin only" {
  run --separate-stderr bash -o pipefail -c \
    'printf "%s\n" "$@" | sloctl get project' bash "$TEST_PROJECT" "${TEST_PROJECT}-hoth"
  verify_get_success "$output" "$(read_files "${TEST_INPUTS}/projects.yaml")"
}

@test "projects, labels filtering, OR conditions" {
  want=$(read_files "${TEST_INPUTS}/projects.yaml")
  for label in \
    "-l purpose=defensive" \
    "-l purpose=offensive,purpose=defensive" \
    "-l purpose=defensive,purpose=offensive" \
    "-l purpose=defensive -l purpose=offensive" \
    "-l purpose=offensive -l purpose=defensive"; do
    run_sloctl get project -l "test-run=$TEST_PROJECT" "$label"
    verify_get_success "$output" "$want"
  done
}

@test "projects, labels filtering, AND conditions" {
  want=$(read_files "${TEST_INPUTS}/projects.yaml" | yq -r --arg project "$TEST_PROJECT" 'map(select(.metadata.name == $project))')
  for label in \
    "-l purpose=offensive" \
    "-l purpose=defensive,team=vader" \
    "-l purpose=offensive,team=vader" \
    "-l purpose=offensive,purpose=defensive,team=sidious" \
    "-l team=sidious,purpose=offensive,purpose=defensive" \
    "-l team=sidious,purpose=defensive,purpose=offensive" \
    "-l purpose=offensive -l purpose=defensive,team=sidious" \
    "-l purpose=offensive -l team=sidious,purpose=defensive" \
    "-l team=sidious -l purpose=offensive -l purpose=defensive" \
    "-l purpose=defensive -l purpose=offensive -l team=sidious" \
    "-l purpose=offensive -l purpose=defensive -l team=sidious"; do
    run_sloctl get project -l "test-run=$TEST_PROJECT" "$label"
    verify_get_success "$output" "$want"
  done
}

@test "projects, labels filtering with name" {
  run_sloctl get project -l purpose=defensive "${TEST_PROJECT}-hoth"
  want=$(read_files "${TEST_INPUTS}/projects.yaml" | yq -r --arg project "${TEST_PROJECT}-hoth" 'map(select(.metadata.name == $project))')
  verify_get_success "$output" "$want"

  run_sloctl get project -l purpose=offensive "${TEST_PROJECT}-hoth"
  assert_success_joined_output
  assert_output "No resources found."
}

@test "check full alert policy output" {
  run_sloctl get alertpolicy -p "$TEST_PROJECT" trigger-alert-immediately
  assert_success_joined_output
  assert_equal \
    "$(yq --sort-keys -y -r . <<< "$output")" \
    "$(yq --sort-keys -y -r . "${TEST_OUTPUTS}/alertpolicy.yaml")"
}

@test "check full direct output" {
  run_sloctl get direct -p "$TEST_PROJECT" splunk-direct
  assert_success_joined_output
  assert_equal \
    "$(yq --sort-keys -y -r . <<< "$output")" \
    "$(yq --sort-keys -y -r . "${TEST_OUTPUTS}/direct.yaml")"
}

@test "check get adjustment" {
  # SLO specified - no matches.
  run_sloctl get budgetadjustments --slo slo-that-not-exists --project default
  assert_success_joined_output
  assert_output "No resources found."

  # SLO not specified - no matches.
  run_sloctl get budgetadjustments
  assert_success
}

@test "check jq filter for project" {
  for alias in --jq -q; do
    run_sloctl get project "$TEST_PROJECT" "$alias" .[].metadata.name
    assert_success_joined_output
    assert_output "$TEST_PROJECT"
  done
}

@test "report configurable client timeout" {
  SLOCTL_TIMEOUT=10ns run_sloctl get project

  assert_failure
  assert_stderr --partial - < "$TEST_OUTPUTS/client-timeout-hint.txt"
}

test_get() {
  local \
    kind="$1" \
    input="$3" \
    output="$4"
  local aliases
  IFS=" " read -ra aliases <<< "$2"
  aliases+=("$kind")

  for alias in "${aliases[@]}"; do
    # Currently we cannot apply user groups and DataExport has very strict
    # org limits making it impossible to test with applied objects.
    if [[ "$kind" == "UserGroup" ]] || [[ "$kind" == "DataExport" ]]; then
      run_sloctl get "$alias"
      assert_success_joined_output
      refute_output --partial "Available Commands:"

      continue
    fi

    if [[ "$kind" == "Project" ]] || [[ "$kind" == "BudgetAdjustment" ]] || [[ "$kind" == "Report" ]]; then
      # shellcheck disable=2046
      run_sloctl get "$alias" $(yq -r .[].metadata.name "$input")
      verify_get_success "$output" "$(read_files "$input")"

      continue
    fi

    run_sloctl get "$alias" -p "$TEST_PROJECT"
    # Default RoleBinding is created for each project once created so we
    # need to filter out only the ones we created.
    if [[ "$kind" == "RoleBinding" ]]; then
      verify_get_success \
        "$(yq '[.[] | select(.spec.roleRef == "project-viewer")]' <<< "$output")" \
        "$(read_files "$input")"
    else
      verify_get_success "$output" "$(read_files "$input")"
    fi

    # Make sure the name filtering actually works.
    first_obj_name="$(yq -r '.[0].metadata.name' "$input")"
    run_sloctl get "$alias" -p "$TEST_PROJECT" "$first_obj_name"
    verify_get_success "$output" "$(yq -Y '[.[0]]' "$input")"
  done

  for alias in "${aliases[@]}"; do
    if [[ "$kind" == "Project" ]] || [[ "$kind" == "UserGroup" ]] || [[ "$kind" == "BudgetAdjustment" ]] || [[ "$kind" == "Report" ]]; then
      run_sloctl get "$alias" "fake-name-123-321"
      assert_success_joined_output
      assert_output "No resources found."

      continue
    fi

    run_sloctl get "$alias" "fake-name-123-321"
    assert_success_joined_output
    assert_output "No resources found in 'default' project."
    run_sloctl get "$alias" -p "fake-project-123-321"
    assert_success_joined_output
    assert_output "No resources found in 'fake-project-123-321' project."
  done
}

verify_get_success() {
  local \
    have="$1" \
    want="$2"
  assert_success_joined_output
  # Since cobra does not return errors on unknown subcommands (https://github.com/spf13/cobra/issues/706)
  # we need to hack our way around it.
  refute_output --partial "Available Commands:"
  # We can't retrieve the same object we applied so we need to compare the minimum.
  filter='[.[] | {"name": .metadata.name, "project": .metadata.project, "labels": .metadata.labels, "annotations": .metadata.annotations}] | sort_by(.name, .project)'
  assert_equal \
    "$(yq --sort-keys -y -r "$filter" <<< "$have")" \
    "$(yq --sort-keys -y -r "$filter" <<< "$want")"
}
