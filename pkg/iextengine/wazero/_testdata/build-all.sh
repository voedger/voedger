#!/usr/bin/env bash
set -Eeuo pipefail

cd ./allocs
./build.sh
cd ../basicusage
./build.sh
cd ../benchmarks
./build.sh
cd ../panics
./build.sh
cd ../tests/
./build.sh