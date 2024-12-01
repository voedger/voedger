#!/bin/bash

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "$REPO_ROOT"

# Iterate through all subfolders with their own go.mod and run tests
for dir in $(find . -name "go.mod" -not -path "./go.mod" -exec dirname {} \;); do
    echo "Running tests in $dir..."
    
    # Run tests normally in all other directories
    (cd "$dir" && go test ./... --short)
done

