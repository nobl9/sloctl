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

contents="$(< "$1")"
if [[ "$contents" != *"# The edited file had an error: Validation for Project"* ]]; then
  printf "expected apply validation error in reopened edit file\n" >&2
  exit 29
fi

yq -Y -i '
  if type == "array" then
    del(.[0].metadata.labels."orig in")
  else
    del(.metadata.labels."orig in")
  end
' "$1"
