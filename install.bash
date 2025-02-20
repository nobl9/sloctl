#!/usr/bin/env bash

# The install script is based off of the Apache 2.0 licensed script from Helm,
# the Kubernetes resource manager: https://github.com/helm/helm.

PROGRAM_NAME="sloctl"
GITHUB_REPOSITORY="nobl9/$PROGRAM_NAME"

DESIRED_VERSION=""
USE_SUDO="true"
DEBUG="false"
VERIFY_CHECKSUM="true"
PROGRAM_INSTALL_DIR="/usr/local/bin"

HAS_CURL="$(type "curl" &>/dev/null && echo true || echo false)"
HAS_WGET="$(type "wget" &>/dev/null && echo true || echo false)"
HAS_OPENSSL="$(type "openssl" &>/dev/null && echo true || echo false)"

initVersion() {
  if [[ "$DESIRED_VERSION" != "" ]] && [[ "$DESIRED_VERSION" != "v"* ]]; then
    echo "Expected version arg ('${DESIRED_VERSION}') to begin with 'v', fixing..."
    export DESIRED_VERSION="v${DESIRED_VERSION}"
  fi
}

# initArch discovers the architecture for this system.
initArch() {
  ARCH=$(uname -m)
  case $ARCH in
  armv5*) ARCH="armv5" ;;
  armv6*) ARCH="armv6" ;;
  armv7*) ARCH="arm" ;;
  aarch64) ARCH="arm64" ;;
  x86) ARCH="386" ;;
  x86_64) ARCH="amd64" ;;
  i686) ARCH="386" ;;
  i386) ARCH="386" ;;
  esac
}

# initOS discovers the operating system for this system.
initOS() {
  OS=$(uname | tr '[:upper:]' '[:lower:]')

  case "$OS" in
  # Minimalist GNU for Windows
  mingw* | cygwin*) OS='windows' ;;
  esac
}

# runs the given command as root (detects if we are root already)
runAsRoot() {
  if [ $EUID -ne 0 ] && [ "$USE_SUDO" = "true" ]; then
    sudo "${@}"
  else
    "${@}"
  fi
}

# verifySupported checks that the os/arch combination is supported for
# binary builds, as well whether or not necessary tools are present.
verifySupported() {
  local supported="darwin-amd64\ndarwin-arm64\nlinux-amd64\nlinux-arm64\nwindows-amd64\nwindows-arm64"
  if ! echo "$supported" | grep -q "${OS}-${ARCH}"; then
    echo "No prebuilt binary for ${OS}-${ARCH}."
    echo "To build from source, go to https://github.com/${GITHUB_REPOSITORY}"
    exit 1
  fi

  if [ "${HAS_CURL}" != "true" ] && [ "${HAS_WGET}" != "true" ]; then
    echo "Either curl or wget is required"
    exit 1
  fi

  if [ "${VERIFY_CHECKSUM}" == "true" ] && [ "${HAS_OPENSSL}" != "true" ]; then
    echo "In order to verify checksum, openssl must first be installed."
    echo "Please install openssl or set --no-verify-checksum flag."
    exit 1
  fi
}

