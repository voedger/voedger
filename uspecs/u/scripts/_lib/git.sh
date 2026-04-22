#!/usr/bin/env bash

# Well, we do not neeed it, since it is sourced, just for consistency with other scripts
set -Eeuo pipefail

# git.sh -- Git branch and pull request automation
#
# Provides reusable functions for the PR workflow: branch creation from a
# remote default branch, and PR submission via GitHub CLI.
# Intended to be sourced, not executed directly.
#
# Concepts:
#   pr_remote   The remote that owns the target branch for PRs.
#               "upstream" when a fork setup is detected, otherwise "origin".
#   change_branch  The current working branch (named {change-name}).
#   pr_branch      The squashed PR branch (named {change-name}--pr).



# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

# shellcheck source=utils.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/utils.sh"

get_project_dir() {
    local script_dir
    script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
    # _lib/ -> scripts/ -> u/ -> uspecs/ -> project root
    cd "$script_dir/../../../.." && pwd
}

# // TODO why here?
read_conf_param() {
    local param_name="$1"
    local conf_file
    conf_file="$(get_project_dir)/uspecs/u/conf.md"

    if [ ! -f "$conf_file" ]; then
        error "conf.md not found: $conf_file"
    fi

    local line raw
    line=$(grep -E "^- ${param_name}:" "$conf_file" | head -1 || true)
    raw="${line#*: }"

    if [ -z "$raw" ]; then
        error "Parameter '${param_name}' not found in conf.md"
    fi

    local value
    value=$(echo "$raw" | sed 's/^[[:space:]`]*//' | sed 's/[[:space:]`]*$//')
    echo "$value"
}

determine_pr_remote() {
    if git remote | grep -q '^upstream$'; then
        echo "upstream"
    else
        echo "origin"
    fi
}

gh_create_pr() {
    # Usage: printf '%s' "$body" | gh_create_pr <pr_remote> <default_branch> <head_branch> <title>
    # Creates a PR via GitHub CLI and outputs the PR URL. Reads body from stdin.
    local pr_remote="$1" default_branch="$2" head_branch="$3" title="$4"

    local pr_repo
    pr_repo="$(git remote get-url "$pr_remote" | sed -E 's#.*github.com[:/]##; s#\.git$##')"
    local pr_args=('--repo' "$pr_repo" '--base' "$default_branch" '--title' "$title" '--body-file' '-')

    if [[ "$pr_remote" == "upstream" ]]; then
        local origin_owner
        origin_owner="$(git remote get-url origin | sed -E 's#.*github.com[:/]##; s#\.git$##; s#/.*##')"
        gh pr create "${pr_args[@]}" --head "${origin_owner}:${head_branch}"
    else
        gh pr create "${pr_args[@]}" --head "$head_branch"
    fi
}

# git_validate_working_tree
# Reflects scenario: "Project inside Git working tree"
# Validates that the current directory is inside a git working tree.
git_validate_working_tree() {
    if ! is_git_repo "$PWD"; then
        error "No git repository found at $PWD"
    fi
}

# git_validate_clean_repo <current_branch> <default_branch>
# Reflects scenario: "Git working tree is clean"
# Validates: inside git working tree, no uncommitted changes, not on default branch.
git_validate_clean_repo() {
    local current_branch="$1"
    local default_branch="$2"

    git_validate_working_tree

    if [[ -n $(git status --porcelain) ]]; then
        error "Working directory has uncommitted changes. Commit or stash changes first"
    fi

    if [[ "$current_branch" == "$default_branch" ]]; then
        error "Current branch is the default branch '$default_branch'"
    fi
}

check_prerequisites() {
    # Check if git repository exists
    if ! is_git_repo "$PWD"; then
        error "No git repository found at $PWD"
    fi

    # Check if GitHub CLI is installed
    if ! command -v gh &> /dev/null; then
        error "GitHub CLI (gh) is not installed. Install from https://cli.github.com/"
    fi

    # Check if origin remote exists
    if ! git remote | grep -q '^origin$'; then
        error "'origin' remote does not exist"
    fi

    # Check if working directory is clean
    if [[ -n $(git status --porcelain) ]]; then
        error "Working directory has uncommitted changes. Commit or stash changes first"
    fi
}

git_default_branch_name() {
    local branch
    branch=$(git ls-remote --symref origin HEAD | awk '/^ref:/ {sub(/refs\/heads\//, "", $2); print $2}') || {
        error "Cannot determine the default branch from remote"
    }
    if [[ -z "$branch" ]]; then
        error "Cannot determine the default branch from remote"
    fi
    echo "$branch"
}

# ---------------------------------------------------------------------------
# Public functions
# ---------------------------------------------------------------------------

# git_pr_info <map_nameref> [project_dir]
# Populates an associative array with PR remote info.
# Keys populated: pr_remote, default_branch
# project_dir: directory to run git commands from (defaults to $PWD)
# Returns non-zero if info cannot be determined.
git_pr_info() {
    local -n _git_pr_info_map="$1"
    local project_dir="${2:-$PWD}"
    local pr_remote default_branch
    pr_remote=$(cd "$project_dir" && determine_pr_remote) || return 1
    default_branch=$(cd "$project_dir" && git_default_branch_name) || return 1
    _git_pr_info_map["pr_remote"]="$pr_remote"
    _git_pr_info_map["default_branch"]="$default_branch"
}

