# run_sloctl
# ==========
#
# Summary: Run the sloctl command.
#
# Usage: run_sloctl <args>
#
# Options:
#   <args>    Arguments to sloctl invocation.
#             These include subcommands like 'apply', and flags.
#
# The output of sloctl is sanitized, the trailing whitespaces,
# if present, are removed for easier output validation.
# Stderr is separated from stdout into $stderr and $output.
run_sloctl() {
  bats_require_minimum_version 1.5.0
  run --separate-stderr bash -c "set -eo pipefail && sloctl $* | sed 's/ *$//'"
}

# read_files
# ==========
#
# Summary: Read the provided files and convert them into one YAML list.
#
# Usage: read_files <file_paths>
#
# Options:
#   <file_paths>    File paths to read from.
#
# Using -s (slurp) switch helps unify all the inputs under a single list.
# This way each input can be either flattened (if an array) or added
# to the list as is. This is particularly useful with '---' separate
# documents style.
# yq works with json as it is only a preprocessor for jq.
read_files() {
  yq -sY '[ .[] | if type == "array" then .[] else . end]' "$@"
}

# assert_applied
# ==============
#
# Summary: Fail if the expected objects were not applied.
#
# Usage: assert_applied <expected>
#
# Options:
#   <expected>    The expected YAML string.
assert_applied() {
  _assert_objects_existence "apply" "$1"
}

# assert_deleted
# ==============
#
# Summary: Fail if the expected objects were not deleted.
#
# Usage: assert_deleted <expected>
#
# Options:
#   <expected>    The expected YAML string.
assert_deleted() {
  _assert_objects_existence "delete" "$1"
}

# _assert_objects_existence
# =========================
#
# Summary: Helper function which either asserts objects exist of not.
#
# Usage: _assert_objects_existence <verb> <expected>
#
# Options:
#   <verb>        Either 'apply' or 'delete'.
#   <objects>     List of objects to assert existence for.
#
# yq -c (compact) switch is used in order for 'read -r' to put each
# document on a separate line, which is then processed with 'read -r'.
# If the processed object is not of kind Project or RoleBinding, '-p' flag
# is added to sloctl invocation.
# 'sloctl get ${kind} ${name} -p ${project}' is used to retrieve each object
# and verify it with the respective <verb> logic:
# - apply: assert that the output contains the expected object.
# - delete: assert that the output contains 'No resources found'.
_assert_objects_existence() {
  load_lib "bats-support"

  assert [ -n "$2" ]
  assert [ "$(yq -r 'type' <<<"$2")" = "array" ]

  yq -c .[] <<<"$2" | while read -r object; do
    name=$(yq -r .metadata.name <<<"$object")
    kind=$(yq -r .kind <<<"$object")
    args=("get" "${kind,,}" "$name") # Converts kind to lowercase.
    if [[ "$kind" != "Project" ]] && [[ "$kind" != "RoleBinding" ]] && [[ "$kind" != "BudgetAdjustment" ]] && [[ "$kind" != "Report" ]]; then
      project=$(yq -r .metadata.project <<<"$object")
      args+=(-p "$project")
    fi

    case "$1" in
    apply)
      run_sloctl "${args[*]}"
      if ! refute_output --partial "No resources found"; then
        printf "Assertion failed for 'sloctl %s'\n" "${args[*]}" >&2
      fi
      # We can't retrieve the same object we applied so we need to compare the minimum.
      filter='[.[] | {"name": .metadata.name, "project": .metadata.project, "labels": .metadata.labels, "annotations": .metadata.annotations}] | sort_by(.name, .project)'
      # shellcheck disable=2154
      have=$(yq --sort-keys -y "$filter" <<<"$output")
      want=$(yq --sort-keys -y '[
          .[] | select(.kind == "'"$kind"'") |
          select(.metadata.name == "'"$name"'") |
          if .metadata.project then
            select(.metadata.project == "'"$project"'")
          else
            .
          end] | '"$filter"'' <<<"$2")
      assert_equal "$have" "$want"
      ;;
    delete)
      run_sloctl "${args[*]}"
      assert_output --partial "No resources found"
      ;;
    *)
      fail "Unknown verb '$1'"
      ;;
    esac
  done
}

# generate_inputs
# ===============
#
# Summary: Copy test inputs into a temporary directory and modify their names.
#
# Usage: generate_inputs <dir>
#
# Options:
#   <dir>    Directory to generate the inputs into.
#
# Each Project gets a hash appended to its name which contains the test number,
# the current timestamp and the git commit hash.
#
# This is done in order to avoid conflicts between tests in case we ever run
# them in parallel or a cleanup after the test fails for whatever reason.
# It works for both YAML and JSON files.
generate_inputs() {
  load_lib "bats-support"

  directory="$1"
  test_filename=$(basename "$BATS_TEST_FILENAME" .bats)
  TEST_INPUTS="$directory/$test_filename"
  mkdir "$TEST_INPUTS"

  test_hash="${BATS_TEST_NUMBER}-$(date +%s)-$SLOCTL_GIT_REVISION"
  TEST_PROJECT="e2e-$test_hash"
  # This time is used to test BudgetAdjustment objects that have a start time in the future.
  NEXT_DAY_TIME=$(date -d "@$(($(date +%s) + 1 * 24 * 60 * 60))" +%Y-%m-%dT%H:%M:%SZ)

  files=$(find "$TEST_SUITE_INPUTS/$test_filename" -type f \( -iname \*.json -o -iname \*.yaml -o -iname \*.yml \))
  for file in $files; do
    pipeline='
      if .kind == "Project" then
        .metadata.labels.origin = ["sloctl-e2e-tests"]
      else
        .
      end'
    filter='
      if type == "array" then
        [.[] | '"$pipeline"' ]
      else
        '"$pipeline"'
      end'
    new_file="${file/$TEST_SUITE_INPUTS/$directory}"
    mkdir -p "$(dirname "$new_file")"
    sed_project_replace="s/<PROJECT>/$TEST_PROJECT/g"
    sed_next_day_time_replace="s/<NEXT_DAY_TIME>/$NEXT_DAY_TIME/g"
    if [[ $file =~ .*.ya?ml ]]; then
      yq -Y "$filter" "$file" | sed "$sed_project_replace" | sed "$sed_next_day_time_replace" >"$new_file"
    elif [[ $file == *.json ]]; then
      jq "$filter" "$file" | sed "$sed_project_replace" | sed "$sed_next_day_time_replace" >"$new_file"
    else
      fail "test input file: ${file} must be either YAML or JSON"
    fi
  done

  export TEST_INPUTS
  export TEST_PROJECT
}

