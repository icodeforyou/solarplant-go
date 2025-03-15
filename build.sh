#!/bin/bash
set -e

LDFLAGS=(
  "-X 'main.CommitHash=$(git rev-parse --short HEAD)'"
  "-X 'main.BuildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ')'"
)

VERSION=$(git describe --tags --always --dirty)
LDFLAGS=("-X 'main.Version=$VERSION'")

go build -ldflags="${LDFLAGS[*]}" -o ./bin/solarplant ./main.go