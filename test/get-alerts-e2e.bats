#!/usr/bin/env bash
# bats file_tags=e2e

# Alerts are pre-populated in the database by the alert-setup commandrunner.
# This test file verifies that 'sloctl get alert' returns the correct alerts
# based on various filtering flags.
#
# Expected alerts in project 'sloctl-alert-tests':
#   1. alert-triggered-slo1-high (Triggered, High, SLO: alert-test-slo-1, Service: alert-test-service-1, Policy: alert-policy-high-burn)
#   2. alert-resolved-slo1-high  (Resolved,  High, SLO: alert-test-slo-1, Service: alert-test-service-1, Policy: alert-policy-high-burn)
#   3. alert-triggered-slo2-low  (Triggered, Low,  SLO: alert-test-slo-2, Service: alert-test-service-2, Policy: alert-policy-low-burn)
#   4. alert-resolved-slo2-low   (Resolved,  Low,  SLO: alert-test-slo-2, Service: alert-test-service-2, Policy: alert-policy-low-burn)

ALERT_PROJECT="sloctl-alert-tests"

# setup is run before each test.
setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"
}

@test "get all alerts in project" {
  run_sloctl get alert -p "$ALERT_PROJECT"
  assert_success_joined_output
  refute_output --partial "No resources found"
  assert_equal "$(yq -r '[.[].kind] | unique | .[]' <<<"$output")" "Alert"

  count=$(yq -r 'length' <<<"$output")
  assert [ "$count" -ge 4 ]
}

@test "get alert aliases work" {
  for alias in alert alerts Alert Alerts; do
    run_sloctl get "$alias" -p "$ALERT_PROJECT"
    assert_success_joined_output
    refute_output --partial "No resources found"
  done
}

@test "get alerts no results for non-existent project" {
  run_sloctl get alert -p "non-existent-project-xyz-123"
  assert_success_joined_output
  assert_output "No resources found in 'non-existent-project-xyz-123' project."
}

@test "get alerts filtered by --slo flag for slo-1" {
  run_sloctl get alert -p "$ALERT_PROJECT" --slo alert-test-slo-1
  assert_success_joined_output
  refute_output --partial "No resources found"

  count=$(yq -r 'length' <<<"$output")
  assert [ "$count" -ge 2 ]

  slo_names=$(yq -r '[.[].spec.slo.name] | unique | .[]' <<<"$output")
  assert_equal "$slo_names" "alert-test-slo-1"
}

@test "get alerts filtered by --slo flag for slo-2" {
  run_sloctl get alert -p "$ALERT_PROJECT" --slo alert-test-slo-2
  assert_success_joined_output
  refute_output --partial "No resources found"

  count=$(yq -r 'length' <<<"$output")
  assert [ "$count" -ge 2 ]

  slo_names=$(yq -r '[.[].spec.slo.name] | unique | .[]' <<<"$output")
  assert_equal "$slo_names" "alert-test-slo-2"
}

@test "get alerts filtered by --slo with multiple values" {
  run_sloctl get alert -p "$ALERT_PROJECT" --slo alert-test-slo-1 --slo alert-test-slo-2
  assert_success_joined_output
  refute_output --partial "No resources found"

  count=$(yq -r 'length' <<<"$output")
  assert [ "$count" -ge 4 ]
}

@test "get alerts filtered by --service flag" {
  run_sloctl get alert -p "$ALERT_PROJECT" --service alert-test-service-1
  assert_success_joined_output
  refute_output --partial "No resources found"

  service_names=$(yq -r '[.[].spec.service.name] | unique | .[]' <<<"$output")
  assert_equal "$service_names" "alert-test-service-1"
}

@test "get alerts filtered by --service with multiple values" {
  run_sloctl get alert -p "$ALERT_PROJECT" --service alert-test-service-1 --service alert-test-service-2
  assert_success_joined_output
  refute_output --partial "No resources found"

  count=$(yq -r 'length' <<<"$output")
  assert [ "$count" -ge 4 ]
}

