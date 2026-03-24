#!/usr/bin/env bash
# bats file_tags=e2e

ALERT_PROJECT="sloctl-alert-tests"
ALERT_OUTPUTS="$BATS_TEST_DIRNAME/outputs/get-alerts-e2e"

# setup is run before each test.
setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"
}

# verify_alert_output compares the actual alert output against expected YAML.
# Dynamic fields (timestamps, conditions, coolDown) are stripped; all stable
# fields (apiVersion, kind, metadata, severity, status, slo, service,
# alertPolicy, objective) are compared via sorted YAML.
verify_alert_output() {
  local \
    have="$1" \
    want="$2"
  assert_success_joined_output
  refute_output --partial "Available Commands:"
  filter='[.[] | {
    "apiVersion": .apiVersion,
    "kind": .kind,
    "name": .metadata.name,
    "project": .metadata.project,
    "severity": .spec.severity,
    "status": .spec.status,
    "slo": .spec.slo.name,
    "service": .spec.service.name,
    "alertPolicy": .spec.alertPolicy.name,
    "objective": .spec.objective.name
  }] | sort_by(.name)'
  assert_equal \
    "$(yq --sort-keys -y -r "$filter" <<<"$have")" \
    "$(yq --sort-keys -y -r "$filter" <<<"$want")"
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
  want=$(read_files "${ALERT_OUTPUTS}/all-alerts.yaml")

  run_sloctl get alert -p "$ALERT_PROJECT"
  verify_alert_output "$output" "$want"
  assert_all_alerts_have_fields "$output"
}

@test "get alert aliases work" {
  want=$(read_files "${ALERT_OUTPUTS}/all-alerts.yaml")

  for alias in alert alerts Alert Alerts; do
    run_sloctl get "$alias" -p "$ALERT_PROJECT"
    verify_alert_output "$output" "$want"
  done
}

@test "get alerts no results for non-existent project" {
  run_sloctl get alert -p "non-existent-project-xyz-123"
  assert_success_joined_output
  assert_output "No resources found in 'non-existent-project-xyz-123' project."
}

@test "get alerts filtered by --slo flag for slo-1" {
  want=$(read_files "${ALERT_OUTPUTS}/slo-1-alerts.yaml")

  run_sloctl get alert -p "$ALERT_PROJECT" --slo alert-test-slo-1
  verify_alert_output "$output" "$want"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts filtered by --slo flag for slo-2" {
  want=$(read_files "${ALERT_OUTPUTS}/slo-2-alerts.yaml")

  run_sloctl get alert -p "$ALERT_PROJECT" --slo alert-test-slo-2
  verify_alert_output "$output" "$want"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts filtered by --slo with multiple values" {
  want=$(read_files "${ALERT_OUTPUTS}/all-alerts.yaml")

  run_sloctl get alert -p "$ALERT_PROJECT" --slo alert-test-slo-1 --slo alert-test-slo-2
  verify_alert_output "$output" "$want"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts filtered by --service flag for service-1" {
  want=$(read_files "${ALERT_OUTPUTS}/service-1-alerts.yaml")

  run_sloctl get alert -p "$ALERT_PROJECT" --service alert-test-service-1
  verify_alert_output "$output" "$want"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts filtered by --service with multiple values" {
  want=$(read_files "${ALERT_OUTPUTS}/all-alerts.yaml")

  run_sloctl get alert -p "$ALERT_PROJECT" --service alert-test-service-1 --service alert-test-service-2
  verify_alert_output "$output" "$want"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts filtered by --alert-policy high-burn" {
  want=$(read_files "${ALERT_OUTPUTS}/high-burn-alerts.yaml")

  run_sloctl get alert -p "$ALERT_PROJECT" --alert-policy alert-policy-high-burn
  verify_alert_output "$output" "$want"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts filtered by --alert-policy low-burn" {
  want=$(read_files "${ALERT_OUTPUTS}/low-burn-alerts.yaml")

  run_sloctl get alert -p "$ALERT_PROJECT" --alert-policy alert-policy-low-burn
  verify_alert_output "$output" "$want"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts filtered by --objective flag" {
  want=$(read_files "${ALERT_OUTPUTS}/all-alerts.yaml")

  run_sloctl get alert -p "$ALERT_PROJECT" --objective objective-1
  verify_alert_output "$output" "$want"
  assert_all_alerts_have_fields "$output"
}

