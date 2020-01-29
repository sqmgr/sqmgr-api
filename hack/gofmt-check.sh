#!/bin/bash
BAD_FILES=$(
  find . -type f -name '*.go' |
    xargs gofmt -l
)

if [[ -n $BAD_FILES ]]; then
  echo "The following files fail gofmt:\n$BAD_FILES" >&2
  exit 1
fi