@test "get alerts filtered by --alert-policy flag" {
  run_sloctl get alert -p "$ALERT_PROJECT" --alert-policy alert-policy-high-burn
  assert_success_joined_output
  refute_output --partial "No resources found"

  policy_names=$(yq -r '[.[].spec.alertPolicy.name] | unique | .[]' <<<"$output")
  assert_equal "$policy_names" "alert-policy-high-burn"
}

@test "get alerts filtered by --alert-policy low-burn" {
  run_sloctl get alert -p "$ALERT_PROJECT" --alert-policy alert-policy-low-burn
  assert_success_joined_output
  refute_output --partial "No resources found"

  policy_names=$(yq -r '[.[].spec.alertPolicy.name] | unique | .[]' <<<"$output")
  assert_equal "$policy_names" "alert-policy-low-burn"
}

@test "get alerts filtered by --objective flag" {
  run_sloctl get alert -p "$ALERT_PROJECT" --objective objective-1
  assert_success_joined_output
  refute_output --partial "No resources found"

  objective_names=$(yq -r '[.[].spec.objective.name] | unique | .[]' <<<"$output")
  assert_equal "$objective_names" "objective-1"
}

@test "get only triggered alerts" {
  run_sloctl get alert -p "$ALERT_PROJECT" --triggered --resolved=false
  assert_success_joined_output
  refute_output --partial "No resources found"

  statuses=$(yq -r '[.[].spec.status] | unique | .[]' <<<"$output")
  assert_equal "$statuses" "Triggered"
}

@test "get only resolved alerts" {
  run_sloctl get alert -p "$ALERT_PROJECT" --resolved --triggered=false
  assert_success_joined_output
  refute_output --partial "No resources found"

  statuses=$(yq -r '[.[].spec.status] | unique | .[]' <<<"$output")
  assert_equal "$statuses" "Resolved"
}

@test "get alerts with combined --slo and --alert-policy filter" {
  run_sloctl get alert -p "$ALERT_PROJECT" --slo alert-test-slo-1 --alert-policy alert-policy-high-burn
  assert_success_joined_output
  refute_output --partial "No resources found"

  slo_names=$(yq -r '[.[].spec.slo.name] | unique | .[]' <<<"$output")
  assert_equal "$slo_names" "alert-test-slo-1"

  policy_names=$(yq -r '[.[].spec.alertPolicy.name] | unique | .[]' <<<"$output")
  assert_equal "$policy_names" "alert-policy-high-burn"
}

@test "get alerts with combined --slo and --triggered filter" {
  run_sloctl get alert -p "$ALERT_PROJECT" --slo alert-test-slo-1 --triggered --resolved=false
  assert_success_joined_output
  refute_output --partial "No resources found"

  count=$(yq -r 'length' <<<"$output")
  assert [ "$count" -ge 1 ]

  slo_names=$(yq -r '[.[].spec.slo.name] | unique | .[]' <<<"$output")
  assert_equal "$slo_names" "alert-test-slo-1"

  statuses=$(yq -r '[.[].spec.status] | unique | .[]' <<<"$output")
  assert_equal "$statuses" "Triggered"
}

@test "get alerts with --from time range" {
  run_sloctl get alert -p "$ALERT_PROJECT" --from 2025-06-01T00:00:00Z
  assert_success_joined_output
  refute_output --partial "No resources found"

  count=$(yq -r 'length' <<<"$output")
  assert [ "$count" -ge 1 ]
}

@test "get alerts with --to time range" {
  run_sloctl get alert -p "$ALERT_PROJECT" --to 2025-05-01T00:00:00Z
  assert_success_joined_output
  refute_output --partial "No resources found"

  count=$(yq -r 'length' <<<"$output")
  assert [ "$count" -ge 1 ]
}

@test "get alerts with --from and --to combined" {
  run_sloctl get alert -p "$ALERT_PROJECT" --from 2025-05-01T00:00:00Z --to 2025-06-15T00:00:00Z
  assert_success_joined_output
  refute_output --partial "No resources found"

  count=$(yq -r 'length' <<<"$output")
  assert [ "$count" -ge 1 ]
}

