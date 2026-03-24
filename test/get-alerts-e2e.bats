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

# verify_alerts compares the full alert output against expected values.
# It strips dynamic fields (metadata.name, timestamps) and compares
# all remaining stable fields using sorted YAML.
# Usage: verify_alerts <actual_output> <expected_yaml>
verify_alerts() {
  local \
    have="$1" \
    want="$2"
  assert_success_joined_output
  refute_output --partial "Available Commands:"
  filter='[.[] | {
    "apiVersion": .apiVersion,
    "kind": .kind,
    "project": .metadata.project,
    "severity": .spec.severity,
    "status": .spec.status,
    "slo": .spec.slo.name,
    "service": .spec.service.name,
    "alertPolicy": .spec.alertPolicy.name,
    "objective": .spec.objective.name
  }] | sort_by(.slo, .status, .severity)'
  assert_equal \
    "$(yq --sort-keys -y -r "$filter" <<<"$have")" \
    "$(yq --sort-keys -y -r "$filter" <<<"$want")"
}

# assert_alert_count verifies the exact number of alerts in the output.
assert_alert_count() {
  local \
    have="$1" \
    expected_count="$2"
  local actual_count
  actual_count=$(yq -r 'length' <<<"$have")
  assert_equal "$actual_count" "$expected_count"
}

# assert_all_alerts_have_fields verifies every alert in the output
# contains the required structural fields with non-null, non-empty values.
assert_all_alerts_have_fields() {
  local have="$1"
  local missing
  missing=$(yq -r '[.[] | select(
    .apiVersion == null or .apiVersion == "" or
    .kind == null or .kind == "" or
    .metadata.name == null or .metadata.name == "" or
    .metadata.project == null or .metadata.project == "" or
    .spec.alertPolicy.name == null or .spec.alertPolicy.name == "" or
    .spec.slo.name == null or .spec.slo.name == "" or
    .spec.service.name == null or .spec.service.name == "" or
    .spec.severity == null or .spec.severity == "" or
    .spec.status == null or .spec.status == "" or
    .spec.triggeredMetricTime == null or .spec.triggeredMetricTime == "" or
    .spec.triggeredClockTime == null or .spec.triggeredClockTime == ""
  )] | length' <<<"$have")
  assert_equal "$missing" "0"
}

@test "get all alerts in project" {
  run_sloctl get alert -p "$ALERT_PROJECT"
  assert_success_joined_output
  assert_alert_count "$output" 4
  assert_all_alerts_have_fields "$output"

  assert_equal "$(yq -r '[.[].kind] | unique | .[]' <<<"$output")" "Alert"
  assert_equal "$(yq -r '[.[].apiVersion] | unique | .[]' <<<"$output")" "n9/v1alpha"
  assert_equal "$(yq -r '[.[].metadata.project] | unique | .[]' <<<"$output")" "$ALERT_PROJECT"

  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.severity] | sort' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["High","High","Low","Low"]')"
  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.status] | sort' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["Resolved","Resolved","Triggered","Triggered"]')"
  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.slo.name] | sort' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["alert-test-slo-1","alert-test-slo-1","alert-test-slo-2","alert-test-slo-2"]')"
  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.service.name] | sort' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["alert-test-service-1","alert-test-service-1","alert-test-service-2","alert-test-service-2"]')"
  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.alertPolicy.name] | sort' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["alert-policy-high-burn","alert-policy-high-burn","alert-policy-low-burn","alert-policy-low-burn"]')"
}

