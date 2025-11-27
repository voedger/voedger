#!/usr/bin/env bash
set -Eeuo pipefail

find . -name "go.mod" -type f | while read -r modfile; do
    dir=$(dirname "$modfile")
    echo "Processing directory: $dir"
    (cd "$dir" && go get github.com/voedger/voedger@main && go mod tidy)
done