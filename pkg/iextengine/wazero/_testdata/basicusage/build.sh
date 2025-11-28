#!/usr/bin/env bash
set -Eeuo pipefail

#--wasm-abi=generic is needed to provide support of uint64 in WASM functions parameters
# By default parameters must be int32 because it supposed to work with javascript.

tinygo build --no-debug -o pkg.wasm -scheduler=none -gc=leaking -opt=2 -target=wasi .
tinygo build --no-debug -o pkggc.wasm -scheduler=none -opt=2 -print-allocs=. -target=wasi .
