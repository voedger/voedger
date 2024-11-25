#!/bin/bash
set -euo pipefail
vpm build
rm -rf ../image/pkg
unzip -o "src.var" -d "../image"
rm -f src.var