@test "get alert aliases work" {
  for alias in alert alerts Alert Alerts; do
    run_sloctl get "$alias" -p "$ALERT_PROJECT"
    assert_success_joined_output
    assert_alert_count "$output" 4
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
  assert_alert_count "$output" 2

  assert_equal \
    "$(yq -r '[.[].spec.slo.name] | unique | .[]' <<<"$output")" \
    "alert-test-slo-1"
  assert_equal \
    "$(yq -r '[.[].spec.service.name] | unique | .[]' <<<"$output")" \
    "alert-test-service-1"
  assert_equal \
    "$(yq -r '[.[].spec.alertPolicy.name] | unique | .[]' <<<"$output")" \
    "alert-policy-high-burn"
  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.severity] | unique' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["High"]')"
  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.status] | sort' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["Resolved","Triggered"]')"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts filtered by --slo flag for slo-2" {
  run_sloctl get alert -p "$ALERT_PROJECT" --slo alert-test-slo-2
  assert_success_joined_output
  assert_alert_count "$output" 2

  assert_equal \
    "$(yq -r '[.[].spec.slo.name] | unique | .[]' <<<"$output")" \
    "alert-test-slo-2"
  assert_equal \
    "$(yq -r '[.[].spec.service.name] | unique | .[]' <<<"$output")" \
    "alert-test-service-2"
  assert_equal \
    "$(yq -r '[.[].spec.alertPolicy.name] | unique | .[]' <<<"$output")" \
    "alert-policy-low-burn"
  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.severity] | unique' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["Low"]')"
  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.status] | sort' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["Resolved","Triggered"]')"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts filtered by --slo with multiple values" {
  run_sloctl get alert -p "$ALERT_PROJECT" --slo alert-test-slo-1 --slo alert-test-slo-2
  assert_success_joined_output
  assert_alert_count "$output" 4

  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.slo.name] | unique | sort' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["alert-test-slo-1","alert-test-slo-2"]')"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts filtered by --service flag" {
  run_sloctl get alert -p "$ALERT_PROJECT" --service alert-test-service-1
  assert_success_joined_output
  assert_alert_count "$output" 2

  assert_equal \
    "$(yq -r '[.[].spec.service.name] | unique | .[]' <<<"$output")" \
    "alert-test-service-1"
  assert_equal \
    "$(yq -r '[.[].spec.slo.name] | unique | .[]' <<<"$output")" \
    "alert-test-slo-1"
  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.status] | sort' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["Resolved","Triggered"]')"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts filtered by --service with multiple values" {
  run_sloctl get alert -p "$ALERT_PROJECT" --service alert-test-service-1 --service alert-test-service-2
  assert_success_joined_output
  assert_alert_count "$output" 4

  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.service.name] | unique | sort' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["alert-test-service-1","alert-test-service-2"]')"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts filtered by --alert-policy flag" {
  run_sloctl get alert -p "$ALERT_PROJECT" --alert-policy alert-policy-high-burn
  assert_success_joined_output
  assert_alert_count "$output" 2

  assert_equal \
    "$(yq -r '[.[].spec.alertPolicy.name] | unique | .[]' <<<"$output")" \
    "alert-policy-high-burn"
  assert_equal \
    "$(yq -r '[.[].spec.slo.name] | unique | .[]' <<<"$output")" \
    "alert-test-slo-1"
  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.severity] | unique' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["High"]')"
  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.status] | sort' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["Resolved","Triggered"]')"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts filtered by --alert-policy low-burn" {
  run_sloctl get alert -p "$ALERT_PROJECT" --alert-policy alert-policy-low-burn
  assert_success_joined_output
  assert_alert_count "$output" 2

  assert_equal \
    "$(yq -r '[.[].spec.alertPolicy.name] | unique | .[]' <<<"$output")" \
    "alert-policy-low-burn"
  assert_equal \
    "$(yq -r '[.[].spec.slo.name] | unique | .[]' <<<"$output")" \
    "alert-test-slo-2"
  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.severity] | unique' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["Low"]')"
  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.status] | sort' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["Resolved","Triggered"]')"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts filtered by --objective flag" {
  run_sloctl get alert -p "$ALERT_PROJECT" --objective objective-1
  assert_success_joined_output
  assert_alert_count "$output" 4

  assert_equal \
    "$(yq -r '[.[].spec.objective.name] | unique | .[]' <<<"$output")" \
    "objective-1"
  assert_all_alerts_have_fields "$output"
}

@test "get only triggered alerts" {
  run_sloctl get alert -p "$ALERT_PROJECT" --triggered --resolved=false
  assert_success_joined_output
  assert_alert_count "$output" 2

  assert_equal \
    "$(yq -r '[.[].spec.status] | unique | .[]' <<<"$output")" \
    "Triggered"
  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.slo.name] | sort' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["alert-test-slo-1","alert-test-slo-2"]')"
  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.severity] | sort' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["High","Low"]')"
  assert_all_alerts_have_fields "$output"
}

