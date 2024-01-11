setup_suite() {
	load "test_helper/load"

  # General dependencies shared by all tests.
  ensure_installed go jq yq git

  export SLOCTL_BIN="$BATS_SUITE_TMPDIR/sloctl"
  go build \
    -ldflags "-s -w" \
    -o "$SLOCTL_BIN" \
    "$BATS_TEST_DIRNAME/../cmd/sloctl/main.go"

  export TEST_SUITE_OUTPUTS="$BATS_TEST_DIRNAME/outputs"
  export TEST_SUITE_INPUTS="$BATS_TEST_DIRNAME/inputs"
}