@test "get only triggered alerts" {
  want=$(read_files "${ALERT_OUTPUTS}/triggered-alerts.yaml")

  run_sloctl get alert -p "$ALERT_PROJECT" --triggered --resolved=false
  verify_alert_output "$output" "$want"
  assert_all_alerts_have_fields "$output"
}

@test "get only resolved alerts" {
  want=$(read_files "${ALERT_OUTPUTS}/resolved-alerts.yaml")

  run_sloctl get alert -p "$ALERT_PROJECT" --resolved --triggered=false
  verify_alert_output "$output" "$want"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts with combined --slo and --alert-policy filter" {
  want=$(read_files "${ALERT_OUTPUTS}/slo-1-high-burn.yaml")

  run_sloctl get alert -p "$ALERT_PROJECT" --slo alert-test-slo-1 --alert-policy alert-policy-high-burn
  verify_alert_output "$output" "$want"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts with combined --slo and --triggered filter" {
  want=$(read_files "${ALERT_OUTPUTS}/slo-1-triggered.yaml")

  run_sloctl get alert -p "$ALERT_PROJECT" --slo alert-test-slo-1 --triggered --resolved=false
  verify_alert_output "$output" "$want"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts combined --service and --resolved filter" {
  want=$(read_files "${ALERT_OUTPUTS}/service-2-resolved.yaml")

  run_sloctl get alert -p "$ALERT_PROJECT" --service alert-test-service-2 --resolved --triggered=false
  verify_alert_output "$output" "$want"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts with --from time range" {
  run_sloctl get alert -p "$ALERT_PROJECT" --from 2025-06-01T00:00:00Z
  assert_success_joined_output
  refute_output --partial "No resources found"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts with --to time range" {
  run_sloctl get alert -p "$ALERT_PROJECT" --to 2025-05-01T00:00:00Z
  assert_success_joined_output
  refute_output --partial "No resources found"
  assert_all_alerts_have_fields "$output"
}

@test "get alerts with --from and --to combined" {
  run_sloctl get alert -p "$ALERT_PROJECT" --from 2025-05-01T00:00:00Z --to 2025-06-15T00:00:00Z
  assert_success_joined_output
  refute_output --partial "No resources found"
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
  verify_alert_output "$output" "$(yq -Y '[.[0]]' <<<"$first_alert")"
  assert_all_alerts_have_fields "$output"
}

@test "get alert output in JSON format" {
  want=$(read_files "${ALERT_OUTPUTS}/all-alerts.yaml")

  run_sloctl get alert -p "$ALERT_PROJECT" -o json
  verify_alert_output "$output" "$want"
  assert_all_alerts_have_fields "$output"
}

@test "get alert output in YAML format" {
  want=$(read_files "${ALERT_OUTPUTS}/all-alerts.yaml")

  run_sloctl get alert -p "$ALERT_PROJECT" -o yaml
  verify_alert_output "$output" "$want"
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
    "$(yq --sort-keys -y -r '.' <<<'["High","High","Low","Low"]')"
}

@test "get alert with -q jq alias" {
  run_sloctl get alert -p "$ALERT_PROJECT" -q '[.[].spec.severity] | sort'
  assert_success_joined_output
  assert_equal \
    "$(yq --sort-keys -y -r '.' <<<"$output")" \
    "$(yq --sort-keys -y -r '.' <<<'["High","High","Low","Low"]')"
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

@test "get alerts verifies complete alert structure" {
  want=$(read_files "${ALERT_OUTPUTS}/all-alerts.yaml")

  run_sloctl get alert -p "$ALERT_PROJECT" -o json
  verify_alert_output "$output" "$want"
  assert_all_alerts_have_fields "$output"

  echo "$output" | jq -c '.[]' | while read -r alert; do
    assert [ "$(jq -r '.apiVersion' <<<"$alert")" = "n9/v1alpha" ]
    assert [ "$(jq -r '.kind' <<<"$alert")" = "Alert" ]
    assert [ "$(jq -r '.metadata.project' <<<"$alert")" = "$ALERT_PROJECT" ]
    assert [ -n "$(jq -r '.spec.triggeredMetricTime' <<<"$alert")" ]
    assert [ -n "$(jq -r '.spec.triggeredClockTime' <<<"$alert")" ]
    assert [ -n "$(jq -r '.spec.coolDown' <<<"$alert")" ]
    assert [ -n "$(jq -r '.spec.conditions' <<<"$alert")" ]
  done
}
