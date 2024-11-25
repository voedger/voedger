#!/bin/bash
set -euo pipefail
vpm build
rm -rf ../image
unzip -o "src.var" -d "../"
rm -f src.var
