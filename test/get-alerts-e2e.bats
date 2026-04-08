#!/usr/bin/env bash
# bats file_tags=e2e

# Alerts are pre-populated in the database with static, deterministic data.
# All alert fields including metadata.name are fixed;
# only the organization field is env-dependent and stripped during comparison.

# setup_file is run only once for the whole file.
setup_file() {
  export TEST_PROJECT="alert-test-project"
  export TEST_PROJECT_2="alert-test-project-2"
  export TEST_OUTPUTS="$BATS_TEST_DIRNAME/outputs/get-alerts-e2e"
}

# setup is run before each test.
setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"
}

@test "get all alerts in project" {
  want=$(read_files "${TEST_OUTPUTS}/all-alerts.yaml")

  run_sloctl get alert -p "$TEST_PROJECT"
  verify_alert_output "$output" "$want"
}

@test "get alert aliases work" {
  want=$(read_files "${TEST_OUTPUTS}/all-alerts.yaml")

  for alias in alert alerts Alert Alerts; do
    run_sloctl get "$alias" -p "$TEST_PROJECT"
    verify_alert_output "$output" "$want"
  done
}

@test "get alerts no results for non-existent project" {
  run_sloctl get alert -p "non-existent-project-xyz-123"
  assert_success_joined_output
  assert_output "No resources found in 'non-existent-project-xyz-123' project."
}

@test "get alerts no results for non-existent alert-policy" {
  run_sloctl get alert -p "$TEST_PROJECT" --alert-policy "non-existent-policy-xyz-123"
  assert_success_joined_output
  assert_output "No resources found in '$TEST_PROJECT' project."
}

@test "get alerts filtered by --alert-policy alert-test-policy-high" {
  want=$(read_files "${TEST_OUTPUTS}/policy-high-alerts.yaml")

  run_sloctl get alert -p "$TEST_PROJECT" --alert-policy alert-test-policy-high
  verify_alert_output "$output" "$want"
}

@test "get alerts filtered by --alert-policy alert-test-policy-medium" {
  want=$(read_files "${TEST_OUTPUTS}/policy-medium-alerts.yaml")

  run_sloctl get alert -p "$TEST_PROJECT" --alert-policy alert-test-policy-medium
  verify_alert_output "$output" "$want"
}

@test "get alerts filtered by --alert-policy alert-test-policy-low" {
  want=$(read_files "${TEST_OUTPUTS}/policy-low-alerts.yaml")

  run_sloctl get alert -p "$TEST_PROJECT" --alert-policy alert-test-policy-low
  verify_alert_output "$output" "$want"
}

@test "get alerts filtered by --alert-policy with multiple values" {
  want=$(read_files "${TEST_OUTPUTS}/all-alerts.yaml")

  run_sloctl get alert -p "$TEST_PROJECT" \
    --alert-policy alert-test-policy-high \
    --alert-policy alert-test-policy-medium \
    --alert-policy alert-test-policy-low
  verify_alert_output "$output" "$want"
}

@test "get alerts filtered by --slo flag" {
  want=$(read_files "${TEST_OUTPUTS}/all-alerts.yaml")

  run_sloctl get alert -p "$TEST_PROJECT" --slo alert-test-slo
  verify_alert_output "$output" "$want"
}

@test "get alerts filtered by --service flag" {
  want=$(read_files "${TEST_OUTPUTS}/all-alerts.yaml")

  run_sloctl get alert -p "$TEST_PROJECT" --service alert-test-service
  verify_alert_output "$output" "$want"
}

@test "get alerts filtered by --objective default" {
  want=$(read_files "${TEST_OUTPUTS}/objective-default-alerts.yaml")

  run_sloctl get alert -p "$TEST_PROJECT" --objective default
  verify_alert_output "$output" "$want"
}

@test "get alerts filtered by --objective critical" {
  want=$(read_files "${TEST_OUTPUTS}/objective-critical-alerts.yaml")

  run_sloctl get alert -p "$TEST_PROJECT" --objective critical
  verify_alert_output "$output" "$want"
}

@test "get only triggered alerts" {
  want=$(read_files "${TEST_OUTPUTS}/triggered-alerts.yaml")

  run_sloctl get alert -p "$TEST_PROJECT" --triggered --resolved=false
  verify_alert_output "$output" "$want"
}

@test "get only resolved alerts" {
  want=$(read_files "${TEST_OUTPUTS}/resolved-alerts.yaml")

  run_sloctl get alert -p "$TEST_PROJECT" --resolved --triggered=false
  verify_alert_output "$output" "$want"
}

@test "get alerts with combined --alert-policy and --triggered filter" {
  want=$(read_files "${TEST_OUTPUTS}/policy-high-triggered.yaml")

  run_sloctl get alert -p "$TEST_PROJECT" --alert-policy alert-test-policy-high --triggered --resolved=false
  verify_alert_output "$output" "$want"
}

@test "get alerts with combined --slo and --alert-policy filter" {
  want=$(read_files "${TEST_OUTPUTS}/policy-high-alerts.yaml")

  run_sloctl get alert -p "$TEST_PROJECT" --slo alert-test-slo --alert-policy alert-test-policy-high
  verify_alert_output "$output" "$want"
}

