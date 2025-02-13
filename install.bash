#!/usr/bin/env bash

# The install script is based off of the Apache 2.0 licensed script from Helm,
# the Kubernetes resource manager: https://github.com/helm/helm.

PROGRAM_NAME="sloctl"
GITHUB_REPOSITORY="nobl9/$PROGRAM_NAME"

: "${BINARY_NAME:=$PROGRAM_NAME}"
: "${USE_SUDO:="true"}"
: "${DEBUG:="false"}"
: "${VERIFY_CHECKSUM:="true"}"
: "${SLOCTL_INSTALL_DIR:="/usr/local/bin"}"

HAS_CURL="$(type "curl" &>/dev/null && echo true || echo false)"
HAS_WGET="$(type "wget" &>/dev/null && echo true || echo false)"
HAS_OPENSSL="$(type "openssl" &>/dev/null && echo true || echo false)"

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
  if ! echo "${supported}" | grep -q "${OS}-${ARCH}"; then
    echo "No prebuilt binary for ${OS}-${ARCH}."
    echo "To build from source, go to https://github.com/$GITHUB_REPOSITORY"
    exit 1
  fi

  if [ "${HAS_CURL}" != "true" ] && [ "${HAS_WGET}" != "true" ]; then
    echo "Either curl or wget is required"
    exit 1
  fi

  if [ "${VERIFY_CHECKSUM}" == "true" ] && [ "${HAS_OPENSSL}" != "true" ]; then
    echo "In order to verify checksum, openssl must first be installed."
    echo "Please install openssl or set VERIFY_CHECKSUM=false in your environment."
    exit 1
  fi
}

