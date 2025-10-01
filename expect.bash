#!/usr/bin/env bash

set -e

export SLOCTL_ACCESSIBLE_MODE=1
export NO_COLOR=1

expect -c '
  set timeout 5
  spawn sloctl config use-context
  expect {
    "Select the new context:" {
      send "2\r"
      expect "Switched to context*"
    }
    timeout {
      puts "Timeout waiting for prompt"
      exit 1
    }
  }
  expect eof
'
