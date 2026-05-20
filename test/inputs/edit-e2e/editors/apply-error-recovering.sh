#!/usr/bin/env bash

set -euo pipefail

state_file="${0}.state"

if [[ ! -e "$state_file" ]]; then
  touch "$state_file"
  yq -Y -i '
    if type == "array" then
      .[0].metadata.labels."orig in" = ["sdk"]
    else
      .metadata.labels."orig in" = ["sdk"]
    end
  ' "$1"
  exit 0
fi

first_line="$(sed -n '1p' "$1")"
if [[ "$first_line" != "# Please edit the object below. Lines beginning with a '#' will be ignored," ]]; then
  printf "expected edit notice at top, got: %s\n" "$first_line" >&2
  exit 25
fi

contents="$(< "$1")"
if [[ "$contents" != *"# The edited file had an error: Validation for Project"* ]]; then
  printf "expected apply validation error in reopened edit file\n" >&2
  exit 26
fi
if [[ "$contents" == *"Manifest source"* ]]; then
  printf "did not expect manifest source details in reopened edit file\n" >&2
  exit 27
fi
if [[ "$contents" == *"traceId:"* || "$contents" == *"endpoint:"* ]]; then
  printf "did not expect API transport details in reopened edit file\n" >&2
  exit 28
fi

yq -Y -i '
  if type == "array" then
    del(.[0].metadata.labels."orig in") |
    .[0].spec.description = "Recovered after apply error"
  else
    del(.metadata.labels."orig in") |
    .spec.description = "Recovered after apply error"
  end
' "$1"
