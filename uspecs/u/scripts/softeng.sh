#!/usr/bin/env bash
set -Eeuo pipefail

# softeng automation
#
# Usage:
#   softeng action upr [--no-archive]
#   softeng action umergepr
#   softeng change new <change-name> [--issue-url <url>] [--no-branch] [--branch]
#   softeng change archive <change-folder-name> [-d]
#   softeng change archiveall
#   softeng pr preflight
#   softeng pr create --title <title> [--body <body>]
#   softeng diff specs
#
# change new:
#   Creates Change Folder and change.md with frontmatter:
#     - Folder: <changes_folder from conf.md>/ymdHM-<change-name>
#     - registered_at: YYYY-MM-DDTHH:MM:SSZ
#     - change_id: ymdHM-<change-name>
#     - baseline: <commit-hash> (if git repository)
#     - issue_url: <url> (if --issue-url provided)
#   Creates git branch by default (skip with --no-branch; --branch forces creation explicitly)
#   Prints: <relative-path-to-change-folder> (e.g. uspecs/changes/2602201746-my-change)
#
# change archive [-d]:
#   Archives change folder to <changes-folder>/archive/yymm/ymdHM-<change-name>
#   Adds archived_at metadata and updates folder date prefix
#   -d: commit and push staged changes, checkout default branch, delete branch and refs
#       Requires git repository, clean working tree, PR branch (ending with --pr)
#
# change archiveall:
#   Archives all change folders with modifications vs pr_remote/default_branch
#          No change-folder-name needed; mutually exclusive with -d
#
# pr preflight --change-folder <path>:
#   Checks for uncompleted todo items in Change Folder, then validates preconditions, fetches
#   pr_remote/default_branch, and merges it into the current branch.
#   On success outputs: change_branch=<name>, default_branch=<name>, change_branch_head=<sha>
#
# pr create --title <title> --body <body>:
#   Creates a PR from the current change branch (delegates to _lib/git.sh git_changepr).
#   Body can be passed via --body or piped via stdin.
#   Literal \n sequences in --body are decoded to actual newlines.
#
# diff specs:
#   Outputs git diff of the specs folder between HEAD and pr_remote/default_branch.

get_timestamp() {
    date -u +"%Y-%m-%dT%H:%M:%SZ"
}

get_baseline() {
    local project_dir="$1"
    if is_git_repo "$project_dir"; then
        (cd "$project_dir" && git rev-parse HEAD 2>/dev/null) || echo ""
    else
        echo ""
    fi
}

get_folder_name() {
    local path="$1"
    basename "$path"
}

