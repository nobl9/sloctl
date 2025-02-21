setup_suite() {
	load "test_helper/load"

  # General dependencies shared by all tests.
  ensure_installed jq git sloctl yq

  export TEST_SUITE_OUTPUTS="$BATS_TEST_DIRNAME/outputs"
  export TEST_SUITE_INPUTS="$BATS_TEST_DIRNAME/inputs"

  export SLOCTL_GIT_REVISION="${SLOCTL_GIT_REVISION:=undefined}"
}
