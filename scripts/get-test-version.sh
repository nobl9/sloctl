#!/usr/bin/env bash

latest_tag=$(git tag |
  grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' |
  sort -V |
  tail -n1)

echo "${latest_tag#v}-test"