count_uncompleted_items() {
    local folder="$1"
    local count
    count=$(grep -r "^[[:space:]]*-[[:space:]]*\[ \]" "$folder"/*.md 2>/dev/null | wc -l)
    echo "${count:-0}" | tr -d ' '
}

extract_change_name() {
    local folder_name="$1"
    # shellcheck disable=SC2001
    echo "$folder_name" | sed 's/^[0-9]\{10\}-//'
}

move_folder() {
    local source="$1"
    local destination="$2"
    local project_dir="${3:-}"
    local check_dir="${project_dir:-$PWD}"
    if is_git_repo "$check_dir"; then
        if [[ -n "$project_dir" ]]; then
            local rel_src="${source#"$project_dir/"}"
            local rel_dst="${destination#"$project_dir/"}"
            (cd "$project_dir" && git mv "$rel_src" "$rel_dst" 2>/dev/null) || mv "$source" "$destination"
        else
            git mv "$source" "$destination" 2>/dev/null || mv "$source" "$destination"
        fi
    else
        mv "$source" "$destination"
    fi
}

get_script_dir() {
    cd "$(dirname "${BASH_SOURCE[0]}")" && pwd
}

# shellcheck source=_lib/utils.sh
source "$(get_script_dir)/_lib/utils.sh"
# shellcheck source=_lib/git.sh
source "$(get_script_dir)/_lib/git.sh"

get_project_dir() {
    local script_dir
    script_dir=$(get_script_dir)
    # scripts/ -> u/ -> uspecs/ -> project root
    cd "$script_dir/../../.." && pwd
}

cmd_status_ispr() {
    local project_dir
    project_dir=$(get_project_dir)
    if ! is_git_repo "$project_dir"; then
        return 0
    fi
    local branch
    branch=$(cd "$project_dir" && git branch --show-current 2>&1)
    if [[ "$branch" == *"--pr" ]]; then
        echo "yes"
    fi
}

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

    # trim leading/trailing whitespace and surrounding backticks
    local value
    value=$(echo "$raw" | sed 's/^[[:space:]`]*//' | sed 's/[[:space:]`]*$//')

    echo "$value"
}

extract_issue_id() {
    # Extract issue ID from the last segment of an issue URL
    # Takes the last /-separated segment, finds the first contiguous
    # run of valid characters (alphanumeric, hyphens, underscores)
    local url="$1"
    local segment="${url##*/}"
    if [[ "$segment" =~ ^[^a-zA-Z0-9_-]*([a-zA-Z0-9_-]+) ]]; then
        echo "${BASH_REMATCH[1]}"
    fi
}

cmd_change_new() {
    local change_name=""
    local issue_url=""
    local opt_branch=""
    local opt_no_branch=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --issue-url)
                if [[ $# -lt 2 || -z "$2" ]]; then
                    error "--issue-url requires a URL argument"
                fi
                issue_url="$2"
                shift 2
                ;;
            --branch)
                opt_branch="1"
                shift
                ;;
            --no-branch)
                opt_no_branch="1"
                shift
                ;;
            *)
                if [ -z "$change_name" ]; then
                    change_name="$1"
                    shift
                else
                    error "Unknown argument: $1"
                fi
                ;;
        esac
    done

    if [ -n "$opt_branch" ] && [ -n "$opt_no_branch" ]; then
        error "--branch and --no-branch are mutually exclusive"
    fi

    local is_new_branch="1"
    if [ -n "$opt_no_branch" ]; then
        is_new_branch=""
    elif [ -z "$opt_branch" ]; then
        # Skip branch creation when not on the default branch (unless --branch forces it)
        local project_dir_check
        project_dir_check=$(get_project_dir)
        if is_git_repo "$project_dir_check"; then
            local current_branch_name
            current_branch_name=$(cd "$project_dir_check" && git symbolic-ref --short HEAD)
            local def_branch
            def_branch=$(cd "$project_dir_check" && git_default_branch_name || echo "")
            if [ "$current_branch_name" != "$def_branch" ]; then
                is_new_branch=""
            fi
        fi
    fi

    if [ -z "$change_name" ]; then
        error "change-name is required"
    fi

    if [[ ! "$change_name" =~ ^[a-z0-9][a-z0-9-]*$ ]]; then
        error "change-name must be kebab-case (lowercase letters, numbers, hyphens): $change_name"
    fi

    local changes_folder_rel
    changes_folder_rel=$(read_conf_param "changes_folder")

    local project_dir
    project_dir=$(get_project_dir)

    local changes_folder="$project_dir/$changes_folder_rel"

    if [ ! -d "$changes_folder" ]; then
        error "Changes folder not found: $changes_folder"
    fi

    local timestamp
    timestamp=$(date -u +"%y%m%d%H%M")

    local folder_name="${timestamp}-${change_name}"
    local change_folder="$changes_folder/$folder_name"

    if [ -d "$change_folder" ]; then
        error "Change folder already exists: $change_folder"
    fi

    mkdir -p "$change_folder"

    local registered_at baseline
    registered_at=$(get_timestamp)
    baseline=$(get_baseline "$project_dir")

    local frontmatter="---"$'\n'
    frontmatter+="registered_at: $registered_at"$'\n'
    frontmatter+="change_id: $folder_name"$'\n'

    if [ -n "$baseline" ]; then
        frontmatter+="baseline: $baseline"$'\n'
    fi

    if [ -n "$issue_url" ]; then
        frontmatter+="issue_url: $issue_url"$'\n'
    fi

    frontmatter+="---"

    printf '%s\n' "$frontmatter" > "$change_folder/change.md"

    if [ -n "$is_new_branch" ]; then
        if is_git_repo "$project_dir"; then
            local branch_name="$change_name"
            if [ -n "$issue_url" ]; then
                local issue_id
                issue_id=$(extract_issue_id "$issue_url")
                if [ -n "$issue_id" ]; then
                    branch_name="${issue_id}-${change_name}"
                fi
            fi
            if ! (cd "$project_dir" && git checkout -b "$branch_name"); then
                echo "Warning: Failed to create branch '$branch_name'" >&2
            fi
        else
            echo "Warning: Not a git repository, cannot create branch" >&2
        fi
    fi

    echo "$changes_folder_rel/$folder_name"
}

convert_links_to_relative() {
    local folder="$1"

    if [ -z "$folder" ]; then
        error "folder path is required for convert_links_to_relative"
    fi

    if [ ! -d "$folder" ]; then
        error "Folder not found: $folder"
    fi

    # Find all .md files in the folder
    local md_files
    md_files=$(find "$folder" -maxdepth 1 -name "*.md" -type f)

    if [ -z "$md_files" ]; then
        # No markdown files to process, return success
        return 0
    fi

    # Process each markdown file
    while IFS= read -r file; do
        # Archive moves folder 2 levels deeper (changes/ -> changes/archive/yymm/)
        # Only paths starting with ../ need adjustment - add ../../ prefix
        #
        # Example: ](../foo) -> ](../../../foo)
        #
        # Skip (do not modify):
        # - http://, https:// (absolute URLs)
        # - # (anchors)
        # - / (absolute paths)
        # - ./ (current directory - stays in same folder)
        # - filename.ext (same folder files like impl.md, issue.md)

        # Add ../../ prefix to paths starting with ../
        # ](../ -> ](../../../
        if ! sed_inplace "$file" -E 's#\]\(\.\./#](../../../#g'; then
            error "Failed to convert links in file: $file"
        fi
    done <<< "$md_files"

    return 0
}

cmd_pr_preflight() {
    local change_folder_path=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --change-folder) change_folder_path="$2"; shift 2 ;;
            *) error "Unknown flag: $1" ;;
        esac
    done
    if [ -z "$change_folder_path" ]; then
        error "pr preflight requires --change-folder <path>"
    fi
    if [ ! -d "$change_folder_path" ]; then
        error "Change folder not found: $change_folder_path"
    fi
    local uncompleted_count
    uncompleted_count=$(count_uncompleted_items "$change_folder_path")
    if [ "$uncompleted_count" -gt 0 ]; then
        echo "Cannot create PR: $uncompleted_count uncompleted todo item(s) found"
        echo ""
        echo "Uncompleted items:"
        grep -rn "^[[:space:]]*-[[:space:]]*\[ \]" "$change_folder_path"/*.md 2>/dev/null | sed 's/^/  /'
        echo ""
        echo "Complete or cancel todo items before creating a PR"
        exit 1
    fi
    git_mergedef
}

cmd_change_archiveall() {
    if [ $# -gt 0 ]; then
        error "change archiveall takes no arguments"
    fi

    local changes_folder_rel
    changes_folder_rel=$(read_conf_param "changes_folder")

    local project_dir
    project_dir=$(get_project_dir)

    if ! is_git_repo "$project_dir"; then
        error "change archiveall requires a git repository"
    fi

    local -A pr_info
    if ! git_pr_info pr_info "$project_dir"; then
        error "change archiveall requires remote info to be available (remote reachable?)"
    fi
    local default_branch="${pr_info[default_branch]:-}"
    local pr_remote="${pr_info[pr_remote]:-}"

    local changes_folder="$project_dir/$changes_folder_rel"

    echo "Fetching ${pr_remote}/${default_branch}..."
    (cd "$project_dir" && git fetch "$pr_remote" "$default_branch" 2>&1)

    if [ ! -d "$changes_folder" ]; then
        error "Changes folder not found: $changes_folder"
    fi

    local archived=0 unchanged=0 failed=0
    local script_path
    script_path="$(get_script_dir)/softeng.sh"

    for folder_path in "$changes_folder"/*/; do
        [ -d "$folder_path" ] || continue
        local fname
        fname=$(basename "$folder_path")
        [ "$fname" = "archive" ] && continue

        local rel_folder="$changes_folder_rel/$fname"
        local diff_output
        diff_output=$(cd "$project_dir" && git diff --name-only "${pr_remote}/${default_branch}" HEAD -- "$rel_folder")
        if [ -z "$diff_output" ]; then
            unchanged=$((unchanged + 1))
            continue
        fi

        if bash "$script_path" change archive "$fname"; then
            archived=$((archived + 1))
        else
            echo "Warning: could not archive $fname" >&2
            failed=$((failed + 1))
        fi
    done

    echo "Done: $archived archived, $unchanged unchanged, $failed failed"
}

# changes_archive <project_dir> <changes_folder> <change_folder> <is_git> <result_var>
# Archives an active change folder: updates YAML metadata, converts links,
# moves to archive/YYMM/YYMMDDHHMM-<change_name>.
# project_dir: absolute path to project root
# changes_folder: relative to project_dir (e.g. uspecs/changes)
# change_folder: relative to project_dir (e.g. uspecs/changes/2601010000-my-change)
# is_git: non-empty if project is a git repo
# Sets result_var (nameref) to the archived folder path, relative to project_dir.
changes_archive() {
    local project_dir="$1"
    local changes_folder="$2"
    local change_folder="$3"
    local is_git="$4"
    local -n result_ref="$5"

    local abs_change="$project_dir/$change_folder"
    local abs_changes="$project_dir/$changes_folder"

    local folder_basename
    folder_basename=$(basename "$change_folder")

    local change_name
    change_name=$(extract_change_name "$folder_basename")

    local change_file="$abs_change/change.md"

    local timestamp
    timestamp=$(get_timestamp)

    # Insert archived_at into YAML front matter (before closing ---)
    local temp_file
    temp_create_file temp_file
    # // TODO archived_at may already exists...
    awk -v ts="$timestamp" '
        /^---$/ {
            if (count == 0) {
                print
                count++
            } else {
                print "archived_at: " ts
                print
            }
            next
        }
        /^archived_at:/ { next }
        { print }
    ' "$change_file" > "$temp_file"
    if cat "$temp_file" > "$change_file"; then
        :  # Success, continue
    else
        error "failed to update $change_file"
    fi

    # Add ../ prefix to relative links for archive folder depth
    if ! convert_links_to_relative "$abs_change"; then
        error "failed to convert links to relative paths"
    fi

    local archive_dir="$abs_changes/archive"

    local date_prefix
    date_prefix=$(date -u +"%y%m%d%H%M")

    local yymm="${date_prefix:0:4}"

    local archive_sub="$archive_dir/$yymm"
    mkdir -p "$archive_sub"

    local dest="$archive_sub/${date_prefix}-${change_name}"

    if [ -d "$dest" ]; then
        error "Archive folder already exists: $dest"
    fi

    if [ -n "$is_git" ]; then
        (cd "$project_dir" && quiet git add "$change_folder")
    fi

    move_folder "$abs_change" "$dest" "$project_dir"

    local rel_dest="${dest#"$project_dir/"}"

    if [ -n "$is_git" ]; then
        (cd "$project_dir" && quiet git add "$rel_dest")
    fi

    # shellcheck disable=SC2034
    result_ref="$rel_dest"
}

cmd_change_archive() {
    local folder_name=""
    local delete_branch=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            -d)
                delete_branch="1"
                shift
                ;;
            *)
                if [ -z "$folder_name" ]; then
                    folder_name="$1"
                    shift
                else
                    error "Unknown argument: $1"
                fi
                ;;
        esac
    done

    if [ -z "$folder_name" ]; then
        error "change-folder-name is required"
    fi

    local changes_folder_rel
    changes_folder_rel=$(read_conf_param "changes_folder")

    local project_dir
    project_dir=$(get_project_dir)

    local is_git=""
    if is_git_repo "$project_dir"; then
        is_git="1"
    fi

    if [ -n "$delete_branch" ] && [ -z "$is_git" ]; then
        error "-d requires a git repository"
    fi

    local changes_folder="$project_dir/$changes_folder_rel"

    local path_to_change_folder="$changes_folder/$folder_name"

    if [ ! -d "$path_to_change_folder" ]; then
        error "Folder not found: $path_to_change_folder"
    fi

    local change_file="$path_to_change_folder/change.md"
    if [ ! -f "$change_file" ]; then
        error "change.md not found in folder: $path_to_change_folder"
    fi

    if [[ "$folder_name" == archive/* ]]; then
        error "Folder is already in archive: $folder_name"
    fi

    local uncompleted_count
    uncompleted_count=$(count_uncompleted_items "$path_to_change_folder")

    if [ "$uncompleted_count" -gt 0 ]; then
        echo "Cannot archive: $uncompleted_count uncompleted todo item(s) found"
        echo ""
        echo "Uncompleted items:"
        grep -rn "^[[:space:]]*-[[:space:]]*\[ \]" "$path_to_change_folder"/*.md 2>/dev/null | sed 's/^/  /'
        echo ""
        echo "Complete or cancel todo items before archiving"
        exit 1
    fi

    local change_name
    change_name=$(extract_change_name "$folder_name")

    if [ -n "$delete_branch" ] && [ -n "$is_git" ]; then
        local branch_name
        branch_name=$(cd "$project_dir" && git symbolic-ref --short HEAD 2>/dev/null || echo "")
        if [ -z "$branch_name" ]; then
            error "-d requires a named branch (HEAD is detached)"
        fi

        local -A pr_info
        if ! git_pr_info pr_info; then
            error "-d requires git remote info to be available (remote reachable?)"
        fi
        local default_branch="${pr_info[default_branch]:-}"
        local pr_remote="${pr_info[pr_remote]:-}"

        # a) no uncommitted changes
        local git_status
        git_status=$(cd "$project_dir" && git status --porcelain)
        if [ -n "$git_status" ]; then
            error "-d requires a clean working tree (uncommitted changes found)"
        fi

        # b) branch must not be the default branch
        if [ "$branch_name" = "$default_branch" ]; then
            error "-d cannot be used on the default branch '$default_branch'"
        fi

        # c) check whether the remote branch actually exists
        # exit non-zero means remote unreachable/auth failed -- treat as hard error, not as "branch gone"
        local remote_exists
        if ! remote_exists=$(cd "$project_dir" && git ls-remote --heads "${pr_remote:-origin}" "$branch_name"); then
            error "Cannot reach remote '${pr_remote:-origin}'. Check connectivity and authentication."
        fi

        # d) no divergence (skip when remote branch is already gone)
        if [ -n "$remote_exists" ]; then
            (cd "$project_dir" && git fetch "${pr_remote:-origin}" "$branch_name" 2>&1)
            local behind
            behind=$(cd "$project_dir" && git rev-list --count "HEAD..${pr_remote:-origin}/$branch_name")
            if [ "$behind" -gt 0 ]; then
                error "Branch '$branch_name' is behind ${pr_remote:-origin} by $behind commit(s). Pull or rebase first."
            fi
        fi

        # e) branch must be a PR branch
        if [[ "$branch_name" != *--pr ]]; then
            error "-d can only be used on a PR branch (must end with '--pr'): '$branch_name'"
        fi

        # f) PR branch with remote gone: skip archive, just refresh default and switch
        if [ -z "$remote_exists" ]; then
            echo "Remote branch '${pr_remote:-origin}/$branch_name' no longer exists; skipping archive."
            echo "Switching to $default_branch..."
            if ! (cd "$project_dir" && git checkout "$default_branch" 2>&1); then
                error "Failed to checkout '$default_branch'. Resolve manually."
            fi

            local deleted_branch_hash=""
            if (cd "$project_dir" && git show-ref --verify --quiet "refs/heads/$branch_name"); then
                deleted_branch_hash=$(cd "$project_dir" && git rev-parse "refs/heads/$branch_name")
                (cd "$project_dir" && git branch -D "$branch_name" 2>&1)
            fi
            (cd "$project_dir" && git branch -dr "${pr_remote:-origin}/$branch_name") 2>/dev/null || true

            if [ -n "$deleted_branch_hash" ]; then
                echo "Deleted branch: $branch_name ($deleted_branch_hash)"
                echo "To restore: git branch $branch_name $deleted_branch_hash"
            fi

            echo "Updating local $default_branch from ${pr_remote:-origin}/$default_branch..."
            (cd "$project_dir" && git fetch "${pr_remote:-origin}" "$default_branch" 2>&1)
            if ! (cd "$project_dir" && git merge --ff-only "${pr_remote:-origin}/$default_branch" 2>&1); then
                error "Cannot fast-forward '$default_branch' to '${pr_remote:-origin}/$default_branch' (branches have diverged). Run 'git rebase' or resolve manually."
            fi

            echo "Done: skipped archive (remote branch gone), switched to $default_branch"
            return 0
        fi
    fi

    local rel_change_folder="$changes_folder_rel/$folder_name"

    local archive_path
    changes_archive "$project_dir" "$changes_folder_rel" "$rel_change_folder" "$is_git" archive_path

    if [ -n "$delete_branch" ] && [ -n "$is_git" ]; then
        (cd "$project_dir" && git commit -m "archive $rel_change_folder to $archive_path" 2>&1)
        if [ -n "$remote_exists" ]; then
            if ! (cd "$project_dir" && git push 2>&1); then
                local archive_commit
                archive_commit=$(cd "$project_dir" && git rev-parse HEAD)
                error "Push to '${pr_remote:-origin}/$branch_name' failed. Branch '$branch_name' preserved (archive commit: $archive_commit). Resolve the push issue and re-run the archive command."
            fi
        else
            echo "Remote branch '${pr_remote:-origin}/$branch_name' no longer exists; skipping push."
        fi

        if ! (cd "$project_dir" && git checkout "$default_branch" 2>&1); then
            error "Failed to checkout '$default_branch'. Resolve manually."
        fi

        local deleted_branch_hash=""
        if (cd "$project_dir" && git show-ref --verify --quiet "refs/heads/$branch_name"); then
            deleted_branch_hash=$(cd "$project_dir" && git rev-parse "refs/heads/$branch_name")
            (cd "$project_dir" && git branch -D "$branch_name" 2>&1)
        else
            echo "Warning: branch '$branch_name' not found, skipping branch deletion" >&2
        fi
        (cd "$project_dir" && git branch -dr "${pr_remote:-origin}/$branch_name") 2>/dev/null || true

        if [ -n "$deleted_branch_hash" ]; then
            echo "Deleted branch: $branch_name ($deleted_branch_hash)"
            echo "To restore: git branch $branch_name $deleted_branch_hash"
        fi

        echo "Updating local $default_branch from ${pr_remote:-origin}/$default_branch..."
        (cd "$project_dir" && git fetch "${pr_remote:-origin}" "$default_branch" 2>&1)
        if ! (cd "$project_dir" && git merge --ff-only "${pr_remote:-origin}/$default_branch" 2>&1); then
            error "Cannot fast-forward '$default_branch' to '${pr_remote:-origin}/$default_branch' (branches have diverged). Run 'git rebase' or resolve manually."
        fi
    fi
}

# changes_validate_single_wcf <project_dir> <changes_folder_rel> <pr_remote> <default_branch>
# Reflects scenario: "Exactly one Working Change Folder"
# Detects the Working Change Folder (WCF) -- a change folder whose files have been
# modified since merge-base with pr_remote/default_branch.
# Outputs the relative path from changes_folder (e.g. "my-change" for active,
# "archive/yymm/timestamp-name" for archived). Fails if not exactly one WCF is found.
changes_validate_single_wcf() {
    local project_dir="$1"
    local changes_folder_rel="$2"
    local pr_remote="$3"
    local default_branch="$4"

    local merge_base
    merge_base=$(cd "$project_dir" && git merge-base HEAD "${pr_remote}/${default_branch}")

    local diff_output
    diff_output=$(cd "$project_dir" && git diff --name-only "$merge_base" HEAD -- "$changes_folder_rel")

    # Collect unique change folder paths.
    # Active folders: first path component (e.g. "my-change")
    # Archived folders: archive/yymm/<name> (3 components)
    local -A folders=()
    while IFS= read -r line; do
        [[ -z "$line" ]] && continue
        # Strip changes_folder_rel/ prefix
        local rel="${line#"${changes_folder_rel}/"}"
        local top="${rel%%/*}"
        [[ -z "$top" ]] && continue
        if [[ "$top" == "archive" ]]; then
            # Extract archive/yymm/<name> (3 path components)
            local rest="${rel#archive/}"
            local yymm="${rest%%/*}"
            rest="${rest#"$yymm"/}"
            local name="${rest%%/*}"
            if [[ -n "$yymm" && -n "$name" ]]; then
                folders["archive/$yymm/$name"]=1
            fi
        else
            folders["$top"]=1
        fi
    done <<< "$diff_output"

    local count=${#folders[@]}
    if [[ "$count" -eq 0 ]]; then
        error "No Working Change Folder found (no changes in $changes_folder_rel since merge-base)"
    elif [[ "$count" -gt 1 ]]; then
        local names
        names=$(printf '%s\n' "${!folders[@]}" | sort)
        error "Multiple Working Change Folders found (expected exactly one):\n$names"
    fi

    printf '%s\n' "${!folders[@]}"
}

# changes_validate_todos_completed <wcf_path> <project_dir>
# Reflects scenario: "All todo items are completed"
# Checks that there are no uncompleted todo items in the WCF.
# On failure, outputs error to stderr and exits.
changes_validate_todos_completed() {
    local wcf_path="$1"
    local project_dir="$2"

    local uncompleted_count
    uncompleted_count=$(count_uncompleted_items "$wcf_path")
    if [[ "$uncompleted_count" -gt 0 ]]; then
        local uncompleted_files
        uncompleted_files=$(grep -rl "^[[:space:]]*-[[:space:]]*\[ \]" "$wcf_path"/*.md 2>/dev/null | sed "s|^$project_dir/||")

        {
            echo "Error: $uncompleted_count uncompleted todo item(s) found in files:"
            echo ""
            echo "$uncompleted_files"
            echo ""
            echo "Complete todo items before creating a PR."
        } >&2
        exit 1
    fi
}

# cmd_action_upr
# Full upr flow: validate, detect WCF, check no existing PR, read change.md,
# compute pr_title/commit_message/see_details_line,
# set upstream, squash, force-push, open PR creation in browser, output prompt.
cmd_action_upr() {
    local opt_no_archive=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --no-archive) opt_no_archive="1"; shift ;;
            *) error "Unknown argument: $1" ;;
        esac
    done

    local project_dir
    project_dir=$(get_project_dir)
    cd "$project_dir"

    prompt_start_log

    # Validate preconditions
    check_prerequisites

    local current_branch
    current_branch=$(git symbolic-ref --short HEAD)

    local pr_remote default_branch
    pr_remote=$(determine_pr_remote)
    default_branch=$(git_default_branch_name)

    git_validate_clean_repo "$current_branch" "$default_branch"

    echo "Branch: $current_branch -> $pr_remote/$default_branch"

    # Fetch remote default branch
    echo "Fetching $pr_remote/$default_branch..."
    quiet git fetch "$pr_remote" "$default_branch"

    # Check for changes since branching
    echo "Checking for changes since branching..."
    local merge_base
    merge_base=$(git merge-base HEAD "${pr_remote}/${default_branch}")
    local diff_stat
    diff_stat=$(git diff --name-only "$merge_base" HEAD)
    if [[ -z "$diff_stat" ]]; then
        error "No changes detected in the current branch since branching from $default_branch"
    fi

    # Detect Working Change Folder
    local changes_folder_rel
    changes_folder_rel=$(read_conf_param "changes_folder")
    local wcf_name
    wcf_name=$(changes_validate_single_wcf "$project_dir" "$changes_folder_rel" "$pr_remote" "$default_branch")

    echo "Working Change Folder: $wcf_name"

    local wcf_path="$project_dir/$changes_folder_rel/$wcf_name"
    local change_file="$wcf_path/change.md"

    if [[ ! -f "$change_file" ]]; then
        error "change.md not found in Working Change Folder: $wcf_path"
    fi

    # Check for uncompleted todo items
    echo "Checking for uncompleted to-do items..."
    changes_validate_todos_completed "$wcf_path" "$project_dir"

    local prompts_file
    prompts_file="$(get_script_dir)/../prompts.md"

    # Check if PR already exists for this branch
    echo "Checking for existing PR..."
    local pr_state pr_number
    if pr_state=$(gh pr view --json state -q ".state" 2>/dev/null); then
        # PR exists -- check its state
        pr_number=$(gh pr view --json number -q ".number")

        if [[ "$pr_state" == "OPEN" ]]; then
            # PR exists and is OPEN -- open in browser and show message
            local pr_url
            pr_url=$(gh pr view --json url -q ".url")
            quiet gh pr view --web || true

            prompt_start_instructions
            # shellcheck disable=SC2034  # open_vars used via nameref in section_templ
            declare -A open_vars=([pr_url]="$pr_url")
            section_templ "$prompts_file" "upr_already_exists" open_vars
            return 0
        elif [[ "$pr_state" == "MERGED" ]]; then
            echo "PR #${pr_number} for this branch was already merged. Proceeding with new PR creation..."
        fi
        # PR exists but is CLOSED -- proceed silently with new PR creation
    fi

    # Read change.md: title and optional issue_url
    local full_title
    full_title=$(md_read_title "$change_file")
    # change_title is text after ":" in the heading, trimmed
    local change_title
    if [[ "$full_title" == *:* ]]; then
        change_title="${full_title#*:}"
        change_title="${change_title#"${change_title%%[![:space:]]*}"}"
    else
        change_title="$full_title"
    fi

    local issue_url pr_title commit_message see_details_line
    issue_url=$(md_read_frontmatter_field "$change_file" "issue_url" 2>/dev/null) || true

    see_details_line="See change.md for details"

    if [[ -n "$issue_url" ]]; then
        local issue_id
        issue_id=$(extract_issue_id "$issue_url")
        pr_title="[${issue_id}] ${change_title}"
        commit_message="Closes #${issue_id}: ${change_title}"$'\n'"${see_details_line}"
    else
        pr_title="${change_title}"
        commit_message="${change_title}"$'\n'"${see_details_line}"
    fi

    # Archive WCF if active and --no-archive not set
    if [[ -z "$opt_no_archive" && -d "$wcf_path" && "$wcf_name" != archive/* ]]; then
        echo "Archiving WCF $wcf_name..."
        local archived_path
        changes_archive "$project_dir" "$changes_folder_rel" "$changes_folder_rel/$wcf_name" "1" archived_path

        # Update change_file to archived location
        change_file="$project_dir/$archived_path/change.md"

        if [[ -n $(git status --porcelain) ]]; then
            quiet git add -A
            quiet git commit -m "Archive $wcf_name"
        fi
    fi

    # Count commits since merge-base to decide whether to squash
    local commit_count
    commit_count=$(git rev-list --count "$merge_base"..HEAD)

    # Set upstream if not already set
    if ! git rev-parse --abbrev-ref "@{upstream}" >/dev/null 2>&1; then
        quiet git push -u origin "$current_branch"
    fi

    echo "PR title: $pr_title"
    echo "Commits since merge-base: $commit_count"

    local pre_push_head=""
    if [[ "$commit_count" -gt 1 ]]; then
        # Record pre-push HEAD for branch restoration
        pre_push_head=$(git rev-parse HEAD)

        # Squash branch into single commit
        echo "Squashing $commit_count commits into one..."
        quiet git reset --soft "$merge_base"
        quiet git commit -m "$commit_message"

        # Register branch restoration handler in case force-push fails
        atexit_push "git reset --hard ${pre_push_head}"

        # Force-push
        echo "Force-pushing squashed commit..."
        quiet git push --force

        # Force-push succeeded -- remove restoration handler
        atexit_pop
    else
        # Already a single commit -- skip squash and force-push
        echo "Single commit, pushing..."
        quiet git push
    fi

    # Prepare PR body: strip YAML frontmatter --- delimiters (keep field lines as plain text).
    # Fenced code blocks (```yaml / ```) are NOT used because GitHub interprets backtick
    # sequences in the PR body incorrectly.
    local pr_body_file
    temp_create_file pr_body_file
    local pr_body_max_lines=40
    local pr_body_max_chars=4000
    awk '
        BEGIN { fm=0 }
        /^---$/ && fm==0 { fm=1; next }
        /^---$/ && fm==1 { fm=2; next }
        { print }
    ' "$change_file" > "$pr_body_file"
    local pr_body_truncated=false
    local pr_body_lines
    pr_body_lines=$(wc -l < "$pr_body_file")
    if (( pr_body_lines > pr_body_max_lines )); then
        head -n "$pr_body_max_lines" "$pr_body_file" > "${pr_body_file}.tmp"
        mv "${pr_body_file}.tmp" "$pr_body_file"
        pr_body_truncated=true
    fi
    local pr_body_size
    pr_body_size=$(wc -c < "$pr_body_file")
    if (( pr_body_size > pr_body_max_chars )); then
        local truncated
        truncated=$(head -c "$pr_body_max_chars" "$pr_body_file")
        printf '%s' "$truncated" > "$pr_body_file"
        pr_body_truncated=true
    fi
    if [[ "$pr_body_truncated" == "true" ]]; then
        printf '\n\n---\n(truncated -- see change.md for full details)\n' >> "$pr_body_file"
    fi

    # Create PR via gh CLI
    echo "Creating PR..."
    local pr_url
    pr_url=$(gh_create_pr "$pr_remote" "$default_branch" "$current_branch" "$pr_title" < "$pr_body_file")

    # Open the created PR in browser
    echo "Opening PR in browser..."
    quiet gh pr view --web || true

    prompt_start_instructions

    # Output success prompt
    if [[ -n "$pre_push_head" ]]; then
        declare -A vars=([pre_push_head]="$pre_push_head" [pr_url]="$pr_url")
        section_templ "$prompts_file" "upr_success" vars
    else
        # shellcheck disable=SC2034  # vars used via nameref
        declare -A vars=([pr_url]="$pr_url")
        section_templ "$prompts_file" "upr_success_no_squash" vars
    fi
}

# cmd_action_umergepr
# Full umergepr flow: validate, detect WCF, check PR state, handle branches,
# archive WCF if active, attempt merge, handle failure, branch cleanup.
cmd_action_umergepr() {
    local project_dir
    project_dir=$(get_project_dir)
    cd "$project_dir"

    prompt_start_log

    # Validate preconditions
    check_prerequisites

    local current_branch
    current_branch=$(git symbolic-ref --short HEAD)

    local pr_remote default_branch
    pr_remote=$(determine_pr_remote)
    default_branch=$(git_default_branch_name)

    git_validate_clean_repo "$current_branch" "$default_branch"

    echo "Branch: $current_branch -> $pr_remote/$default_branch"

    # Check upstream
    if ! git rev-parse --abbrev-ref "@{upstream}" >/dev/null 2>&1; then
        error "Current branch '$current_branch' has no upstream"
    fi

    # Fetch remote default branch
    echo "Fetching $pr_remote/$default_branch..."
    quiet git fetch "$pr_remote" "$default_branch"

    # Detect Working Change Folder
    local changes_folder_rel
    changes_folder_rel=$(read_conf_param "changes_folder")
    local wcf_name
    wcf_name=$(changes_validate_single_wcf "$project_dir" "$changes_folder_rel" "$pr_remote" "$default_branch")
    echo "Working Change Folder: $wcf_name"

    local prompts_file
    prompts_file="$(get_script_dir)/../prompts.md"

    # Check PR state
    echo "Checking PR state..."
    local pr_state pr_number
    if ! pr_state=$(gh pr view --json state -q ".state" 2>/dev/null); then
        # No PR found
        prompt_start_instructions
        section_templ "$prompts_file" "umergepr_no_pr"
        return 0
    fi

    pr_number=$(gh pr view --json number -q ".number")
    local pr_url
    pr_url=$(gh pr view --json url -q ".url")

    echo "PR #$pr_number state: $pr_state"

    if [[ "$pr_state" != "OPEN" ]]; then
        # PR is not in OPEN state
        quiet gh pr view --web || true

        local branch_head
        branch_head=$(git rev-parse HEAD)

        # Delete local branch, upstream and remote tracking ref (errors ignored)
        quiet git checkout "$default_branch" || true
        git branch -D "$current_branch" >/dev/null 2>&1 || true
        git branch -dr "origin/$current_branch" >/dev/null 2>&1 || true

        # shellcheck disable=SC2034  # vars used via nameref
        declare -A vars=(
            [pr_number]="$pr_number"
            [pr_state]="$pr_state"
            [branch_name]="$current_branch"
            [branch_head]="$branch_head"
        )
        prompt_start_instructions
        section_templ "$prompts_file" "umergepr_not_open" vars
        return 0
    fi

    # PR is in OPEN state
    # Archive WCF if active
    local wcf_path="$project_dir/$changes_folder_rel/$wcf_name"
    local archived_path=""
    if [[ -d "$wcf_path" && ! "$wcf_path" == */archive/* ]]; then
        echo "Archiving WCF $wcf_name..."
        changes_archive "$project_dir" "$changes_folder_rel" "$changes_folder_rel/$wcf_name" "1" archived_path

        # Commit the archive
        if [[ -n $(git status --porcelain) ]]; then
            quiet git add -A
            quiet git commit -m "Archive $wcf_name"
            echo "Pushing archive commit..."
            quiet git push || true
        fi
    fi

    # Sync PR branch with latest base branch (handles "base branch was modified" error)
    echo "Updating PR branch with latest base..."
    quiet gh pr update-branch || echo "Warning: gh pr update-branch failed (may not be needed)"

    # Record branch HEAD before merge deletes it
    local branch_head
    branch_head=$(git rev-parse HEAD)

    # Attempt merge with squash and delete branch
    echo "Merging PR #$pr_number (squash)..."
    if ! quiet gh pr merge --squash --delete-branch; then
        # Merge failed
        quiet gh pr view --web || true

        # shellcheck disable=SC2034  # vars used via nameref
        declare -A fail_vars=([pr_number]="$pr_number")
        prompt_start_instructions
        section_templ "$prompts_file" "umergepr_merge_failed" fail_vars
        return 0
    fi

    # Merge succeeded -- cleanup
    echo "Merge succeeded, cleaning up..."
    # gh pr merge --delete-branch switches to default branch and deletes local branch,
    # but in fork workflows (crossRepoPR) it skips remote branch deletion by design.
    # Explicitly delete the branch on origin (the fork) and clean up tracking ref.
    if git ls-remote --exit-code --heads origin "$current_branch" >/dev/null 2>&1; then
        echo "Deleting branch $current_branch from origin..."
        quiet git push origin --delete "$current_branch" || echo "Warning: failed to delete $current_branch from origin"
    fi
    git branch -dr "origin/$current_branch" >/dev/null 2>&1 || true

    # When upstream remote exists, fast-forward local default branch.
    # Retry fetch+ff for up to 5 seconds -- the squashed commit may not appear
    # immediately due to eventual consistency.
    if [[ "$pr_remote" == "upstream" ]]; then
        echo "Syncing local $default_branch with $pr_remote/$default_branch..."
        cd "$project_dir"
        # Ensure we are on the default branch (gh pr merge should have switched,
        # but be explicit to avoid accidentally fast-forwarding a wrong branch).
        quiet git checkout "$default_branch" || true

        echo "Fetching $pr_remote/$default_branch..."
        quiet git fetch "$pr_remote" "$default_branch"

        if ! git merge-base --is-ancestor HEAD "$pr_remote/$default_branch" 2>/dev/null; then
            # Local branch has diverged from upstream -- log and skip sync entirely
            echo "Warning: local $default_branch has diverged from $pr_remote/$default_branch, skipping sync"
            echo "  local HEAD: $(git rev-parse --short HEAD)"
            echo "  $pr_remote/$default_branch: $(git rev-parse --short "$pr_remote/$default_branch")"
            echo "  merge-base: $(git merge-base HEAD "$pr_remote/$default_branch" | cut -c1-7)"
            echo "  local-only commits:"
            git log --oneline "$pr_remote/$default_branch..HEAD" 2>&1 | sed 's/^/    /'
        else
            # Fast-forward is possible -- retry ff+WCF detection for up to 5 seconds
            # (squashed commit may not appear immediately due to eventual consistency)
            local _wcf_check_path="$project_dir/${archived_path:-$changes_folder_rel/$wcf_name}"
            local _wcf_found=false
            for _attempt in 1 2 3 4 5; do
                echo "Fast-forwarding $default_branch (attempt $_attempt)..."
                quiet git fetch "$pr_remote" "$default_branch"
                quiet git merge --ff-only "$pr_remote/$default_branch"
                quiet git push origin "$default_branch" || echo "Warning: failed to push $default_branch to origin"
                if [[ -d "$_wcf_check_path" ]]; then
                    _wcf_found=true
                    break
                fi
                sleep 1
            done
            if [[ "$_wcf_found" != "true" ]]; then
                echo "Warning: WCF not detected in $default_branch after 5 seconds"
            fi
        fi
    fi

    # shellcheck disable=SC2034  # vars used via nameref
    declare -A success_vars=(
        [pr_number]="$pr_number"
        [pr_url]="$pr_url"
        [branch_name]="$current_branch"
        [branch_head]="$branch_head"
    )
    # shellcheck disable=SC2119
    prompt_start_instructions
    section_templ "$prompts_file" "umergepr_success" success_vars
}

main() {
    git_path

    if [ $# -lt 1 ]; then
        error "Usage: softeng <command> [args...]"
    fi

    local command="$1"
    shift

    case "$command" in
        action)
            if [ $# -lt 1 ]; then
                error "Usage: softeng action <keyword>"
            fi
            local keyword="$1"
            shift
            case "$keyword" in
                upr)
                    cmd_action_upr "$@"
                    ;;
                umergepr)
                    cmd_action_umergepr "$@"
                    ;;
                *)
                    error "Unknown action keyword: $keyword. Available: upr, umergepr"
                    ;;
            esac
            ;;
        change)
            if [ $# -lt 1 ]; then
                error "Usage: softeng change <subcommand> [args...]"
            fi
            local subcommand="$1"
            shift

            case "$subcommand" in
                new)
                    cmd_change_new "$@"
                    ;;
                archive)
                    cmd_change_archive "$@"
                    ;;
                archiveall)
                    cmd_change_archiveall "$@"
                    ;;
                *)
                    error "Unknown change subcommand: $subcommand. Available: new, archive, archiveall"
                    ;;
            esac
            ;;
        pr)
            if [ $# -lt 1 ]; then
                error "Usage: softeng pr <subcommand> [args...]"
            fi
            local subcommand="$1"
            shift

            case "$subcommand" in
                preflight)
                    cmd_pr_preflight "$@"
                    ;;
                create)
                    git_changepr "$@"
                    ;;
                *)
                    error "Unknown pr subcommand: $subcommand. Available: preflight, create"
                    ;;
            esac
            ;;
        diff)
            if [ $# -lt 1 ]; then
                error "Usage: softeng diff <target>"
            fi
            local target="$1"
            shift

            case "$target" in
                specs)
                    git_diff specs "$@"
                    ;;
                *)
                    error "Unknown diff target: $target. Available: specs"
                    ;;
            esac
            ;;
        status)
            if [ $# -lt 1 ]; then
                error "Usage: softeng status <subcommand> [args...]"
            fi
            local subcommand="$1"
            shift

            case "$subcommand" in
                ispr)
                    cmd_status_ispr "$@"
                    ;;
                *)
                    error "Unknown status subcommand: $subcommand. Available: ispr"
                    ;;
            esac
            ;;
        *)
            error "Unknown command: $command"
            ;;
    esac
}

main "$@"
