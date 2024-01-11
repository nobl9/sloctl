#!/usr/bin/env bash
# bats file_tags=tools

# setup is run before each test.
setup_file() {
	load "test_helper/load"
	load_lib "bats-assert"

	ensure_installed goreleaser

	export DIST_DIR="dist"
	export SLOCTL_VERSION=v1.0.0
	export BRANCH=main
	export REVISION=test-rev

	git tag "$SLOCTL_VERSION"

	run goreleaser release --snapshot --clean
	assert_success
}

setup() {
	load "test_helper/load"
	load_lib "bats-assert"
}

teardown_file() {
	rm -rf "$DIST_DIR"
	if git show-ref --tags "$SLOCTL_VERSION" --quiet; then
	  git tag --delete "$SLOCTL_VERSION"
	fi
}

@test "metadata project name" {
	assert_meta "project_name" "sloctl"
}

@test "metadata tag" {
	assert_meta "tag" "$SLOCTL_VERSION"
}

@test "macos replacement for darwin artifact" {
	assert_equal \
		"$(jq -r \
			'.[] | select(.name | test("^sloctl-macos")) | .goos' \
			"$DIST_DIR/artifacts.json")" \
		"darwin"
}

@test "run binary" {
	goos=$(jq -r .runtime.goos "$DIST_DIR/metadata.json")
	run "$DIST_DIR/sloctl_${goos}_amd64_v1/sloctl" version
	assert_success
	assert_output --partial "sloctl/1.0.0-SNAPSHOT"
}

assert_meta() {
	local \
		key="$1" \
		want="$2"
	assert_equal "$(jq -r ."$key" "$DIST_DIR/metadata.json")" "$want"
}