# git_prbranch <name>
# Fetch pr_remote and create a local branch from its default branch.
git_prbranch() {
    local name="${1:-}"
    [[ -z "$name" ]] && error "Usage: git_prbranch <name>"

    local pr_remote default_branch
    pr_remote=$(determine_pr_remote)
    default_branch=$(git_default_branch_name)

    echo "Fetching $pr_remote/$default_branch..."
    git fetch "$pr_remote" "$default_branch" 2>&1

    echo "Creating branch: $name"
    git checkout -b "$name" "$pr_remote/$default_branch"
}

# git_ffdefault
# Fetch pr_remote/default_branch and fast-forward the local default branch to it.
# Switches to the default branch if not already on it, and leaves there after completion.
# Fail fast if any of the following conditions are true:
#     working directory is not clean
#     branches have diverged (fast-forward not possible)
git_ffdefault() {
    check_prerequisites

    local pr_remote default_branch
    pr_remote=$(determine_pr_remote)
    default_branch=$(git_default_branch_name)

    local current_branch
    current_branch=$(git symbolic-ref --short HEAD)

    if [[ "$current_branch" != "$default_branch" ]]; then
        echo "Switching to '$default_branch'..."
        git checkout "$default_branch"
    fi

    echo "Fetching $pr_remote/$default_branch..."
    git fetch "$pr_remote" "$default_branch" 2>&1

    echo "Fast-forwarding $default_branch..."
    if ! git merge --ff-only "$pr_remote/$default_branch" 2>&1; then
        error "Cannot fast-forward '$default_branch' to '$pr_remote/$default_branch'. The branches have diverged."
    fi
}

# git_pr --title <title> --body <body> --next-branch <branch> [--delete-branch]
# Literal \n sequences in --body are decoded to actual newlines.
# Stage all changes, commit, push to origin, and open a PR against
# pr_remote's default branch. Switch to --next-branch afterwards.
# If --delete-branch is set, delete the current branch after switching.
# If no changes exist, switch to --next-branch and exit cleanly.
git_pr() {
    local title="" body="" next_branch="" delete_branch=false
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --title)         title="$2";       shift 2 ;;
            --body)          body="$2";        shift 2 ;;
            --next-branch)   next_branch="$2"; shift 2 ;;
            --delete-branch) delete_branch=true; shift ;;
            *) error "Unknown flag: $1" ;;
        esac
    done
    [[ -z "$title" ]]       && error "--title is required"
    [[ -z "$body" ]]        && error "--body is required"
    [[ -z "$next_branch" ]] && error "--next-branch is required"

    # Decode literal \n sequences to actual newlines
    body="${body//\\n/$'\n'}"

    local default_branch branch_name
    default_branch=$(git_default_branch_name)
    branch_name=$(git symbolic-ref --short HEAD)

    if [[ "$delete_branch" == "true" && "$branch_name" == "$next_branch" ]]; then
        error "Cannot delete branch '$branch_name' because it is the same as --next-branch"
    fi

    # Nothing to commit -- switch to next branch and exit
    if [[ -z $(git status --porcelain) ]]; then
        echo "No changes to commit. Cleaning up..."
        git checkout "$next_branch"
        if [[ "$delete_branch" == "true" ]]; then
            git branch -d "$branch_name"
        fi
        echo "No updates were needed."
        return 0
    fi

    local pr_remote
    pr_remote=$(determine_pr_remote)

    echo "Committing changes..."
    git add -A
    git commit -m "$title"

    echo "Pushing branch to origin..."
    git push -u origin "$branch_name"

    echo "Creating pull request to $pr_remote..."
    local pr_url
    pr_url=$(printf '%s' "$body" | gh_create_pr "$pr_remote" "$default_branch" "$branch_name" "$title")
    echo "Pull request created successfully!"

    echo "Switching to $next_branch..."
    git checkout "$next_branch"
    if [[ "$delete_branch" == "true" ]]; then
        echo "Deleting local branch $branch_name..."
        git branch -d "$branch_name"
        echo "Deleting local reference to remote branch..."
        git branch -dr "origin/$branch_name"
    fi

    # Output PR info for caller to parse (to stderr so it doesn't interfere with normal output)
    echo "PR_URL=$pr_url" >&2
    echo "PR_BRANCH=$branch_name" >&2
    echo "PR_BASE=$default_branch" >&2
}

