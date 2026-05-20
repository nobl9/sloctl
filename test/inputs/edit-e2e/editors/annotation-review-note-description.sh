#!/usr/bin/env bash

set -euo pipefail

actual_names="$(yq -r '[if type == "array" then .[] else . end | select(.kind == "Annotation") | .metadata.name] | sort | join(" ")' "$1")"
expected_names="edit-annotation-secondary"

if [[ "$actual_names" != "$expected_names" ]]; then
  printf "expected edited annotations [%s], got [%s]\n" "$expected_names" "$actual_names" >&2
  exit 24
fi

yq -Y -i '
  if type == "array" then
    .[0].spec.description = "Edited by sloctl edit review note filters e2e"
  else
    .spec.description = "Edited by sloctl edit review note filters e2e"
  end
' "$1"