@test "get alerts with narrow --from and --to returns no results" {
  run_sloctl get alert -p "$ALERT_PROJECT" --from 2020-01-01T00:00:00Z --to 2020-01-02T00:00:00Z
  assert_success_joined_output
  assert_output "No resources found in '$ALERT_PROJECT' project."
}

@test "get alert by name" {
  run_sloctl get alert -p "$ALERT_PROJECT"
  assert_success_joined_output
  alert_name=$(yq -r '.[0].metadata.name' <<<"$output")

  run_sloctl get alert "$alert_name" -p "$ALERT_PROJECT"
  assert_success_joined_output
  refute_output --partial "No resources found"

  returned_name=$(yq -r '.[0].metadata.name' <<<"$output")
  assert_equal "$returned_name" "$alert_name"
}

@test "get alert output in JSON format" {
  run_sloctl get alert -p "$ALERT_PROJECT" -o json
  assert_success_joined_output
  refute_output --partial "No resources found"

  kind=$(echo "$output" | jq -r '.[0].kind')
  assert_equal "$kind" "Alert"
}

@test "get alert output in YAML format" {
  run_sloctl get alert -p "$ALERT_PROJECT" -o yaml
  assert_success_joined_output
  refute_output --partial "No resources found"

  kind=$(yq -r '.[0].kind' <<<"$output")
  assert_equal "$kind" "Alert"
}

@test "get alert with --jq filter" {
  run_sloctl get alert -p "$ALERT_PROJECT" --jq '.[0].metadata.name'
  assert_success_joined_output
  refute_output --partial "No resources found"
  refute_output --partial "Available Commands:"

  assert [ -n "$output" ]
  refute_output ""
}

@test "get alert with -q jq alias" {
  run_sloctl get alert -p "$ALERT_PROJECT" -q '.[0].spec.severity'
  assert_success_joined_output
  assert_regex "$output" "^(High|Low|Medium)$"
}

@test "get alert with --all-projects flag" {
  run_sloctl get alert -A
  assert_success_joined_output
  refute_output --partial "Available Commands:"
}

@test "get alerts combined --service and --resolved filter" {
  run_sloctl get alert -p "$ALERT_PROJECT" --service alert-test-service-2 --resolved --triggered=false
  assert_success_joined_output
  refute_output --partial "No resources found"

  service_names=$(yq -r '[.[].spec.service.name] | unique | .[]' <<<"$output")
  assert_equal "$service_names" "alert-test-service-2"

  statuses=$(yq -r '[.[].spec.status] | unique | .[]' <<<"$output")
  assert_equal "$statuses" "Resolved"
}

@test "get alerts verifies alert structure fields" {
  run_sloctl get alert -p "$ALERT_PROJECT" -o json
  assert_success_joined_output

  first_alert=$(echo "$output" | jq '.[0]')

  assert [ "$(jq -r '.apiVersion' <<<"$first_alert")" = "n9/v1alpha" ]
  assert [ "$(jq -r '.kind' <<<"$first_alert")" = "Alert" ]
  assert [ -n "$(jq -r '.metadata.name' <<<"$first_alert")" ]
  assert [ -n "$(jq -r '.spec.alertPolicy.name' <<<"$first_alert")" ]
  assert [ -n "$(jq -r '.spec.slo.name' <<<"$first_alert")" ]
  assert [ -n "$(jq -r '.spec.service.name' <<<"$first_alert")" ]
  assert [ -n "$(jq -r '.spec.severity' <<<"$first_alert")" ]
  assert [ -n "$(jq -r '.spec.status' <<<"$first_alert")" ]
  assert [ -n "$(jq -r '.spec.triggeredMetricTime' <<<"$first_alert")" ]
  assert [ -n "$(jq -r '.spec.triggeredClockTime' <<<"$first_alert")" ]
}
