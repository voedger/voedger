#!/usr/bin/env bash
set -Eeuo pipefail
vpm build
rm -rf ../image/pkg
unzip -o "src.var" -d "../image"
rm -f src.var
