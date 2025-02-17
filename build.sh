#!/bin/bash
set -e

go build -o ./bin/solarplant  ./main.go
go build -o ./bin/energy_forecast ./cmd/energy_forecast/main.go
go build -o ./bin/energy_price ./cmd/energy_price/main.go
