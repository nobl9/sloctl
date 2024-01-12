#!/usr/bin/env bash
# bats file_tags=unit

# setup is run before each test.
setup() {
	load "test_helper/load"
	load_lib "bats-support"
	load_lib "bats-assert"
}

@test "sloctl version" {
	run_sloctl version

	assert_output --regexp "sloctl/v1.0.0-PC-123-test-e2602ddc (.* .* go[0-9.])"
}
