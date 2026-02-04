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
#   - origin remote must exist (upstream remote is optional, will use origin if missing)
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

# Verify origin remote exists
if ! git remote | grep -q '^origin$'; then
    echo "Error: 'origin' remote does not exist" >&2
    echo "Please add the origin remote first." >&2
    exit 1
fi

# Determine PR remote: use upstream if exists, otherwise use origin
if git remote | grep -q '^upstream$'; then
    PR_REMOTE="upstream"
    echo "Using 'upstream' remote for PR target"
else
    PR_REMOTE="origin"
    echo "No 'upstream' remote found, using 'origin' for PR target"
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

# Update main from PR remote with rebase
echo "Updating main from $PR_REMOTE..."
git fetch "$PR_REMOTE"
git rebase "$PR_REMOTE/main"

# Push updated main to origin
echo "Pushing updated main to origin..."
git push origin main

# Run update-from-home.sh with current directory's uspecs/u as target
echo "Running update-from-home.sh..."
bash "$SCRIPT_DIR/update-from-home.sh" "$PWD/uspecs/u"

# Read version info for branch name, commit, and PR messages
VERSION_INFO=$(cat "$PWD/uspecs/version.txt" 2>/dev/null)
if [[ -z "$VERSION_INFO" ]]; then
    echo "Error: Failed to read version info from uspecs/version.txt" >&2
    exit 1
fi

BRANCH_NAME="update-uspecs-${VERSION_INFO}"

echo "Creating branch: $BRANCH_NAME"
git checkout -b "$BRANCH_NAME"

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
git commit -m "Update uspecs to ${VERSION_INFO}"

# Push to origin
echo "Pushing branch to origin..."
git push -u origin "$BRANCH_NAME"

# Create PR using GitHub CLI
echo "Creating pull request to $PR_REMOTE..."
PR_REPO="$(git remote get-url "$PR_REMOTE" | sed -E 's#.*github.com[:/]##; s#\.git$##')"
PR_BODY="Update uspecs/u from USPECS_HOME

Version: ${VERSION_INFO}"
PR_ARGS=('--repo' "$PR_REPO" '--base' 'main' '--title' "Update uspecs to ${VERSION_INFO}" '--body' "$PR_BODY")

if [[ "$PR_REMOTE" == "upstream" ]]; then
    # PR from fork to upstream
    ORIGIN_OWNER="$(git remote get-url origin | sed -E 's#.*github.com[:/]##; s#\.git$##; s#/.*##')"
    gh pr create "${PR_ARGS[@]}" --head "${ORIGIN_OWNER}:${BRANCH_NAME}"
else
    # PR within same repo (origin)
    gh pr create "${PR_ARGS[@]}" --head "$BRANCH_NAME"
fi
echo "Pull request created successfully!"

# Clean up local branch (remote branch remains for PR)
echo "Cleaning up local branch..."
git checkout main
git branch -d "$BRANCH_NAME"
echo "Done!"