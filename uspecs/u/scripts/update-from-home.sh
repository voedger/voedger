#!/usr/bin/env bash
set -Eeuo pipefail

# update-from-home.sh
#
# Description:
#   Updates the current uspecs project with the latest files from the USPECS_HOME directory.
#   This script synchronizes uspecs/u content and configuration files (AGENTS.md, CLAUDE.md)
#   from a central uspecs home location to the current project.
#
# Usage:
#   ./update-from-home.sh [target-directory]
#
# Arguments:
#   target-directory - Optional path to the uspecs/u directory to update.
#                      If not provided, uses the directory where this script is located.
#
# Prerequisites:
#   - USPECS_HOME environment variable must be set
#   - USPECS_HOME must point to a directory that contains the 'uspecs' folder
#   - The directory structure should be: $USPECS_HOME/uspecs/u/

# Check if USPECS_HOME environment variable is set
if [[ -z "${USPECS_HOME:-}" ]]; then
    echo "Error: USPECS_HOME environment variable is not set" >&2
    echo "" >&2
    echo "Please set USPECS_HOME to the directory that contains the 'uspecs' folder." >&2
    echo "Example (USPECS_HOME should contain uspecs/ as a subdirectory):" >&2
    echo "  export USPECS_HOME=/path/to/seeai0/uspecs0" >&2
    echo "" >&2
    echo "You can add this to your shell profile (~/.bashrc, ~/.zshrc, etc.) to make it permanent." >&2
    exit 1
fi

# Verify USPECS_HOME exists
if [[ ! -d "$USPECS_HOME" ]]; then
    echo "Error: USPECS_HOME directory does not exist: $USPECS_HOME" >&2
    echo "" >&2
    echo "Please verify that USPECS_HOME points to a valid directory." >&2
    exit 1
fi

# Set source and target directories based on USPECS_HOME
SOURCE_DIR="$USPECS_HOME/uspecs/u"

# Use provided target directory or default to script's parent directory
if [[ $# -ge 1 ]]; then
    TARGET_DIR="$1"
else
    TARGET_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fi

# Verify source directory exists
if [[ ! -d "$SOURCE_DIR" ]]; then
    echo "Error: Source directory does not exist: $SOURCE_DIR" >&2
    echo "" >&2
    echo "Expected to find uspecs/u under USPECS_HOME ($USPECS_HOME)" >&2
    echo "USPECS_HOME should point to a directory that contains the 'uspecs' folder." >&2
    echo "Please verify your USPECS_HOME setting." >&2
    exit 1
fi

# Check if source and target are the same location
SOURCE_DIR_NORMALIZED="$(cd "$SOURCE_DIR" && pwd)"
TARGET_DIR_NORMALIZED="$(cd "$TARGET_DIR" && pwd)"

if [[ "$SOURCE_DIR_NORMALIZED" == "$TARGET_DIR_NORMALIZED" ]]; then
    echo "Error: Cannot update USPECS_HOME from itself" >&2
    echo "" >&2
    echo "You are trying to run this script in the USPECS_HOME location." >&2
    echo "This script should be run in a different project that you want to update," >&2
    echo "not in the USPECS_HOME location itself." >&2
    echo "" >&2
    echo "Current location: $TARGET_DIR_NORMALIZED" >&2
    echo "USPECS_HOME: $SOURCE_DIR_NORMALIZED" >&2
    echo "" >&2
    echo "Please run this script from a different project directory." >&2
    exit 1
fi

echo "Source directory: $SOURCE_DIR"
echo "Target directory: $TARGET_DIR"
echo ""

# Step 1: Collect all files from source
echo "Collecting files from source..."
source_files=()
pushd "$SOURCE_DIR" > /dev/null
while IFS= read -r -d '' file; do
    file="${file#./}"
    source_files+=("$file")
done < <(find . -type f -print0)
popd > /dev/null
echo "  Found ${#source_files[@]} file(s) in source"
echo ""

# Step 2: Collect all files from target
echo "Collecting files from target..."
target_files=()
if [[ -d "$TARGET_DIR" ]]; then
    pushd "$TARGET_DIR" > /dev/null
    while IFS= read -r -d '' file; do
        file="${file#./}"
        target_files+=("$file")
    done < <(find . -type f -print0)
    popd > /dev/null
fi
echo "  Found ${#target_files[@]} file(s) in target"
echo ""

# Step 3: Determine files to remove (in target but not in source)
echo "Determining files to remove..."
files_to_remove=()
for target_file in "${target_files[@]}"; do
    # Check if file exists in source
    found=0
    for source_file in "${source_files[@]}"; do
        if [[ "$source_file" == "$target_file" ]]; then
            found=1
            break
        fi
    done

    if [[ $found -eq 0 ]]; then
        files_to_remove+=("$target_file")
    fi
done
echo "  ${#files_to_remove[@]} file(s) to remove"
echo ""

# Step 4: Remove files
if [[ ${#files_to_remove[@]} -gt 0 ]]; then
    echo "Removing files..."
    for file in "${files_to_remove[@]}"; do
        echo "  Removing: $file"
        rm -f "$TARGET_DIR/$file" || echo "  Warning: Could not remove $file (may be busy)" >&2
    done
    echo ""
fi

# Step 5: Remove empty directories
echo "Removing empty directories..."
removed_dirs=0
if [[ -d "$TARGET_DIR" ]]; then
    pushd "$TARGET_DIR" > /dev/null
    while IFS= read -r -d '' dir; do
        dir="${dir#./}"
        [[ -z "$dir" ]] && continue
        if [[ ! -d "$SOURCE_DIR/$dir" ]]; then
            echo "  Removing directory: $dir"
            rm -rf "${TARGET_DIR:?}/${dir:?}" || echo "  Warning: Could not remove directory $dir (may be busy)" >&2
            ((++removed_dirs))
        fi
    done < <(find . -depth -type d -print0)
    popd > /dev/null
fi
if [[ $removed_dirs -eq 0 ]]; then
    echo "  No directories to remove"
fi
echo ""

# Step 6: Copy all files from source to target
echo "Copying files from source to target..."
copied_count=0
for file in "${source_files[@]}"; do
    target_file="$TARGET_DIR/$file"

    # Create parent directory if needed
    mkdir -p "$(dirname "$target_file")"

    # Copy the file
    cp -f "$SOURCE_DIR/$file" "$target_file"
    ((++copied_count))
done
echo "  Copied $copied_count file(s)"
echo ""

echo "Successfully synchronized $SOURCE_DIR to $TARGET_DIR"
echo ""

# Copy AGENTS.md and CLAUDE.md from USPECS_HOME to target parent directory
SOURCE_ROOT="$USPECS_HOME"
TARGET_ROOT="$(dirname "$(dirname "$TARGET_DIR")")"

if [[ -f "$SOURCE_ROOT/AGENTS.md" ]]; then
    cp -f "$SOURCE_ROOT/AGENTS.md" "$TARGET_ROOT/"
    echo "Successfully copied AGENTS.md to $TARGET_ROOT"
else
    echo "Warning: AGENTS.md not found at $SOURCE_ROOT/AGENTS.md" >&2
fi

if [[ -f "$SOURCE_ROOT/CLAUDE.md" ]]; then
    cp -f "$SOURCE_ROOT/CLAUDE.md" "$TARGET_ROOT/"
    echo "Successfully copied CLAUDE.md to $TARGET_ROOT"
else
    echo "Warning: CLAUDE.md not found at $SOURCE_ROOT/CLAUDE.md" >&2
fi

