#!/bin/bash
set -e

./build.sh

set -a
source .env
set +a
./bin/solarplant