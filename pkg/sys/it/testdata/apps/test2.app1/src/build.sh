#!/bin/bash
# Copyright (c) 2024-present unTill Software Development Group B.V.
# @author Denis Gribanov

set -euo pipefail

readonly target_dir="../image/pkg"

TEMP_DIR=$(mktemp -d)
vpm build
unzip "src.var" -d "$TEMP_DIR"
if [ -d "$target_dir" ]; then
    rm -rf "$target_dir"
fi
mkdir -p "$target_dir"
mv "$TEMP_DIR/build/"* "$target_dir"
rm -f src.var
