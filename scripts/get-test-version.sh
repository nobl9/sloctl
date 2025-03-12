#!/usr/bin/env bash

set -e pipefail

latest_tag=$(git tag |
  grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' |
  sort -V |
  tail -n1)

echo "${latest_tag#v}-test"