# git_mergedef
# Validate preconditions, fetch pr_remote/default_branch, and merge it into the current branch.
# On success outputs:
#     change_branch=<name>
#     default_branch=<name>
#     change_branch_head=<sha>  (HEAD before the merge)
git_mergedef() {
    check_prerequisites

    local pr_remote default_branch current_branch
    pr_remote=$(determine_pr_remote)
    default_branch=$(git_default_branch_name)
    current_branch=$(git symbolic-ref --short HEAD)

    if [[ "$current_branch" == "$default_branch" ]]; then
        error "Current branch '$current_branch' is the default branch; cannot create PR from it"
    fi

    if [[ "$current_branch" == *--pr ]]; then
        error "Current branch '$current_branch' ends with '--pr'; cannot create PR from a PR branch"
    fi

    local change_branch_head
    change_branch_head=$(git rev-parse HEAD)

    echo "Fetching $pr_remote/$default_branch..."
    git fetch "$pr_remote" "$default_branch" 2>&1

    echo "Merging $pr_remote/$default_branch into $current_branch..."
    git merge "$pr_remote/$default_branch" 2>&1

    echo "change_branch=$current_branch"
    echo "default_branch=$default_branch"
    echo "change_branch_head=$change_branch_head"
}

# git_diff <target>
# Output git diff of the target folder between HEAD and pr_remote/default_branch.
# Available targets: specs
git_diff() {
    local target="${1:-}"
    [[ -z "$target" ]] && error "Usage: git_diff <target>"

    local diff_path
    case "$target" in
        specs)
            diff_path=$(read_conf_param "specs_folder")
            ;;
        *)
            error "Unknown diff target: $target. Available: specs"
            ;;
    esac

    local pr_remote default_branch
    pr_remote=$(determine_pr_remote)
    default_branch=$(git_default_branch_name)

    local project_dir
    project_dir=$(get_project_dir)

    git fetch "$pr_remote" "$default_branch" >/dev/null 2>&1 || true
    (cd "$project_dir" && git diff "$pr_remote/$default_branch" HEAD -- "$diff_path")
}

# git_changepr --title <title> --body <body>
# Literal \n sequences in --body are decoded to actual newlines.
# Create a PR from the current change_branch:
#   - Fail fast if pr_branch ({change_branch}--pr) already exists.
#   - Create pr_branch from pr_remote/default_branch.
#   - Squash-merge change_branch into pr_branch and commit with title.
#   - Push pr_branch to origin and create a PR via GitHub CLI.
#   - Delete change_branch (locally, tracking ref, and remote; skip if absent).
#   - Output pr_url on success.
#   - On failure after pr_branch creation: roll back pr_branch, preserve change_branch.
git_changepr() {
    local title="" body=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --title) title="$2"; shift 2 ;;
            --body)  body="$2";  shift 2 ;;
            *) error "Unknown flag: $1" ;;
        esac
    done
    [[ -z "$title" ]] && error "--title is required"
    if [[ -z "$body" ]]; then
        if is_tty; then
            error "--body is required (or pipe body via stdin)"
        fi
        body=$(cat)
    fi
    [[ -z "$body" ]] && error "--body is required (or pipe body via stdin)"

    # Decode literal \n sequences to actual newlines
    body="${body//\\n/$'\n'}"

    local pr_remote default_branch change_branch pr_branch
    pr_remote=$(determine_pr_remote)
    default_branch=$(git_default_branch_name)
    change_branch=$(git symbolic-ref --short HEAD)
    pr_branch="${change_branch}--pr"

    # Fail fast if pr_branch already exists
    if git show-ref --verify --quiet "refs/heads/$pr_branch"; then
        error "PR branch '$pr_branch' already exists"
    fi

    # Create pr_branch from pr_remote/default_branch
    git_prbranch "$pr_branch"

    # Rollback pr_branch on failure; preserve change_branch
    local success=false
    rollback_pr_branch() {
        if [[ "$success" != "true" ]]; then
            echo "Rolling back: removing pr_branch '$pr_branch'..." >&2
            git checkout "$change_branch" 2>/dev/null || true
            git branch -D "$pr_branch" 2>/dev/null || true
            git push origin --delete "$pr_branch" 2>/dev/null || true
            git branch -dr "origin/$pr_branch" 2>/dev/null || true
        fi
    }
    trap rollback_pr_branch ERR

    # Squash-merge change_branch into pr_branch and commit
    echo "Squash-merging $change_branch into $pr_branch..."
    git merge --squash "$change_branch"
    git commit -m "$title"

    # Push pr_branch to origin
    echo "Pushing $pr_branch to origin..."
    git push -u origin "$pr_branch"

    # Create PR via GitHub CLI
    echo "Creating pull request..."
    local pr_url
    pr_url=$(printf '%s' "$body" | gh_create_pr "$pr_remote" "$default_branch" "$pr_branch" "$title")

    success=true
    trap - ERR

    # Delete change_branch (locally, tracking ref, and remote; skip silently if absent)
    echo "Deleting change branch $change_branch..."
    if git show-ref --verify --quiet "refs/heads/$change_branch"; then
        if ! git branch -D "$change_branch" 2>/dev/null; then
            echo "Warning: failed to delete local branch '$change_branch'" >&2
        fi
    fi
    git branch -dr "origin/$change_branch" 2>/dev/null || true
    if ! git push origin --delete "$change_branch" 2>/dev/null; then
        # Warn only if the remote branch actually existed
        if git ls-remote --exit-code --heads origin "$change_branch" >/dev/null 2>&1; then
            echo "Warning: failed to delete remote branch 'origin/$change_branch'" >&2
        fi
    fi

    echo "$pr_url"
}