@test "get alerts with combined --service and --resolved filter" {
  want=$(read_files "${TEST_OUTPUTS}/resolved-alerts.yaml")

  run_sloctl get alert -p "$TEST_PROJECT" --service alert-test-service --resolved --triggered=false
  verify_alert_output "$output" "$want"
}

@test "get all alerts in project-2" {
  want=$(read_files "${TEST_OUTPUTS}/project-2-all-alerts.yaml")

  run_sloctl get alert -p "$TEST_PROJECT_2"
  verify_alert_output "$output" "$want"
}

@test "get only triggered alerts in project-2" {
  want=$(read_files "${TEST_OUTPUTS}/project-2-triggered.yaml")

  run_sloctl get alert -p "$TEST_PROJECT_2" --triggered --resolved=false
  verify_alert_output "$output" "$want"
}

@test "get only resolved alerts in project-2" {
  want=$(read_files "${TEST_OUTPUTS}/project-2-resolved.yaml")

  run_sloctl get alert -p "$TEST_PROJECT_2" --resolved --triggered=false
  verify_alert_output "$output" "$want"
}

@test "get alerts with --from time range" {
  want=$(read_files "${TEST_OUTPUTS}/from-alerts.yaml")

  run_sloctl get alert -p "$TEST_PROJECT" --from 2026-01-15T11:00:00Z
  verify_alert_output "$output" "$want"
}

@test "get alerts with --to time range" {
  want=$(read_files "${TEST_OUTPUTS}/to-alerts.yaml")

  run_sloctl get alert -p "$TEST_PROJECT" --to 2026-01-15T08:30:00Z
  verify_alert_output "$output" "$want"
}

@test "get alerts with --from and --to combined" {
  want=$(read_files "${TEST_OUTPUTS}/from-to-alerts.yaml")

  run_sloctl get alert -p "$TEST_PROJECT" --from 2026-01-15T09:00:00Z --to 2026-01-15T10:02:00Z
  verify_alert_output "$output" "$want"
}

@test "get alerts with narrow --from and --to returns no results" {
  run_sloctl get alert -p "$TEST_PROJECT" --from 2020-01-01T00:00:00Z --to 2020-01-02T00:00:00Z
  assert_success_joined_output
  assert_output "No resources found in '$TEST_PROJECT' project."
}

@test "get alert by name" {
  run_sloctl get alert -p "$TEST_PROJECT"
  assert_success_joined_output
  first_alert="$output"
  alert_name=$(yq -r '.[0].metadata.name' <<<"$first_alert")

  run_sloctl get alert "$alert_name" -p "$TEST_PROJECT"
  verify_alert_output "$output" "$(yq -Y '[.[0]]' <<<"$first_alert")"
}

@test "get alert output in JSON format" {
  want=$(read_files "${TEST_OUTPUTS}/all-alerts.yaml")

  run_sloctl get alert -p "$TEST_PROJECT" -o json
  verify_alert_output "$output" "$want"
}

@test "get alert output in YAML format" {
  want=$(read_files "${TEST_OUTPUTS}/all-alerts.yaml")

  run_sloctl get alert -p "$TEST_PROJECT" -o yaml
  verify_alert_output "$output" "$want"
}

@test "get alert with --jq filter" {
  run_sloctl get alert -p "$TEST_PROJECT" --jq '.[0].metadata.name'
  assert_success_joined_output
  refute_output --partial "Available Commands:"
  assert [ -n "$output" ]

  run_sloctl get alert -p "$TEST_PROJECT" --jq '[.[].spec.severity] | sort'
  assert_success_joined_output
  assert_equal \
    "$(yq --sort-keys -y -r '.' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<<'["High","High","High","High","Low","Low","Medium","Medium"]')"
}

@test "get alert with -q jq alias" {
  run_sloctl get alert -p "$TEST_PROJECT" -q '[.[].spec.severity] | sort'
  assert_success_joined_output
  assert_equal \
    "$(yq --sort-keys -y -r '.' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<<'["High","High","High","High","Low","Low","Medium","Medium"]')"
}

@test "get alert with --all-projects flag" {
  run_sloctl get alert -A
  assert_success_joined_output
  refute_output --partial "Available Commands:"

  assert_equal "$(yq -r '[.[].kind] | unique | .[]' <<<"$output")" "Alert"
  count=$(yq -r 'length' <<<"$output")
  assert [ "$count" = 10 ]
}

# verify_alert_output compares the actual alert output against expected YAML.
# Only the organization field is env-dependent and stripped during comparison.
# All other fields including metadata.name, timestamps, conditions,
# and coolDown are static and compared as-is.
verify_alert_output() {
  local \
    have="$1" \
    want="$2"
  assert_success_joined_output
  refute_output --partial "Available Commands:"
  filter='[.[] | del(.organization)] | sort_by(.spec.slo.name, .spec.status, .spec.severity)'
  assert_equal \
    "$(yq --sort-keys -y -r "$filter" <<<"$have")" \
    "$(yq --sort-keys -y -r "$filter" <<<"$want")"
}
