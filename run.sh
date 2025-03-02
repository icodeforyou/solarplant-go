#!/bin/bash
set -e

go build -o ./bin/solarplant ./main.go

set -a
source .env
set +a
./bin/solarplant