#!/usr/bin/env bash
set -Eeuo pipefail

# update-from-home-pr.sh
#
# Description:
#   Creates a branch, runs update-from-home.sh, commits changes, pushes to origin,
#   and opens a PR to the upstream repository.
#
# Usage:
#   ./update-from-home-pr.sh <target-directory>
#
# Arguments:
#   target-directory - Path to the git repository to update
#
# Prerequisites:
#   - upstream and origin remotes must exist in the target repository
#   - No uncommitted changes in target working directory
#   - GitHub CLI (gh) must be installed
#   - USPECS_HOME environment variable must be set

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Validate arguments
if [[ $# -ne 1 ]]; then
    echo "Error: Missing required argument" >&2
    echo "" >&2
    echo "Usage: $0 <target-directory>" >&2
    echo "" >&2
    echo "Example: $0 /path/to/project" >&2
    exit 1
fi

TARGET_DIR="$1"

# Verify target directory exists
if [[ ! -d "$TARGET_DIR" ]]; then
    echo "Error: Target directory does not exist: $TARGET_DIR" >&2
    exit 1
fi

# Change to target directory
echo "Changing to directory: $TARGET_DIR"
cd "$TARGET_DIR"

# Check if working directory is clean
if [[ -n $(git status --porcelain) ]]; then
    echo "Error: Working directory has uncommitted changes" >&2
    echo "Please commit or stash your changes before running this script." >&2
    exit 1
fi

# Verify upstream remote exists
if ! git remote | grep -q '^upstream$'; then
    echo "Error: 'upstream' remote does not exist" >&2
    echo "Please add the upstream remote first." >&2
    exit 1
fi

# Verify origin remote exists
if ! git remote | grep -q '^origin$'; then
    echo "Error: 'origin' remote does not exist" >&2
    echo "Please add the origin remote first." >&2
    exit 1
fi

# Verify GitHub CLI is installed
if ! command -v gh &> /dev/null; then
    echo "Error: GitHub CLI (gh) is not installed" >&2
    echo "Please install it from https://cli.github.com/" >&2
    exit 1
fi

# Switch to main branch
echo "Switching to main branch..."
git checkout main

# Update main from upstream with rebase
echo "Updating main from upstream..."
git fetch upstream
git rebase upstream/main

# Push updated main to origin
echo "Pushing updated main to origin..."
git push origin main

# Create branch name with timestamp
BRANCH_NAME="update-from-home-$(date +%y%m%d-%H%M%S)"

echo "Creating branch: $BRANCH_NAME"
git checkout -b "$BRANCH_NAME"

# Run update-from-home.sh with current directory's uspecs/u as target
echo "Running update-from-home.sh..."
bash "$SCRIPT_DIR/update-from-home.sh" "$PWD/uspecs/u"

# Check if there are any changes to commit
if [[ -z $(git status --porcelain) ]]; then
    echo "No changes to commit. Cleaning up..."
    git checkout -
    git branch -d "$BRANCH_NAME"
    echo "No updates were needed."
    exit 0
fi

# Commit changes
echo "Committing changes..."
git add -A
git commit -m "Update from USPECS_HOME"

# Push to origin
echo "Pushing branch to origin..."
git push -u origin "$BRANCH_NAME"

# Create PR to upstream using GitHub CLI
echo "Creating pull request to upstream..."
UPSTREAM_REPO="$(git remote get-url upstream | sed -E 's#.*github.com[:/]##; s#\.git$##')"
ORIGIN_OWNER="$(git remote get-url origin | sed -E 's#.*github.com[:/]##; s#\.git$##; s#/.*##')"
gh pr create --repo "$UPSTREAM_REPO" --base main --head "${ORIGIN_OWNER}:${BRANCH_NAME}" --title "Update from USPECS_HOME" --body "Automated update from USPECS_HOME"
echo "Pull request created successfully!"