@test "get only resolved alerts" {
  run_sloctl get alert -p "$ALERT_PROJECT" --resolved --triggered=false
  assert_success_joined_output
  assert_alert_count "$output" 2

  assert_equal \
    "$(yq -r '[.[].spec.status] | unique | .[]' <<<"$output")" \
    "Resolved"
  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.slo.name] | sort' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["alert-test-slo-1","alert-test-slo-2"]')"
  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.severity] | sort' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["High","Low"]')"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts with combined --slo and --alert-policy filter" {
  run_sloctl get alert -p "$ALERT_PROJECT" --slo alert-test-slo-1 --alert-policy alert-policy-high-burn
  assert_success_joined_output
  assert_alert_count "$output" 2

  assert_equal \
    "$(yq -r '[.[].spec.slo.name] | unique | .[]' <<<"$output")" \
    "alert-test-slo-1"
  assert_equal \
    "$(yq -r '[.[].spec.alertPolicy.name] | unique | .[]' <<<"$output")" \
    "alert-policy-high-burn"
  assert_equal \
    "$(yq -r '[.[].spec.service.name] | unique | .[]' <<<"$output")" \
    "alert-test-service-1"
  assert_equal \
    "$(yq --sort-keys -y -r '[.[].spec.status] | sort' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["Resolved","Triggered"]')"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts with combined --slo and --triggered filter" {
  run_sloctl get alert -p "$ALERT_PROJECT" --slo alert-test-slo-1 --triggered --resolved=false
  assert_success_joined_output
  assert_alert_count "$output" 1

  assert_equal \
    "$(yq -r '.[0].spec.slo.name' <<<"$output")" \
    "alert-test-slo-1"
  assert_equal \
    "$(yq -r '.[0].spec.status' <<<"$output")" \
    "Triggered"
  assert_equal \
    "$(yq -r '.[0].spec.severity' <<<"$output")" \
    "High"
  assert_equal \
    "$(yq -r '.[0].spec.service.name' <<<"$output")" \
    "alert-test-service-1"
  assert_equal \
    "$(yq -r '.[0].spec.alertPolicy.name' <<<"$output")" \
    "alert-policy-high-burn"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts with --from time range" {
  run_sloctl get alert -p "$ALERT_PROJECT" --from 2025-06-01T00:00:00Z
  assert_success_joined_output
  refute_output --partial "No resources found"

  assert_equal "$(yq -r '[.[].kind] | unique | .[]' <<<"$output")" "Alert"
  assert_equal "$(yq -r '[.[].metadata.project] | unique | .[]' <<<"$output")" "$ALERT_PROJECT"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts with --to time range" {
  run_sloctl get alert -p "$ALERT_PROJECT" --to 2025-05-01T00:00:00Z
  assert_success_joined_output
  refute_output --partial "No resources found"

  assert_equal "$(yq -r '[.[].kind] | unique | .[]' <<<"$output")" "Alert"
  assert_equal "$(yq -r '[.[].metadata.project] | unique | .[]' <<<"$output")" "$ALERT_PROJECT"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts with --from and --to combined" {
  run_sloctl get alert -p "$ALERT_PROJECT" --from 2025-05-01T00:00:00Z --to 2025-06-15T00:00:00Z
  assert_success_joined_output
  refute_output --partial "No resources found"

  assert_equal "$(yq -r '[.[].kind] | unique | .[]' <<<"$output")" "Alert"
  assert_equal "$(yq -r '[.[].metadata.project] | unique | .[]' <<<"$output")" "$ALERT_PROJECT"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts with narrow --from and --to returns no results" {
  run_sloctl get alert -p "$ALERT_PROJECT" --from 2020-01-01T00:00:00Z --to 2020-01-02T00:00:00Z
  assert_success_joined_output
  assert_output "No resources found in '$ALERT_PROJECT' project."
}

@test "get alert by name" {
  run_sloctl get alert -p "$ALERT_PROJECT"
  assert_success_joined_output
  first_alert="$output"
  alert_name=$(yq -r '.[0].metadata.name' <<<"$first_alert")

  run_sloctl get alert "$alert_name" -p "$ALERT_PROJECT"
  assert_success_joined_output
  assert_alert_count "$output" 1

  assert_equal \
    "$(yq -r '.[0].metadata.name' <<<"$output")" \
    "$alert_name"
  assert_equal \
    "$(yq -r '.[0].kind' <<<"$output")" \
    "Alert"
  assert_equal \
    "$(yq -r '.[0].apiVersion' <<<"$output")" \
    "n9/v1alpha"
  assert_equal \
    "$(yq -r '.[0].metadata.project' <<<"$output")" \
    "$ALERT_PROJECT"
  assert_all_alerts_have_fields "$output"

  verify_alerts "$output" "$(yq -Y '[.[0]]' <<<"$first_alert")"
}

