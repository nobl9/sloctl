#!/usr/bin/env bash
# bats file_tags=e2e

# setup_file is run only once for the whole file.
setup_file() {
	load "test_helper/load"
	load_lib "bats-assert"

	generate_inputs "$BATS_FILE_TMPDIR"
	run_sloctl apply -f "'$TEST_INPUTS/**'"
	assert_success
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

@test "dataexport" {
	run_sloctl aws-iam-ids dataexport
	assert_success
	assert_output --regexp "[-a-zA-Z0-9]+"
}

@test "direct" {
	run_sloctl aws-iam-ids direct splunk-observability-direct
	assert_success
	assert_output --regexp "externalID: [-a-zA-Z0-9]+\naccountID: \"\d+\""
}
