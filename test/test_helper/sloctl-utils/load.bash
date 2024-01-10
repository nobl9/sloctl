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
run_sloctl() {
  run bash -c "set -o pipefail && $SLOCTL_BIN $* | sed 's/ *$//'"
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
  load "test_helper/bats-support/load"

  assert [ -n "$2" ]
  assert [ "$(yq -r 'type' <<<"$2")" = "array" ]

  yq -c .[] <<<"$2" | while read -r object; do
    name=$(yq -r .metadata.name <<<"$object")
    kind=$(yq -r .kind <<<"$object")
    args=("get" "${kind,,}" "$name") # Converts kind to lowercase.
    if [[ "$kind" != "Project" ]] && [[ "$kind" != "RoleBinding" ]]; then
      project=$(yq -r .metadata.project <<<"$object")
      args+=(-p "$project")
    fi

    case "$1" in
      apply)
        run_sloctl "${args[*]}"
        # shellcheck disable=2154
        have=$(yq --sort-keys -y '[.[] | del(.status)]' <<<"$output")
        want=$(yq --sort-keys -y '[
          .[] | select(.kind == "'"$kind"'") |
          select(.metadata.name == "'"$name"'") |
          if .metadata.project then
            select(.metadata.project == "'"$project"'")
          else
            .
          end]' <<<"$2")
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
# the current timestamp and the git commit hash. Each object which is part of the
#
# This is done in order to avoid conflicts between tests in case we ever run
# them in parallel or a cleanup after the test fails for whatever reason.
# It works for both YAML and JSON files.
generate_inputs() {
  load "test_helper/bats-support/load"

  directory="$1"
  test_filename=$(basename "$BATS_TEST_FILENAME" .bats)
  TEST_INPUTS="$directory/$test_filename"
  mkdir "$TEST_INPUTS"

  test_hash="${BATS_TEST_NUMBER}-$(date +%s)-$(git rev-parse --short=8 HEAD)"
  TEST_PROJECT="e2e-$test_hash"

  files=$(find "$TEST_SUITE_INPUTS/$test_filename" -type f \( -iname \*.json -o -iname \*.yaml -o -iname \*.yml \))
  for file in $files; do
    pipeline='
      if .kind == "Project" then
        .metadata.labels = {"origin": ["sloctl-e2e-tests"]}
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
    sed_replace="s/<PROJECT>/$TEST_PROJECT/g"
    if [[ $file =~ .*.ya?ml ]]; then
      yq -Y "$filter" "$file" | sed "$sed_replace" >"$new_file"
    elif [[ $file == *.json ]]; then
      jq "$filter" "$file" | sed "$sed_replace" >"$new_file"
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
  load "test_helper/bats-support/load"

  for dep in "$@"; do
    if ! command -v "$dep" >/dev/null 2>&1; then
      fail "ERROR: $dep is not installed!"
    fi
    if [ "$dep" = "yq" ] && [ "$(yq --help | grep "kislyuk/yq")" -eq 1 ]; then
      fail "ERROR: yq is not installed from https://github.com/kislyuk/yq!"
    fi
  done
}