# checkLatestVersion checks if the desired version is available.
checkLatestVersion() {
  if [ "$DESIRED_VERSION" == "" ]; then
    # Get tag from release URL
    local latest_release_url="https://api.github.com/repos/$GITHUB_REPOSITORY/releases/latest"
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

# checkSloctlInstalledVersion checks which version of sloctl is installed and
# if it needs to be changed.
checkSloctlInstalledVersion() {
  if [[ -f "${SLOCTL_INSTALL_DIR}/${BINARY_NAME}" ]]; then
    local version
    version=$("${SLOCTL_INSTALL_DIR}/${BINARY_NAME}" version)
    if [[ $version =~ sloctl/([0-9]+\.[0-9]+\.[0-9]+) ]]; then
      version="${BASH_REMATCH[1]}"
    fi
    if [[ "$version" == "$TAG" ]]; then
      echo "Sloctl ${version} is already ${DESIRED_VERSION:-latest}"
      return 0
    else
      echo "Sloctl ${TAG} is available. Changing from version ${version}."
      return 1
    fi
  else
    return 1
  fi
}

# downloadFile downloads the latest binary package and also the checksum
# for that binary.
downloadFile() {
  SLOCTL_DIST="sloctl-$TAG-$OS-$ARCH"
  DOWNLOAD_BASE_URL="https://github.com/$GITHUB_REPOSITORY/releases/download/$TAG"
  DOWNLOAD_URL="$DOWNLOAD_BASE_URL/$SLOCTL_DIST"
  CHECKSUM_URL="$DOWNLOAD_BASE_URL/sloctl-$TAG.sha256"
  SLOCTL_TMP_ROOT="$(mktemp -dt sloctl-installer-XXXXXX)"
  SLOCTL_TMP_BIN="$SLOCTL_TMP_ROOT/sloctl"
  SLOCTL_SUM_FILE="$SLOCTL_TMP_ROOT/sloctl-$TAG.sha256"
  echo "Downloading $DOWNLOAD_URL"
  if [ "${HAS_CURL}" == "true" ]; then
    curl -SsL --fail "$DOWNLOAD_URL" -o "$SLOCTL_TMP_BIN"
  elif [ "${HAS_WGET}" == "true" ]; then
    wget -q -O "$SLOCTL_TMP_BIN" "$DOWNLOAD_URL"
  fi
  echo "Downloading checksum $CHECKSUM_URL"
  if [ "${HAS_CURL}" == "true" ]; then
    curl -SsL --fail "$CHECKSUM_URL" -o "$SLOCTL_SUM_FILE"
  elif [ "${HAS_WGET}" == "true" ]; then
    wget -q -O "$SLOCTL_SUM_FILE" "$CHECKSUM_URL"
  fi
}

# verifyFile verifies the SHA256 checksum of the binary package
# and the GPG signatures for both the package and checksum file
# (depending on settings in environment).
verifyFile() {
  if [ "${VERIFY_CHECKSUM}" == "true" ]; then
    verifyChecksum
  fi
}

# installFile installs the sloctl binary.
installFile() {
  echo "Preparing to install $BINARY_NAME into ${SLOCTL_INSTALL_DIR}"
  runAsRoot cp "$SLOCTL_TMP_BIN" "$SLOCTL_INSTALL_DIR/$BINARY_NAME"
  echo "$BINARY_NAME installed into $SLOCTL_INSTALL_DIR/$BINARY_NAME"
}

# verifyChecksum verifies the SHA256 checksum of the binary package.
verifyChecksum() {
  printf "Verifying checksum... "
  local actual_sum
  local expected_sum
  actual_sum=$(openssl sha1 -sha256 "${SLOCTL_TMP_BIN}" | awk '{print $2}')
  expected_sum=$(cat "${SLOCTL_SUM_FILE}")
  if [ "$actual_sum" != "$expected_sum" ]; then
    echo "SHA sum of ${SLOCTL_TMP_BIN} does not match. Aborting."
    exit 1
  fi
  echo "Done."
}

# fail_trap is executed if an error occurs.
fail_trap() {
  result=$?
  if [ "$result" != "0" ]; then
    if [[ -n "$INPUT_ARGUMENTS" ]]; then
      echo "Failed to install $BINARY_NAME with the arguments provided: $INPUT_ARGUMENTS"
      help
    else
      echo "Failed to install $BINARY_NAME"
    fi
    echo -e "\tFor support, go to https://github.com/nobl9/sloctl."
  fi
  cleanup
  exit $result
}

# testVersion tests the installed client to make sure it is working.
testVersion() {
  set +e
  command -v "$BINARY_NAME"
  if [ "$?" = "1" ]; then
    echo "$BINARY_NAME not found. Is $SLOCTL_INSTALL_DIR on your '\$PATH?'"
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
It can be used to both install the binary and upgrade an existing ${PROGRAM_NAME} version.
OPTS:
  -h, --help                          Print this message
  --version, -v <desired_version>     When not defined it fetches the latest release from GitHub
  --no-sudo                           Install without sudo
ENV VARIABLES:
  BINARY_NAME           Installed binary name, defaults to ${PROGRAM_NAME}
  USE_SUDO              Whether to install with sudo, defaults to true
  DEBUG                 Print additional debug information, defaults to false
  VERIFY_CHECKSUM       Verify the SHA256 checksum of the binary, defaults to true
  SLOCTL_INSTALL_DIR    Install directory, defaults to /usr/local/bin
Examples:
  SLOCTL_INSTALL_DIR=/home/me/go/bin ${script_name} --no-sudo --version=v0.10.0
EOF
}

# cleanup temporary files.
cleanup() {
  if [[ -d "${SLOCTL_TMP_ROOT:-}" ]]; then
    rm -rf "$SLOCTL_TMP_ROOT"
  fi
}

# Execution.

# Stop execution on any error.
trap "fail_trap" EXIT
set -e

# Set debug if desired.
if [ "${DEBUG}" == "true" ]; then
  set -x
fi

# Parsing input arguments (if any).
export INPUT_ARGUMENTS="${*}"
set -u
while [[ $# -gt 0 ]]; do
  case $1 in
  '--version' | -v)
    shift
    if [[ $# -ne 0 ]]; then
      export DESIRED_VERSION="${1}"
      if [[ "$1" != "v"* ]]; then
        echo "Expected version arg ('${DESIRED_VERSION}') to begin with 'v', fixing..."
        export DESIRED_VERSION="v${1}"
      fi
    else
      echo -e "Please provide the desired version. e.g. --version v0.10.0"
      exit 0
    fi
    ;;
  '--no-sudo')
    USE_SUDO="false"
    ;;
  '--help' | -h)
    help
    exit 0
    ;;
  *)
    exit 1
    ;;
  esac
  shift
done
set +u

# Run.
initArch
initOS
verifySupported
checkLatestVersion
if ! checkSloctlInstalledVersion; then
  downloadFile
  verifyFile
  installFile
fi
testVersion
cleanup
