#!/usr/bin/env bash

set -euo pipefail

state_file="${0}.state"

if [[ ! -e "$state_file" ]]; then
  touch "$state_file"
  yq -Y -i '
    if type == "array" then
      .[0].metadata.name = "renamed-edit-target"
    else
      .metadata.name = "renamed-edit-target"
    end
  ' "$1"
fi
