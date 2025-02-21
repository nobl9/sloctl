#!/usr/bin/env bash
# bats file_tags=unit

# setup_file is run only once for the whole file.
setup_file() {
  load "test_helper/load"

  ensure_installed openssl

  run cp /usr/bin/sloctl /usr/bin/sloctl-backup
}

# teardown_file is run only once for the whole file.
teardown_file() {
  run mv /usr/bin/sloctl-backup /usr/bin/sloctl
  run rm /usr/local/bin/sloctl
}

# setup is run before each test.
setup() {
  load "test_helper/load"
  load_lib "bats-assert"
  load_lib "bats-support"
}

@test "display help message" {
  run ./install.bash --help
  assert_success
  assert_output --partial 'Usage: install.bash'
}

@test "install in default location" {
  run ./install.bash -v v0.11.0-rc1
  assert_success

  run /usr/local/bin/sloctl version
  assert_success
  assert_output 'sloctl/0.11.0-rc1-HEAD-bc9f5fd (linux amd64 go1.23.6)'
}

@test "install in custom location in the PATH" {
  run ./install.bash -v v0.11.0-rc1 -d /usr/bin
  assert_success

  run /usr/bin/sloctl version
  assert_success
  assert_output 'sloctl/0.11.0-rc1-HEAD-bc9f5fd (linux amd64 go1.23.6)'
}
