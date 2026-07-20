#!/usr/bin/env bash

set -euo pipefail

yq -Y -i '
  if type == "array" then
    .[0].spec.description = "Edited by sloctl edit e2e"
  else
    .spec.description = "Edited by sloctl edit e2e"
  end
' "$1"
