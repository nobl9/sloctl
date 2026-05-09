#!/usr/bin/env bash

set -euo pipefail

yq -Y -i '
  if type == "array" then
    .[0].metadata.displayName = "Edited by sloctl edit e2e"
  else
    .metadata.displayName = "Edited by sloctl edit e2e"
  end
' "$1"
