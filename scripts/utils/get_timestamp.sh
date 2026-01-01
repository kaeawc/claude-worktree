#!/usr/bin/env bash

# Cross-platform millisecond timestamp
if [[ "$OSTYPE" == "darwin"* ]]; then
  if command -v gdate &>/dev/null; then
    gdate +%s%3N
  else
    echo "On MacOS you need to run 'brew install coreutils' to have gdate" >&2
    exit 1
  fi
else
  date +%s%3N
fi
