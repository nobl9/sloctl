#!/usr/bin/env bash
# bats file_tags=e2e

# setup_file is run only once for the whole file.
setup_file() {
  load "test_helper/load"
  load_lib "bats-assert"

  generate_inputs "$BATS_FILE_TMPDIR"
  run_sloctl apply -f "'$TEST_INPUTS/**'"
  assert_success_joined_output

  export TEST_OUTPUTS="$TEST_SUITE_OUTPUTS/get"
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
  aliases="alertmethod alertmethods"
  test_get "AlertMethod" "$aliases" "${TEST_INPUTS}/alertmethods.yaml" "$output"
}

@test "alert policies" {
  aliases="alertpolicy alertpolicies"
  test_get "AlertPolicy" "$aliases" "${TEST_INPUTS}/alertpolicies.yaml" "$output"
}

@test "alert silences" {
  aliases="alertsilence alertsilences"
  test_get "AlertSilence" "$aliases" "${TEST_INPUTS}/alertsilences.yaml" "$output"
}

@test "annotations" {
  aliases="annotation annotations"
  test_get "Annotation" "$aliases" "${TEST_INPUTS}/annotations.yaml" "$output"
}

@test "data exports" {
  aliases="dataexport dataexports"
  test_get "DataExport" "$aliases" "" "$output"
}

@test "directs" {
  aliases="direct directs"
  test_get "Direct" "$aliases" "${TEST_INPUTS}/directs.yaml" "$output"
}

@test "user groups" {
  aliases="usergroup usergroups"
  test_get "UserGroup" "$aliases" "" "$output"
}

@test "projects" {
  aliases="projects project"
  test_get "Project" "$aliases" "${TEST_INPUTS}/projects.yaml" "$output"
}

@test "role bindings" {
  aliases="rolebinding rolebindings"
  test_get "RoleBinding" "$aliases" "${TEST_INPUTS}/rolebindings.yaml" "$output"
}

@test "services" {
  aliases="services svc svcs service"
  test_get "Service" "$aliases" "${TEST_INPUTS}/services.yaml" "$output"
}

@test "slos" {
  aliases="slo slos"
  test_get "SLO" "$aliases" "${TEST_INPUTS}/slos.yaml" "$output"
}

@test "slos filtered by service name" {
  # Default project, no matches.
  run_sloctl get slo -s deputy-office
  assert_success_joined_output
  assert_output "No resources found."

  # Wrong name, no matches.
  run_sloctl get slo -s deputy-office -p death-star newrelic-rolling-timeslices-threshold-deputy-home
  assert_success_joined_output
  assert_output "No resources found."

  want=$(read_files "${TEST_OUTPUTS}/slo-by-service-name.yaml")
  for flag_alias in "-s" "--service"; do
    run_sloctl get slo "$flag_alias" deputy-office -p death-star
    verify_get_success "$output" "$want"
  done

  # Combine all filters.
  run_sloctl get slo -s deputy-office -p death-star newrelic-rolling-timeslices-threshold-deputy-office
  verify_get_success "$output" "$want"
}

@test "budget adjustments" {
  aliases="budgetadjustment budgetadjustments"
  test_get "BudgetAdjustment" "$aliases" "${TEST_INPUTS}/budgetadjustments.yaml" "$output"
}

@test "reports" {
  aliases="report reports"
  test_get "Report" "$aliases" "${TEST_INPUTS}/reports.yaml" "$output"
}

@test "agent" {
  aliases="agent agents"
  test_get "Agent" "$aliases" "${TEST_INPUTS}/agent.yaml" "$output"
}

@test "agent with keys" {
  for flag in -k --with-keys; do
    run_sloctl get agent -p "death-star" "$flag"
    assert_success_joined_output
    # Assert length of client_id and regex of client_secret, as the latter may vary.
    client_id="$(yq -r .[].metadata.client_id <<<"$output")"
    client_secret="$(yq -r .[].metadata.client_secret <<<"$output")"
    assert_equal "${#client_id}" 20
    assert_regex "${#client_secret}" "[a-zA-Z0-9_-]+"
    # Finally make sure the whole Agent definition is being presented.
    verify_get_success "$output" "$(read_files "${TEST_INPUTS}/agent.yaml")"
  done
}

@test "projects, multiple names" {
  run_sloctl get project death-star hoth-base
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
    run_sloctl get project "$label"
    verify_get_success "$output" "$want"
  done
}

@test "projects, labels filtering, AND conditions" {
  want=$(read_files "${TEST_INPUTS}/projects.yaml" | yq -r '.[] |= select(.metadata.name == "death-star")')
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
    run_sloctl get project "$label"
    verify_get_success "$output" "$want"
  done
}

@test "projects, labels filtering with name" {
  run_sloctl get project -l purpose=defensive hoth-base
  want=$(read_files "${TEST_INPUTS}/projects.yaml" | yq -r '.[] |= select(.metadata.name == "hoth-base")')
  verify_get_success "$output" "$want"

  run_sloctl get project -l purpose=offensive hoth-base
  assert_success_joined_output
  assert_output "No resources found."
}

@test "check full alert policy output" {
  run_sloctl get alertpolicy -p death-star trigger-alert-immediately
  assert_success_joined_output
  assert_equal \
    "$(yq --sort-keys -y -r . <<<"$output")" \
    "$(yq --sort-keys -y -r . "${TEST_OUTPUTS}/alertpolicy.yaml")"
}

@test "check full direct output" {
  run_sloctl get direct -p death-star splunk-direct
  assert_success_joined_output
  assert_equal \
    "$(yq --sort-keys -y -r . <<<"$output")" \
    "$(yq --sort-keys -y -r . "${TEST_OUTPUTS}/direct.yaml")"
}

test_get() {
  local \
    kind="$1" \
    input="$3" \
    output="$4"
  local aliases
  IFS=" " read -ra aliases <<<"$2"
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

    run_sloctl get "$alias" -p "death-star"
    # Default RoleBinding is created for each project once created so we
    # need to filter out only the ones we created.
    if [[ "$kind" == "RoleBinding" ]]; then
      verify_get_success \
        "$(yq '[.[] | select(.spec.roleRef == "project-viewer")]' <<<"$output")" \
        "$(read_files "$input")"
    else
      verify_get_success "$output" "$(read_files "$input")"
    fi

    # Make sure the name filtering actually works.
    first_obj_name="$(yq -r '.[0].metadata.name' "$input")"
    run_sloctl get "$alias" -p "death-star" "$first_obj_name"
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
    "$(yq --sort-keys -y -r "$filter" <<<"$have")" \
    "$(yq --sort-keys -y -r "$filter" <<<"$want")"
}
