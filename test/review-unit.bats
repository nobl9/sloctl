#!/usr/bin/env bash
# bats file_tags=unit

# setup_file is run only once for the whole file.
setup_file() {
  export SLOCTL_CLIENT_ID=id
  export SLOCTL_CLIENT_SECRET=secret
}

# setup is run before each test.
setup() {
  load "test_helper/load"
  load_lib "bats-support"
  load_lib "bats-assert"
}

@test "note flag with to-review status should fail" {
  run_sloctl review set-status to-review test-slo --note=note-text

  assert_failure
  assert_stderr 'Error: unknown flag: --note'
}

@test "note flag with overdue status should fail" {
  run_sloctl review set-status overdue test-slo --note=note-text

  assert_failure
  assert_stderr 'Error: unknown flag: --note'
}

@test "note flag with not-started status should fail" {
  run_sloctl review set-status not-started test-slo --note=note-text

  assert_failure
  assert_stderr 'Error: unknown flag: --note'
}

@test "missing SLO name argument for reviewed" {
  run_sloctl review set-status reviewed

  assert_failure
  assert_stderr 'Error: you must provide the SLO name as an argument'
}

@test "missing SLO name argument for skipped" {
  run_sloctl review set-status skipped

  assert_failure
  assert_stderr 'Error: you must provide the SLO name as an argument'
}

@test "missing SLO name argument for to-review" {
  run_sloctl review set-status to-review

  assert_failure
  assert_stderr 'Error: you must provide the SLO name as an argument'
}

@test "too many arguments for reviewed" {
  run_sloctl review set-status reviewed slo1 slo2

  assert_failure
  assert_stderr "Error: command accepts only single SLO name as an argument"
}

@test "too many arguments for skipped" {
  run_sloctl review set-status skipped slo1 slo2

  assert_failure
  assert_stderr "Error: command accepts only single SLO name as an argument"
}

@test "too many arguments for to-review" {
  run_sloctl review set-status to-review slo1 slo2

  assert_failure
  assert_stderr "Error: command accepts only single SLO name as an argument"
}
