#!/usr/bin/env bash

set -euo pipefail

yq -Y -i '
  if type == "array" then
    .[0].spec.roleRef = "project-editor" |
    del(.[0].spec.user)
  else
    .spec.roleRef = "project-editor" |
    del(.spec.user)
  end
' "$1"
