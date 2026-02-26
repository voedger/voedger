#!/usr/bin/env bash
set -Eeuo pipefail

# pr.sh -- Git branch and pull request automation
#
# Provides reusable commands for the PR workflow: branch creation from a
# remote default branch, and PR submission via GitHub CLI.
#
# Concepts:
#   pr_remote   The remote that owns the target branch for PRs.
#               "upstream" when a fork setup is detected, otherwise "origin".
#   change_branch  The current working branch (named {change-name}).
#   pr_branch      The squashed PR branch (named {change-name}--pr).
#
# Usage:
#   pr.sh info
#       Output PR configuration in key=value format:
#         pr_remote=<upstream|origin>
#         default_branch=<branch-name>
#
#   pr.sh prbranch <name>
#       Fetch pr_remote and create a local branch from its default branch.
#
#   pr.sh mergedef
#       Validate preconditions, fetch pr_remote/default_branch, and merge it into the current branch.
#       On success outputs:
#           change_branch=<name>
#           default_branch=<name>
#           change_branch_head=<sha>  (HEAD before the merge)
#
#   pr.sh diff specs
#       Output git diff of the specs folder between HEAD and pr_remote/default_branch.
#
#   pr.sh changepr --title <title> --body <body>
#       Create a PR from the current change_branch:
#         - Fail fast if pr_branch ({change_branch}--pr) already exists.
#         - Create pr_branch from pr_remote/default_branch.
#         - Squash-merge change_branch into pr_branch and commit with title.
#         - Push pr_branch to origin and create a PR via GitHub CLI.
#         - Delete change_branch (locally, tracking ref, and remote; skip if absent).
#         - Output pr_url on success.
#         - On failure after pr_branch creation: roll back pr_branch, preserve change_branch.
#
#   pr.sh pr --title <title> --body <body> --next-branch <branch> [--delete-branch]
#       Stage all changes, commit, push to origin, and open a PR against
#       pr_remote's default branch. Switch to --next-branch afterwards.
#       If --delete-branch is set, delete the current branch after switching.
#       If no changes exist, switch to --next-branch and exit cleanly.
#
#   pr.sh ffdefault
#       Fetch pr_remote/default_branch and fast-forward the local default branch to it.
#       Switches to the default branch if not already on it, and leaves there after completion.
#       Fail fast if any of the following conditions are true:
#           working directory is not clean
#           branches have diverged (fast-forward not possible)



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

read_conf_param() {
    local param_name="$1"
    local conf_file
    conf_file="$(get_project_dir)/uspecs/u/conf.md"

    if [ ! -f "$conf_file" ]; then
        error "conf.md not found: $conf_file"
    fi

    local line raw
    line=$(_grep -E "^- ${param_name}:" "$conf_file" | head -1 || true)
    raw="${line#*: }"

    if [ -z "$raw" ]; then
        error "Parameter '${param_name}' not found in conf.md"
    fi

    local value
    value=$(echo "$raw" | sed 's/^[[:space:]`]*//' | sed 's/[[:space:]`]*$//')
    echo "$value"
}

error() {
    echo "Error: $1" >&2
    exit 1
}

determine_pr_remote() {
    if git remote | _grep -q '^upstream$'; then
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
    if ! git remote | _grep -q '^origin$'; then
        error "'origin' remote does not exist"
    fi

    # Check if working directory is clean
    if [[ -n $(git status --porcelain) ]]; then
        error "Working directory has uncommitted changes. Commit or stash changes first"
    fi
}

default_branch_name() {
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
# Commands
# ---------------------------------------------------------------------------

cmd_info() {
    local pr_remote default_branch
    pr_remote=$(determine_pr_remote)
    default_branch=$(default_branch_name)
    echo "pr_remote=$pr_remote"
    echo "default_branch=$default_branch"
}

cmd_prbranch() {
    local name="${1:-}"
    [[ -z "$name" ]] && error "Usage: pr.sh prbranch <name>"

    local pr_remote default_branch
    pr_remote=$(determine_pr_remote)
    default_branch=$(default_branch_name)

    echo "Fetching $pr_remote/$default_branch..."
    git fetch "$pr_remote" "$default_branch" 2>&1

    echo "Creating branch: $name"
    git checkout -b "$name" "$pr_remote/$default_branch"
}

cmd_ffdefault() {
    check_prerequisites

    local pr_remote default_branch
    pr_remote=$(determine_pr_remote)
    default_branch=$(default_branch_name)

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

cmd_pr() {
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

    local default_branch branch_name
    default_branch=$(default_branch_name)
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

cmd_mergedef() {
    check_prerequisites

    local pr_remote default_branch current_branch
    pr_remote=$(determine_pr_remote)
    default_branch=$(default_branch_name)
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

cmd_diff() {
    local target="${1:-}"
    [[ -z "$target" ]] && error "Usage: pr.sh diff <target>"

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
    default_branch=$(default_branch_name)

    local project_dir
    project_dir=$(get_project_dir)

    git fetch "$pr_remote" "$default_branch" >/dev/null 2>&1 || true
    (cd "$project_dir" && git diff "$pr_remote/$default_branch" HEAD -- "$diff_path")
}

cmd_changepr() {
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

    local pr_remote default_branch change_branch pr_branch
    pr_remote=$(determine_pr_remote)
    default_branch=$(default_branch_name)
    change_branch=$(git symbolic-ref --short HEAD)
    pr_branch="${change_branch}--pr"

    # Fail fast if pr_branch already exists
    if git show-ref --verify --quiet "refs/heads/$pr_branch"; then
        error "PR branch '$pr_branch' already exists"
    fi

    # Create pr_branch from pr_remote/default_branch
    cmd_prbranch "$pr_branch"

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

# ---------------------------------------------------------------------------
# Dispatch
# ---------------------------------------------------------------------------

if [[ $# -lt 1 ]]; then
    error "Usage: pr.sh <info|prbranch|mergedef|diff|changepr|pr|ffdefault> [args...]"
fi

command="$1"; shift
case "$command" in
    info)      cmd_info "$@" ;;
    prbranch)  cmd_prbranch "$@" ;;
    mergedef)  cmd_mergedef "$@" ;;
    diff)      cmd_diff "$@" ;;
    changepr)  cmd_changepr "$@" ;;
    pr)        cmd_pr "$@" ;;
    ffdefault) cmd_ffdefault "$@" ;;
    *)         error "Unknown command: $command. Available: info, prbranch, mergedef, diff, changepr, pr, ffdefault" ;;
esac
