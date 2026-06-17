#!/usr/bin/env bash
# bats file_tags=unit

setup() {
	bats_require_minimum_version 1.5.0

	load "test_helper/load"
	load_lib "bats-support"
	load_lib "bats-assert"

	ensure_installed python3
}

teardown() {
	if [ -n "${server_pid:-}" ]; then
		kill "$server_pid" 2>/dev/null || true
		wait "$server_pid" 2>/dev/null || true
	fi
}

@test "get reads delayed user IDs from stdin" {
	local config server_url stdin_file
	start_get_server "$TEST_SUITE_INPUTS/get-unit/users.json"
	config="$(write_get_config "$server_url")"
	stdin_file="$BATS_TEST_TMPDIR/user-id"
	printf '%s\n' '00u4d8j2imVHGmBJH4x1' >"$stdin_file"

	run --separate-stderr python3 "$TEST_SUITE_INPUTS/get-unit/run_with_delayed_stdin.py" 2 0.2 "$stdin_file" \
		sloctl get user --config "$config" -o json

	assert_success
	assert_output --partial "00u4d8j2imVHGmBJH4x1"
}

write_get_config() {
	local server_url="$1"
	local config="$BATS_TEST_TMPDIR/config.toml"

	{
		printf '%s\n' 'defaultContext = "local"'
		printf '%s\n' '[contexts]'
		printf '%s\n' '  [contexts.local]'
		printf '%s\n' '    clientId = "id"'
		printf '%s\n' '    clientSecret = "secret"'
		printf '%s\n' '    project = "default"'
		printf '    url = "%s/api"\n' "$server_url"
		printf '%s\n' '    disableOkta = true'
		printf '%s\n' '    timeout = "1s"'
	} >"$config"

	printf '%s\n' "$config"
}

start_get_server() {
	local response_path="${1:-$TEST_SUITE_INPUTS/get/agent.yaml}"
	local url_file="$BATS_TEST_TMPDIR/server-url"

	python3 "$TEST_SUITE_INPUTS/get-unit/server.py" "$response_path" "$url_file" \
		>"$BATS_TEST_TMPDIR/server.log" 2>&1 &
	server_pid=$!

	server_url="$(wait_for_server_url "$url_file")"
}

wait_for_server_url() {
	local url_file="$1"

	for _ in {1..50}; do
		if [ -s "$url_file" ]; then
			sed -n '1p' "$url_file"
			return
		fi
		sleep 0.1
	done

	printf 'server did not start\n' >&2
	return 1
}
