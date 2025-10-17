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

@test "missing --status flag" {
  run_sloctl review set test-slo

  assert_failure
  assert_stderr "Error: invalid status '': must be one of: reviewed, skipped, pending, overdue, notStarted"
}

@test "invalid status value" {
  run_sloctl review set test-slo --status=invalid

  assert_failure
  assert_stderr "Error: invalid status 'invalid': must be one of: reviewed, skipped, pending, overdue, notStarted"
}

@test "note flag without reviewed or skipped status" {
  run_sloctl review set test-slo --status=pending --note=note-text

  assert_failure
  assert_stderr 'Error: note annotation is only applicable for reviewed and skipped statuses'
}



@test "missing SLO name argument" {
  run_sloctl review set

  assert_failure
  assert_stderr 'Error: you must provide the SLO name as an argument'
}

@test "too many arguments" {
  run_sloctl review set slo1 slo2

  assert_failure
  assert_stderr "Error: 'review set' command accepts only single SLO name as an argument"
}