# select_object
# =============
#
# Summary: Select an object from a given file by its original name.
#
# Usage: select_object <name> <file>
#
# Options:
#   <name>    Object name to search for.
#   <file>    File path(s) to read from.
#
# Since generate_inputs appends hashes to Project names in order to
# extract an object by its former name a regex match with jq 'test'
# function has to be performed.
select_object() {
  yq '[if type == "array" then .[] else . end |
    select(.metadata.name | test("^'"$1"'"))]' "$1" "$2"
}

# ensure_installed
# ================
#
# Summary: Ensure the provided dependencies are installed.
#
# Usage: ensure_installed <dependencies>
#
# Options:
#   <dependencies>    List of dependencies to check for.
#
# If 'yq' is provided as one of the dependencies, ensure it is coming from https://github.com/kislyuk/yq.
ensure_installed() {
  load_lib "bats-support"

  for dep in "$@"; do
    if ! command -v "$dep" >/dev/null 2>&1; then
      fail "ERROR: $dep is not installed!"
    fi
    if [ "$dep" = "yq" ] && [ "$(yq --help | grep "kislyuk/yq")" -eq 1 ]; then
      fail "ERROR: yq is not installed from https://github.com/kislyuk/yq!"
    fi
  done
}

# load_lib
# ========
#
# Summary: Load a given bats library.
#
# Usage: load_lib <name>
#
# Options:
#   <name>    Name of the library to load.
load_lib() {
  local name="$1"
  load "/usr/lib/bats/${name}/load.bash"
}

# assert_success_joined_output
# ============================
#
# Summary: Assert success of command.
#
# Usage: assert_success_joined_output
#
# In case erroroneus code is detected, both stderr and stdout are conjoined.
# This is neccessary due to `run --separate-stderr` usage.
# Otherwise, only stdout is printed which is not very useful.
assert_success_joined_output() {
  output+="
$stderr" assert_success
}

# assert_stderr
# =============
#
# Summary: Fail if `$stderr' does not match the expected stderr.
#
# Usage: assert_stderr [-p | -e] [- | [--] <expected>]
#
# Options:
#   -p, --partial  Match if `expected` is a substring of `$stderr`
#   -e, --regexp   Treat `expected` as an extended regular expression
#   -, --stdin     Read `expected` value from STDIN
#   <expected>     The expected value, substring or regular expression
#
# IO:
#   STDIN - [=$1] expected stderr
#   STDERR - details, on failure
#            error message, on error
# Globals:
#   stderr
# Returns:
#   0 - if stderr matches the expected value/partial/regexp
#   1 - otherwise
#
# Similarly to `assert_output`, this function verifies that a command or function produces the expected stderr.
# (It is the logical complement of `refute_stderr`.)
# The stderr matching can be literal (the default), partial or by regular expression.
# The expected stderr can be specified either by positional argument or read from STDIN by passing the `-`/`--stdin` flag.
#
# NOTE: This was copied from bats-assert,
# once a new version is avilable in the official Docker image, we can abandond this.
assert_stderr() {
  output="$stderr"
  assert_output "$@"
}

# run_mcp_inspector
# ==========
#
# Summary: Run the modelcontextprotocol/inspector via npx.
#
# Usage: run_mcp_inspector <args>
#
# Options:
#   <args>    Arguments to inspector invocation.
#
# The output of the inspector is captured in $output.
run_mcp_inspector() {
  bats_require_minimum_version 1.5.0
  run npx -y @modelcontextprotocol/inspector --cli sloctl mcp "$@"
}

# generate_outputs
# ===============
#
# Summary: Prepare test outputs for the current test file.
#
# Usage: generate_outputs
#
# Each occurrence of <PROJECT> in the expected output files will be replaced
# with $TEST_PROJECT value.
# TEST_PROJECT must be set before calling generate_outputs.
#
# The function exports TEST_OUTPUTS environment variable which points
# at the specific file's test outputs.
generate_outputs() {
  load_lib "bats-support"

  test_filename=$(basename "$BATS_TEST_FILENAME" .bats)
  TEST_OUTPUTS="$TEST_SUITE_OUTPUTS/$test_filename"

  if [[ -z "$TEST_PROJECT" ]]; then
    fail "TEST_PROJECT environment variable is not set. Call generate_inputs() before generate_outputs()."
  fi

  # Use generated project name in the outputs.
  for file in "$TEST_OUTPUTS"/*; do
    run sed -i "s/<PROJECT>/$TEST_PROJECT/g" "$file"
  done

  export TEST_OUTPUTS
}

