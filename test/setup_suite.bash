setup_suite() {
  load "test_helper/sloctl-utils/load"

  # General dependencies shared by all tests.
  ensure_installed go jq yq git

  # Version does need to be kept up to date,
  # it is only here to assure we do not break `sloctl version` command.
  PKG="github.com/nobl9/n9/pkg/version"
  LD_FLAGS="$(
    cat <<EOF
    -X ${PKG}.BuildVersion=A.B.C.D
    -X ${PKG}.BuildGitBranch=$(git rev-parse --short=8 HEAD)
    -X ${PKG}.BuildGitRevision=$(git rev-parse --abbrev-ref HEAD)
EOF
  )"
  export SLOCTL_BIN="$BATS_SUITE_TMPDIR/sloctl"
  go build \
    -ldflags "$LD_FLAGS" \
    -o "$SLOCTL_BIN" \
    "$BATS_TEST_DIRNAME/../main.go"

  export TEST_SUITE_OUTPUTS="$BATS_TEST_DIRNAME/outputs"
  export TEST_SUITE_INPUTS="$BATS_TEST_DIRNAME/inputs"
}
