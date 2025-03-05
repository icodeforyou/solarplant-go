#!/bin/bash
set -e

COMMIT_HASH="$(git rev-parse --short HEAD)"
BUILD_TIMESTAMP=$(date '+%Y-%m-%dT%H:%M:%S')

LDFLAGS=(
  "-X 'main.CommitHash=$(git rev-parse --short HEAD)'"
  "-X 'main.BuildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ')'"
)

go build -ldflags="${LDFLAGS[*]}" -o ./bin/solarplant ./main.go