@test "get alert output in JSON format" {
  run_sloctl get alert -p "$ALERT_PROJECT" -o json
  assert_success_joined_output
  assert_alert_count "$output" 4

  assert_equal "$(echo "$output" | jq -r '[.[].kind] | unique | .[]')" "Alert"
  assert_equal "$(echo "$output" | jq -r '[.[].apiVersion] | unique | .[]')" "n9/v1alpha"
  assert_equal "$(echo "$output" | jq -r '[.[].metadata.project] | unique | .[]')" "$ALERT_PROJECT"
  assert_all_alerts_have_fields "$output"
}

@test "get alert output in YAML format" {
  run_sloctl get alert -p "$ALERT_PROJECT" -o yaml
  assert_success_joined_output
  assert_alert_count "$output" 4

  assert_equal "$(yq -r '[.[].kind] | unique | .[]' <<<"$output")" "Alert"
  assert_equal "$(yq -r '[.[].apiVersion] | unique | .[]' <<<"$output")" "n9/v1alpha"
  assert_equal "$(yq -r '[.[].metadata.project] | unique | .[]' <<<"$output")" "$ALERT_PROJECT"
  assert_all_alerts_have_fields "$output"
}

@test "get alert with --jq filter" {
  run_sloctl get alert -p "$ALERT_PROJECT" --jq '.[0].metadata.name'
  assert_success_joined_output
  refute_output --partial "Available Commands:"
  assert [ -n "$output" ]

  run_sloctl get alert -p "$ALERT_PROJECT" --jq '[.[].spec.severity] | sort'
  assert_success_joined_output
  assert_equal \
    "$(yq --sort-keys -y -r '.' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["High","High","Low","Low"]')"
}

@test "get alert with -q jq alias" {
  run_sloctl get alert -p "$ALERT_PROJECT" -q '[.[].spec.severity] | sort'
  assert_success_joined_output
  assert_equal \
    "$(yq --sort-keys -y -r '.' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<< '["High","High","Low","Low"]')"
}

@test "get alert with --all-projects flag" {
  run_sloctl get alert -A
  assert_success_joined_output
  refute_output --partial "Available Commands:"

  assert_equal "$(yq -r '[.[].kind] | unique | .[]' <<<"$output")" "Alert"
  count=$(yq -r 'length' <<<"$output")
  assert [ "$count" -ge 4 ]
  assert_all_alerts_have_fields "$output"
}

@test "get alerts combined --service and --resolved filter" {
  run_sloctl get alert -p "$ALERT_PROJECT" --service alert-test-service-2 --resolved --triggered=false
  assert_success_joined_output
  assert_alert_count "$output" 1

  assert_equal \
    "$(yq -r '.[0].spec.service.name' <<<"$output")" \
    "alert-test-service-2"
  assert_equal \
    "$(yq -r '.[0].spec.status' <<<"$output")" \
    "Resolved"
  assert_equal \
    "$(yq -r '.[0].spec.slo.name' <<<"$output")" \
    "alert-test-slo-2"
  assert_equal \
    "$(yq -r '.[0].spec.alertPolicy.name' <<<"$output")" \
    "alert-policy-low-burn"
  assert_equal \
    "$(yq -r '.[0].spec.severity' <<<"$output")" \
    "Low"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts verifies alert structure fields" {
  run_sloctl get alert -p "$ALERT_PROJECT" -o json
  assert_success_joined_output
  assert_alert_count "$output" 4

  echo "$output" | jq -c '.[]' | while read -r alert; do
    assert [ "$(jq -r '.apiVersion' <<<"$alert")" = "n9/v1alpha" ]
    assert [ "$(jq -r '.kind' <<<"$alert")" = "Alert" ]
    assert [ "$(jq -r '.metadata.project' <<<"$alert")" = "$ALERT_PROJECT" ]
    assert [ -n "$(jq -r '.metadata.name' <<<"$alert")" ]
    assert [ -n "$(jq -r '.spec.alertPolicy.name' <<<"$alert")" ]
    assert [ -n "$(jq -r '.spec.slo.name' <<<"$alert")" ]
    assert [ -n "$(jq -r '.spec.service.name' <<<"$alert")" ]
    assert [ -n "$(jq -r '.spec.severity' <<<"$alert")" ]
    assert [ -n "$(jq -r '.spec.status' <<<"$alert")" ]
    assert [ -n "$(jq -r '.spec.triggeredMetricTime' <<<"$alert")" ]
    assert [ -n "$(jq -r '.spec.triggeredClockTime' <<<"$alert")" ]
  done
}
