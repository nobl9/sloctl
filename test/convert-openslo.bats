#!/usr/bin/env bash
# bats file_tags=unit

# setup_file is run only once for the whole file.
setup_file() {
  load "test_helper/load"
  load_lib "bats-assert"

  generate_inputs "$BATS_TEST_TMPDIR"
  export TEST_OUTPUTS="$TEST_SUITE_OUTPUTS/convert-openslo"
}

# setup is run before each test.
setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"
}

@test "missing file flag" {
  run_sloctl convert openslo

  assert_failure
  output="$stderr"
  assert_output 'Error: required flag(s) "file" not set'
}

@test "from file" {
  run_sloctl convert openslo -f "${TEST_INPUTS}/service.yaml"

  assert_success_joined_output
  assert_output - <"${TEST_OUTPUTS}/service.yaml"
}

@test "from files (YAML and JSON)" {
  run_sloctl convert openslo -f "${TEST_INPUTS}/service.yaml"  -f "${TEST_INPUTS}/nested/service.json"

  assert_success_joined_output
  assert_output - <"${TEST_OUTPUTS}/services.yaml"
}

@test "from directory" {
  run_sloctl convert openslo -f "${TEST_INPUTS}/nested"

  assert_success_joined_output
  assert_output - <"${TEST_OUTPUTS}/directory.yaml"
}

@test "from glob pattern with JSON output" {
  run_sloctl convert openslo -f "${TEST_INPUTS}/nested/**/*.json" -o json

  assert_success_joined_output
  assert_output - <"${TEST_OUTPUTS}/service.json"
}

@test "invalid alert notification target" {
  run_sloctl convert openslo -f "${TEST_INPUTS}/invalid-alert-notification-target.yaml"

  assert_failure
  output="$stderr"
  assert_output - <<EOF
Error: Validation for openslo/v1.AlertNotificationTarget pd-on-call-notification has failed for the following properties:
  - 'spec.target' with value 'PagerDuty':
    - must be one of: discord, email, jira, msteams, opsgenie, pagerduty, servicenow, slack, webhook
EOF
}

@test "referenced object does not exist" {
  run_sloctl convert openslo -f "${TEST_INPUTS}/alert-policy.yaml"

  assert_failure
  output="$stderr"
  assert_output - <<EOF
Error: failed to resolve OpenSLO object references: failed to inline OpenSLO referenced objects: failed to inline v1.AlertPolicy 'low-priority-2': v1.AlertCondition 'memory-usage-breach' referenced at 'spec.conditions[0].conditionRef' does not exist
EOF
}

@test "inline referenced object and export inlined" {
  run_sloctl convert openslo -f "${TEST_INPUTS}/alert-policy.yaml" -f "${TEST_INPUTS}/alert-condition.yaml"

  assert_success_joined_output
  assert_output - <"${TEST_OUTPUTS}/alert-policy-and-method.yaml"
}
