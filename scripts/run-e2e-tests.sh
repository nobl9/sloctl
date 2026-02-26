#!/usr/bin/env bash
# Run e2e tests in Docker with environment variables extracted from current context.
# Environment variables take precedence over config file values.
#
# Usage:
#   ./scripts/run-e2e-tests.sh <docker-image> <revision>

set -eo pipefail

# Arguments
DOCKER_IMAGE="${1:-sloctl-bats-e2e}"
REVISION="${2:-undefined}"

# Docker image containing sloctl binary for config extraction
SLOCTL_IMAGE="${SLOCTL_IMAGE:-sloctl-e2e-test-bin}"

# Default config path (can be overridden by SLOCTL_CONFIG_FILE_PATH env var)
CONFIG_PATH="${SLOCTL_CONFIG_FILE_PATH:-${HOME}/.config/nobl9/config.toml}"

# Extract context config with secrets visible if not all env vars are set
if [ -z "$SLOCTL_CLIENT_ID" ] || [ -z "$SLOCTL_CLIENT_SECRET" ] || \
   [ -z "$SLOCTL_OKTA_ORG_URL" ] || [ -z "$SLOCTL_OKTA_AUTH_SERVER" ]; then

  if [ -f "$CONFIG_PATH" ]; then
    echo "Extracting missing credentials from current context..."
    CONTEXT_CONFIG=$(docker run --rm \
      -v "$CONFIG_PATH:/config.toml:ro" \
      "$SLOCTL_IMAGE" \
      config current-context --verbose --show-secret -o json --config=/config.toml 2>/dev/null || echo "{}")
  else
    CONTEXT_CONFIG="{}"
  fi

  # Extract each variable if not already set, with env vars taking precedence
  if [ -z "$SLOCTL_CLIENT_ID" ]; then
    SLOCTL_CLIENT_ID=$(echo "$CONTEXT_CONFIG" | jq -r '.clientId // empty')
  fi

  if [ -z "$SLOCTL_CLIENT_SECRET" ]; then
    SLOCTL_CLIENT_SECRET=$(echo "$CONTEXT_CONFIG" | jq -r '.clientSecret // empty')
  fi

  if [ -z "$SLOCTL_OKTA_ORG_URL" ]; then
    SLOCTL_OKTA_ORG_URL=$(echo "$CONTEXT_CONFIG" | jq -r '.oktaOrgUrl // empty')
  fi

  if [ -z "$SLOCTL_OKTA_AUTH_SERVER" ]; then
    SLOCTL_OKTA_AUTH_SERVER=$(echo "$CONTEXT_CONFIG" | jq -r '.oktaAuthServer // empty')
  fi
fi

# Validate required variables
if [ -z "$SLOCTL_CLIENT_ID" ] || [ -z "$SLOCTL_CLIENT_SECRET" ]; then
  echo "Error: SLOCTL_CLIENT_ID and SLOCTL_CLIENT_SECRET must be set or available in current context" >&2
  exit 1
fi

# Run e2e tests in Docker
docker run --rm \
  -e SLOCTL_CLIENT_ID="$SLOCTL_CLIENT_ID" \
  -e SLOCTL_CLIENT_SECRET="$SLOCTL_CLIENT_SECRET" \
  -e SLOCTL_OKTA_ORG_URL="$SLOCTL_OKTA_ORG_URL" \
  -e SLOCTL_OKTA_AUTH_SERVER="$SLOCTL_OKTA_AUTH_SERVER" \
  -e SLOCTL_GIT_REVISION="$REVISION" \
  "$DOCKER_IMAGE" -F pretty --filter-tags e2e ./test/*
