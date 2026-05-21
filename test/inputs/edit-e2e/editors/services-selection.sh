#!/usr/bin/env bash

set -euo pipefail

actual_names="$(yq -r '[if type == "array" then .[] else . end | select(.kind == "Service") | .metadata.name] | sort | join(" ")' "$1")"
expected_names="edit-target edit-target-secondary"

if [[ "$actual_names" != "$expected_names" ]]; then
  printf "expected edited services [%s], got [%s]\n" "$expected_names" "$actual_names" >&2
  exit 24
fi
