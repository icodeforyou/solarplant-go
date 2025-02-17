#!/bin/bash
set -e

go build -o ./bin/solarplant  ./main.go
env $(cat .env | xargs) ./bin/solarplant