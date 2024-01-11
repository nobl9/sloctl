#!/usr/bin/env bash
# bats file_tags=unit

# setup is run before each test.
setup() {
	load "test_helper/load"
	load_lib "bats-support"
	load_lib "bats-assert"
}

@test "sloctl version" {
	TMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t "sloctl-bats") # Works on Linux and OSX.
	PKG="github.com/nobl9/sloctl/internal"
	LD_FLAGS="\
    -X ${PKG}.BuildVersion=v0.0.1 \
    -X ${PKG}.BuildGitBranch=PC-123-test \
    -X ${PKG}.BuildGitRevision=e2602ddc"
	go build \
		-ldflags "$LD_FLAGS" \
		-o "${TMP_DIR}/sloctl" \
		"$BATS_TEST_DIRNAME/../cmd/sloctl/main.go"

	run "${TMP_DIR}/sloctl" version

	expected_arch="$(go version | awk '{sub(/\//, " ", $4); print $4, $3}')"
	assert_output --partial "sloctl/v0.0.1-PC-123-test-e2602ddc (${expected_arch})"
}
