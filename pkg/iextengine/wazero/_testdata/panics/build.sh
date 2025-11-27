#!/usr/bin/env bash
set -Eeuo pipefail

tinygo build --no-debug -o pkg.wasm -scheduler=none -gc=leaking -print-allocs=. -opt=2 -target=wasi .