# checkLatestVersion checks if the desired version is available.
checkLatestVersion() {
  if [ "$DESIRED_VERSION" == "" ]; then
    # Get tag from release URL
    local latest_release_url="https://api.github.com/repos/${GITHUB_REPOSITORY}/releases/latest"
    local response=""
    if [ "${HAS_CURL}" == "true" ]; then
      response=$(curl -L --silent --show-error --fail "$latest_release_url" 2>&1 || true)
    elif [ "${HAS_WGET}" == "true" ]; then
      response=$(wget "$latest_release_url" -q -O - 2>&1 || true)
    fi
    if [[ $response =~ \"tag_name\":\ \"([^\"]+)\" ]]; then
      TAG="${BASH_REMATCH[1]}"
    fi
    if [ "$TAG" == "" ]; then
      printf "Could not retrieve the latest release tag information from %s: %s\n" "${latest_release_url}" "${response}"
      exit 1
    fi
  else
    TAG=$DESIRED_VERSION
  fi
}

# checkInstalledVersion checks which version of program is installed and
# if it needs to be changed.
checkInstalledVersion() {
  if [[ -f "${PROGRAM_INSTALL_DIR}/${PROGRAM_NAME}" ]]; then
    local version
    version=$("${PROGRAM_INSTALL_DIR}/${PROGRAM_NAME}" version)
    if [[ $version =~ "${PROGRAM_NAME}"/([0-9]+\.[0-9]+\.[0-9]+) ]]; then
      version="${BASH_REMATCH[1]}"
    fi
    if [[ "$version" == "$TAG" ]]; then
      echo "${PROGRAM_NAME} ${version} is already ${DESIRED_VERSION:-latest}"
      return 0
    else
      echo "${PROGRAM_NAME} ${TAG} is available. Changing from version ${version}."
      return 1
    fi
  else
    return 1
  fi
}

# downloadFile downloads the latest program package and also the checksum
# for that binary.
downloadFile() {
  VERSION="${TAG#v}"

  PROGRAM_DIST="${PROGRAM_NAME}-${VERSION}-${OS}-${ARCH}"
  DOWNLOAD_BASE_URL="https://github.com/${GITHUB_REPOSITORY}/releases/download/$TAG"

  DOWNLOAD_URL="${DOWNLOAD_BASE_URL}/${PROGRAM_DIST}"
  CHECKSUM_URL="${DOWNLOAD_BASE_URL}/${PROGRAM_NAME}-${VERSION}.sha256"

  PROGRAM_TMP_ROOT="$(mktemp -dt "${PROGRAM_NAME}-installer-XXXXXX")"
  PROGRAM_TMP_BIN="${PROGRAM_TMP_ROOT}/${PROGRAM_NAME}"
  PROGRAM_SUM_FILE="${PROGRAM_TMP_ROOT}/${PROGRAM_NAME}-${VERSION}.sha256"

  echo "Downloading ${DOWNLOAD_URL}"
  if [ "$HAS_CURL" == "true" ]; then
    curl -SsL --fail "$DOWNLOAD_URL" -o "$PROGRAM_TMP_BIN"
  elif [ "$HAS_WGET" == "true" ]; then
    wget -q -O "$PROGRAM_TMP_BIN" "$DOWNLOAD_URL"
  fi

  echo "Downloading checksum $CHECKSUM_URL"
  if [ "$HAS_CURL" == "true" ]; then
    curl -SsL --fail "$CHECKSUM_URL" -o "$PROGRAM_SUM_FILE"
  elif [ "$HAS_WGET" == "true" ]; then
    wget -q -O "$PROGRAM_SUM_FILE" "$CHECKSUM_URL"
  fi
}

# installFile installs the prgoram binary.
installFile() {
  echo "Preparing to install ${PROGRAM_NAME} into ${PROGRAM_INSTALL_DIR}"
  runAsRoot cp "$PROGRAM_TMP_BIN" "${PROGRAM_INSTALL_DIR}/${PROGRAM_NAME}"
  echo "${PROGRAM_NAME} installed into ${PROGRAM_INSTALL_DIR}/${PROGRAM_NAME}"
}

# verifyChecksum verifies the SHA256 checksum of the binary package.
verifyChecksum() {
  printf "Verifying checksum... "
  local actual_sum
  local expected_sum
  actual_sum=$(openssl sha1 -sha256 "$PROGRAM_TMP_BIN" | awk '{print $2}')
  expected_sum=$(awk "/${PROGRAM_DIST}/ {print \$1}" "$PROGRAM_SUM_FILE")
  if [ "$actual_sum" != "$expected_sum" ]; then
    echo "SHA sum of ${PROGRAM_TMP_BIN} does not match. Aborting."
    exit 1
  fi
  echo "Done."
}

# fail_trap is executed if an error occurs.
fail_trap() {
  result=$?
  if [ "$result" != "0" ]; then
    if [[ -n "$INPUT_ARGUMENTS" ]]; then
      echo -e "Failed to install ${PROGRAM_NAME} with the arguments provided: ${INPUT_ARGUMENTS}\n"
      help
    else
      echo "Failed to install ${PROGRAM_NAME}"
    fi
    echo -e "\nFor support, go to https://github.com/${GITHUB_REPOSITORY}."
  fi
  cleanup
  exit $result
}

# testVersion tests the installed client to make sure it is working.
testVersion() {
  set +e
  command -v "$PROGRAM_NAME"
  if [ "$?" = "1" ]; then
    echo "${PROGRAM_NAME} not found. Is ${PROGRAM_INSTALL_DIR} on your '\$PATH?'"
    exit 1
  fi
  set -e
}

# help provides possible cli installation arguments.
help() {
  local script_name
  script_name=$(basename "$0")
  cat >&2 <<EOF
Usage: ${script_name} [OPTS]

An installer script for ${PROGRAM_NAME}!
It can be used to both install ${PROGRAM_NAME} for the first time and upgrade an existing version.

OPTS:
  -h, --help            Print this message
  -v, --version         ${PROGRAM_NAME} version, when not defined it fetches the latest release from GitHub
  -d, --dir             Install directory, defaults to /usr/local/bin
  --no-sudo             Do not use sudo for installation
  --no-verify-checksum  Do not verify the checksum of the binary
  --debug               Print additional debug information
Examples:
  ${script_name} --no-sudo --version=v0.10.0 -d /home/me/go/bin
EOF
}

# cleanup temporary files.
cleanup() {
  if [[ -d "${PROGRAM_TMP_ROOT:-}" ]]; then
    rm -rf "$PROGRAM_TMP_ROOT"
  fi
}

# Execution.

# Stop execution on any error.
trap "fail_trap" EXIT
set -e

# Set debug if desired.
if [ "$DEBUG" == "true" ]; then
  set -x
fi

# Normalize args.
# Step 1: Preprocess arguments to split any --option=value pairs.
normalized_args=()
for arg in "$@"; do
  if [[ $arg == --*=* ]]; then
    # Split at the first '=': key gets the part before, value gets the part after.
    normalized_args+=("${arg%%=*}" "${arg#*=}")
  else
    normalized_args+=("$arg")
  fi
done
# Step 2: Reset the positional parameters with our normalized arguments.
set -- "${normalized_args[@]}"

# Parsing input arguments (if any).
export INPUT_ARGUMENTS="${*}"
set -u
while (("$#")); do
  case "$1" in
  --version | -v)
    shift
    if [[ $# -ne 0 ]]; then
      DESIRED_VERSION="$1"
      if [[ "$1" != "v"* ]]; then
        echo "Expected version arg ('${DESIRED_VERSION}') to begin with 'v', fixing..."
        DESIRED_VERSION="v${1}"
      fi
      shift # Shift again to remove the version argument.
    else
      echo "Please provide the desired version. e.g. --version v0.10.0"
      exit 0
    fi
    ;;
  --dir | -d)
    shift
    PROGRAM_INSTALL_DIR="$1"
    shift
    ;;
  '--no-sudo')
    USE_SUDO="false"
    ;;
  '--no-verify-checksum')
    VERIFY_CHECKSUM="false"
    ;;
  '--debug')
    DEBUG="false"
    ;;
  '--help' | -h)
    help
    exit 0
    ;;
  *)
    echo "Invalid option: $1"
    exit 1
    ;;
  esac
done
set +u

# Run.
initVersion
initArch
initOS
verifySupported
checkLatestVersion
if ! checkInstalledVersion; then
  downloadFile
  if [ "$VERIFY_CHECKSUM" == "true" ]; then
    verifyChecksum
  fi
  installFile
fi
testVersion
cleanup
