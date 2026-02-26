#!/usr/bin/env bash
set -Eeuo pipefail

# conf.sh
#
# Description:
#   Manages uspecs lifecycle: install, update, upgrade, and invocation method configuration
#
# Usage:
#   conf.sh install --nlia [--alpha] [--pr]
#   conf.sh update [--pr]
#   conf.sh upgrade [--pr]
#   conf.sh im --add nlia
#   conf.sh im --remove nlic
#
# Internal commands (not for direct use):
#   conf.sh apply <install|update|upgrade> --project-dir <dir> --version <ver> [--current-version <ver>] [flags...]


REPO_OWNER="untillpro"
REPO_NAME="uspecs"
ALPHA_BRANCH="${USPECS_ALPHA_BRANCH:-main}"
GITHUB_API="https://api.github.com"
GITHUB_RAW="https://raw.githubusercontent.com"

case "$OSTYPE" in
    msys*|cygwin*) _TMP_BASE=$(cygpath -w "$TEMP") ;;
    *)             _TMP_BASE="/tmp" ;;
esac

_TEMP_DIRS=()
_TEMP_FILES=()

# checkcmds command1 [command2 ...]
# Verifies that each listed command is available on PATH.
# Prints an error message and exits with status 1 if any command is missing.
checkcmds() {
    local cmd
    for cmd in "$@"; do
        if ! command -v "$cmd" > /dev/null 2>&1; then
            echo "Error: required command not found: $cmd" >&2
            exit 1
        fi
    done
}

# get_pr_info <pr_sh_path> <map_nameref> [project_dir]
# Calls pr.sh info and parses the key=value output into the given associative array.
# Keys populated: pr_remote, default_branch
# project_dir: directory to run pr.sh from (defaults to $PWD)
# Returns non-zero if pr.sh info fails.
get_pr_info() {
    local pr_sh="$1"
    local -n _pr_info_map="$2"
    local project_dir="${3:-$PWD}"
    local output
    output=$(cd "$project_dir" && bash "$pr_sh" info) || return 1
    while IFS='=' read -r key value; do
        [[ -z "$key" ]] && continue
        _pr_info_map["$key"]="$value"
    done <<< "$output"
}

# is_tty
# Returns 0 if stdin is connected to a terminal, 1 if piped or redirected.
is_tty() {
    [ -t 0 ]
}

# is_git_repo <dir>
# Returns 0 if <dir> is inside a git repository, 1 otherwise.
is_git_repo() {
    local dir="$1"
    (cd "$dir" && git rev-parse --git-dir > /dev/null 2>&1)
}

# _GREP_BIN caches the resolved grep binary path for _grep.
_GREP_BIN=""

# _grep [grep-args...]
# Portable grep wrapper. On Windows (msys/cygwin) resolves grep from the git
# installation and fails fast if not found. On other platforms uses system grep.
_grep() {
    if [[ -z "$_GREP_BIN" ]]; then
        case "$OSTYPE" in
            msys*|cygwin*)
                # Use where.exe to get real Windows paths, then pick the grep
                # that lives inside the Git for Windows installation.
                local git_path git_root candidate
                git_path=$(where.exe git 2>/dev/null | head -1 | tr -d $'\r' | tr $'\\\\' / || true)
                if [[ -z "$git_path" ]]; then
                    echo "Error: git not found; cannot locate git's bundled grep" >&2
                    exit 1
                fi
                git_root=$(dirname "$(dirname "$git_path")")
                # Try direct path first (works even if grep is not on PATH).
                # Also try one level up to handle mingw64/bin/git.exe layout where
                # two dirnames give .../mingw64 instead of the git installation root.
                if [[ -x "$git_root/usr/bin/grep.exe" ]]; then
                    _GREP_BIN="$git_root/usr/bin/grep.exe"
                elif [[ -x "$(dirname "$git_root")/usr/bin/grep.exe" ]]; then
                    git_root=$(dirname "$git_root")
                    _GREP_BIN="$git_root/usr/bin/grep.exe"
                else
                    # Fall back to where.exe grep, pick the one under git root
                    while IFS= read -r candidate; do
                        candidate=$(echo "$candidate" | tr -d $'\r' | tr $'\\\\' /)
                        if [[ "$candidate" == "$git_root/"* ]]; then
                            _GREP_BIN="$candidate"
                            break
                        fi
                    done < <(where.exe grep 2>/dev/null || true)
                fi
                if [[ -z "$_GREP_BIN" ]]; then
                    echo "Error: grep not found under git root: $git_root" >&2
                    exit 1
                fi
                ;;
            *)
                _GREP_BIN="grep"
                ;;
        esac
    fi
    "$_GREP_BIN" "$@"
}

# sed_inplace file sed-args...
# Portable in-place sed. Uses -i.bak for BSD compatibility.
# Restores the original file on failure.
sed_inplace() {
    local file="$1"
    shift
    if ! sed -i.bak "$@" "$file"; then
        mv "${file}.bak" "$file" 2>/dev/null || true
        return 1
    fi
    rm -f "${file}.bak"
}

checkcmds curl

error() {
    echo "Error: $1" >&2
    exit 1
}

get_timestamp() {
    date -u +"%Y-%m-%dT%H:%M:%SZ"
}

native_path() {
    case "$OSTYPE" in
        msys*|cygwin*) cygpath -m "$1" ;;
        *)             echo "$1" ;;
    esac
}

get_project_dir() {
    local script_path="${BASH_SOURCE[0]}"
    if [[ -z "$script_path" || ! -f "$script_path" ]]; then
        error "Cannot determine project directory: script path is not available"
    fi
    local script_dir
    script_dir=$(cd "$(dirname "$script_path")" && pwd)
    # Go up 3 levels: scripts -> u -> uspecs -> project_dir
    local project_dir
    project_dir=$(cd "$script_dir/../../.." && pwd)
    native_path "$project_dir"
}

check_not_installed() {
    local project_dir="$1"
    if [[ -f "$project_dir/uspecs/u/uspecs.yml" ]]; then
        error "uspecs is already installed, use update instead"
    fi
}

check_installed() {
    local project_dir
    project_dir=$(get_project_dir)
    if [[ ! -f "$project_dir/uspecs/u/uspecs.yml" ]]; then
        error "uspecs is not installed"
    fi
}



load_config() {
    local project_dir="$1"
    local -n _config_map="$2"
    local metadata_file="$project_dir/uspecs/u/uspecs.yml"

    if [[ ! -f "$metadata_file" ]]; then
        return 0
    fi

    while IFS= read -r line || [[ -n "$line" ]]; do
        [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
        local key value
        key="${line%%:*}"
        value="${line#*: }"
        value=$(echo "$value" | sed 's/^\[\(.*\)\]$/\1/')
        _config_map["$key"]="$value"
    done < "$metadata_file"
}

get_latest_tag() {
    curl -fsSL "$GITHUB_API/repos/$REPO_OWNER/$REPO_NAME/tags" | \
        _grep '"name":' | \
        sed 's/.*"name": *"v\?\([^"]*\)".*/\1/' | \
        head -n 1
}

get_latest_minor_tag() {
    local current_version="$1"
    local major minor
    IFS='.' read -r major minor _ <<< "$current_version"

    local result
    result=$(curl -fsSL "$GITHUB_API/repos/$REPO_OWNER/$REPO_NAME/tags" | \
        _grep '"name":' | \
        sed 's/.*"name": *"v\?\([^"]*\)".*/\1/' | \
        _grep "^$major\.$minor\." | \
        head -n 1 || true)
    echo "${result:-$current_version}"
}

get_latest_major_tag() {
    get_latest_tag
}

get_latest_commit_info() {
    local response
    response=$(curl -fsSL "$GITHUB_API/repos/$REPO_OWNER/$REPO_NAME/commits/$ALPHA_BRANCH")
    local sha
    sha=$(echo "$response" | _grep '"sha":' | head -n 1 | sed 's/.*"sha": *"\([^"]*\)".*/\1/')
    local commit_date
    commit_date=$(echo "$response" | _grep '"date":' | head -n 1 | sed 's/.*"date": *"\([^"]*\)".*/\1/')
    echo "$sha $commit_date"
}

get_alpha_version() {
    curl -fsSL "$GITHUB_RAW/$REPO_OWNER/$REPO_NAME/$ALPHA_BRANCH/version.txt" | tr -d '[:space:]'
}

is_alpha_version() {
    [[ "$1" == *-a* ]]
}

download_archive() {
    local ref="$1"
    local temp_dir="$2"

    local archive_url="https://github.com/$REPO_OWNER/$REPO_NAME/archive/$ref.tar.gz"
    curl -fsSL "$archive_url" | tar -xz -C "$temp_dir" --strip-components=1
}

get_nli_file() {
    local method="$1"
    case "$method" in
        nlia) echo "AGENTS.md" ;;
        nlic) echo "CLAUDE.md" ;;
        *)
            echo "Warning: Unknown invocation method: $method" >&2
            return 1
            ;;
    esac
}

resolve_version_ref() {
    local version="$1"
    local commit="${2:-}"
    if is_alpha_version "$version"; then
        echo "$commit"
    else
        echo "v$version"
    fi
}

format_version_string() {
    local version="$1"
    local commit="$2"
    local commit_timestamp="$3"

    if [[ -n "$commit_timestamp" ]]; then
        echo "${version}, ${commit_timestamp}"
    else
        echo "$version"
    fi
}

sanitize_branch_name() {
    local name="$1"
    name="${name//[^a-zA-Z0-9._/-]/-}"
    while [[ "$name" == *".."* ]]; do name="${name/../-}"; done
    name="${name//@\{/-}"; name="${name//\/./\/-}"; name="${name//\/\//\/}"
    name="${name#/}"; name="${name%/}"; name="${name%.}"; name="${name%.lock}"
    [[ -z "$name" || "$name" == "@" ]] && name="branch"
    echo "$name"
}

format_version_string_branch() {
    local version="$1"
    local commit="$2"
    local commit_timestamp="$3"

    if [[ -n "$commit_timestamp" ]]; then
        # Replace colons with hyphens for git branch name (YYYY-MM-DDTHH-MM-SSZ)
        local timestamp_safe="${commit_timestamp//:/-}"
        local result="${version}-${timestamp_safe}"
    else
        local result="$version"
    fi

    # Sanitize to ensure valid git branch name
    sanitize_branch_name "$result"
}

cleanup_temp() {
    if [[ ${#_TEMP_FILES[@]} -gt 0 ]]; then
        for file in "${_TEMP_FILES[@]}"; do
            rm -f "$file"
        done
    fi
    if [[ ${#_TEMP_DIRS[@]} -gt 0 ]]; then
        for dir in "${_TEMP_DIRS[@]}"; do
            rm -rf "$dir"
        done
    fi
}
trap cleanup_temp EXIT

create_temp_dir() {
    local temp_dir
    temp_dir=$(mktemp -d "$_TMP_BASE/uspecs.XXXXXX")
    _TEMP_DIRS+=("$temp_dir")
    echo "$temp_dir"
}

create_temp_file() {
    local temp_file
    temp_file=$(mktemp "$_TMP_BASE/uspecs.XXXXXX")
    _TEMP_FILES+=("$temp_file")
    echo "$temp_file"
}

show_operation_plan() {
    local operation="$1"
    local current_version="${2:-}"
    local target_version="$3"
    local commit="${4:-}"
    local commit_timestamp="${5:-}"
    local invocation_methods="${6:-}"
    local pr_flag="${7:-false}"
    local project_dir="${8:-}"
    local script_dir="${9:-}"

    echo ""
    echo "=========================================="
    echo "Operation: $operation"
    echo "=========================================="

    # Incoming
    echo "Incoming version:"
    echo "  Version: $target_version"
    if is_alpha_version "$target_version" && [[ -n "$commit" ]]; then
        echo "  Commit: $commit"
        echo "  Timestamp: $commit_timestamp"
    fi
    echo "  Endpoint: $GITHUB_API/repos/$REPO_OWNER/$REPO_NAME/commits/$ALPHA_BRANCH"

    echo ""

    # Existing version (skipped for install)
    if [[ "$operation" != "install" ]]; then
        echo "Existing version:"
        if [[ -n "$current_version" ]]; then
            echo "  Version: $current_version"
            if is_alpha_version "$current_version"; then
                local -A current_config
                load_config "$project_dir" current_config
                local current_commit="${current_config[commit]:-}"
                local current_commit_timestamp="${current_config[commit_timestamp]:-}"
                if [[ -n "$current_commit" ]]; then
                    echo "  Commit: $current_commit"
                    echo "  Timestamp: $current_commit_timestamp"
                fi
            fi
        fi
        echo "  Project folder: $project_dir"
        echo "  uspecs core: uspecs/u"

        if [[ -n "$invocation_methods" ]]; then
            echo "  Natural language invocation files:"
            IFS=',' read -ra methods_array <<< "$invocation_methods"
            for method in "${methods_array[@]}"; do
                method=$(echo "$method" | xargs)
                local file
                file=$(get_nli_file "$method") 2>/dev/null || continue
                echo "    - $file"
            done
        fi
    fi

    # Pull request (if enabled)
    if [[ "$pr_flag" == "true" && -n "$script_dir" ]]; then
        echo ""
        echo "Pull request:"

        local -A pr_info
        local pr_remote="" default_branch="" target_repo_url="" pr_branch=""
        if get_pr_info "$script_dir/_lib/pr.sh" pr_info "$project_dir" 2>/dev/null; then
            pr_remote="${pr_info[pr_remote]:-}"
            default_branch="${pr_info[default_branch]:-}"
            target_repo_url=$(git -C "$project_dir" remote get-url "$pr_remote" 2>/dev/null)

            # Use branch-safe version string for PR branch name
            local version_branch
            version_branch=$(format_version_string_branch "$target_version" "$commit" "$commit_timestamp")
            pr_branch="${operation}-uspecs-${version_branch}"

            echo "  Target remote: $pr_remote"
            echo "  Target repo: $target_repo_url"
            echo "  Base branch: $default_branch"
            echo "  PR branch: $pr_branch"
        else
            echo "  Failed to determine PR details"
        fi
    fi
    echo "=========================================="
}

confirm_action() {
    local action="$1"

    echo ""

    # Try to read from /dev/tty (works even when stdin is piped)
    if [ -e /dev/tty ]; then
        read -p "Proceed with $action? (y/n) " -n 1 -r < /dev/tty
    elif [[ -t 0 ]]; then
        # Stdin is a terminal (not piped)
        read -p "Proceed with $action? (y/n) " -n 1 -r
    else
        # Non-interactive (CI, containers), auto-accept
        return 0
    fi

    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "${action^} cancelled"
        return 1
    fi
}

replace_uspecs_u() {
    local source_dir="$1"
    local project_dir="$2"
    echo "Removing installation metadata file from archive..."
    rm -f "$source_dir/uspecs/u/uspecs.yml"
    echo "Removing old uspecs/u files..."
    find "$project_dir/uspecs/u" -type f -delete
    echo "Installing new uspecs/u..."
    cp -r "$source_dir/uspecs/u" "$project_dir/uspecs/"
}

upgrade_markers() {
    local file="$1"
    sed_inplace "$file" "s/<!-- uspecs:triggering_instructions:begin -->/<!-- uspecs:begin -->/g; s/<!-- uspecs:triggering_instructions:end -->/<!-- uspecs:end -->/g"
}

# Check that both begin and end markers are present in a file.
# Returns 0 if both found, 1 otherwise.
has_markers() {
    local file="$1"
    local begin_marker="$2"
    local end_marker="$3"
    _grep -q "$begin_marker" "$file" && _grep -q "$end_marker" "$file"
}

inject_instructions() {
    local source_file="$1"
    local target_file="$2"

    local begin_marker="<!-- uspecs:begin -->"
    local end_marker="<!-- uspecs:end -->"

    # Upgrade old markers in target first, so we always work with new markers below
    if [[ -f "$target_file" ]]; then
        upgrade_markers "$target_file"
    fi

    if [[ ! -f "$source_file" ]]; then
        echo "Warning: Source file not found: $source_file" >&2
        return 1
    fi

    local temp_extract
    temp_extract=$(create_temp_file)
    sed -n "/$begin_marker/,/$end_marker/p" "$source_file" > "$temp_extract"

    if [[ ! -s "$temp_extract" ]]; then
        echo "Warning: No triggering instructions found in $source_file" >&2
        return 1
    fi

    if [[ ! -f "$target_file" ]]; then
        {
            echo "# Agents instructions"
            echo ""
            cat "$temp_extract"
        } > "$target_file"
        return 0
    fi

    if ! has_markers "$target_file" "$begin_marker" "$end_marker"; then
        {
            echo ""
            cat "$temp_extract"
        } >> "$target_file"
        return 0
    fi

    local temp_output
    temp_output=$(create_temp_file)
    sed "/$begin_marker/,\$d" "$target_file" > "$temp_output"
    cat "$temp_extract" >> "$temp_output"
    sed "1,/$end_marker/d" "$target_file" >> "$temp_output"
    cat "$temp_output" > "$target_file"
}

remove_instructions() {
    local target_file="$1"

    if [[ ! -f "$target_file" ]]; then
        return 0
    fi

    local begin_marker="<!-- uspecs:begin -->"
    local end_marker="<!-- uspecs:end -->"

    upgrade_markers "$target_file"

    if ! has_markers "$target_file" "$begin_marker" "$end_marker"; then
        return 0
    fi

    sed_inplace "$target_file" "/$begin_marker/,/$end_marker/d"
}

write_metadata() {
    local project_dir="$1"
    local version="$2"
    local invocation_methods="$3"
    local commit="${4:-}"
    local commit_timestamp="${5:-}"
    local installed_at="${6:-}"

    local metadata_file="$project_dir/uspecs/u/uspecs.yml"
    local timestamp
    timestamp=$(get_timestamp)

    if [[ -z "$installed_at" ]]; then
        installed_at="$timestamp"
    fi

    {
        echo "# uspecs installation metadata"
        echo "# DO NOT EDIT - managed by uspecs"
        echo "version: $version"
        echo "invocation_methods: [$invocation_methods]"
        echo "installed_at: $installed_at"
        echo "modified_at: $timestamp"
        if [[ -n "$commit" ]]; then
            echo "commit: $commit"
            echo "commit_timestamp: $commit_timestamp"
        fi
    } > "$metadata_file"
}

resolve_update_version() {
    local current_version="$1"
    local project_dir="$2"
    local -n _ruv_target_version="$3"
    local -n _ruv_target_ref="$4"
    local -n _ruv_commit="$5"
    local -n _ruv_commit_timestamp="$6"

    if is_alpha_version "$current_version"; then
        echo "Checking for alpha updates..."
        local -A config
        load_config "$project_dir" config
        local current_commit="${config[commit]:-}"
        local current_commit_timestamp="${config[commit_timestamp]:-}"
        local fetched_commit fetched_timestamp
        read -r fetched_commit fetched_timestamp <<< "$(get_latest_commit_info)"

        if [[ "$current_commit" == "$fetched_commit" ]]; then
            echo "Already on the latest alpha version: $current_version"
            echo "  Commit: $fetched_commit"
            echo "  Timestamp: $current_commit_timestamp"
            return 1
        fi
        _ruv_target_version=$(get_alpha_version)
        _ruv_target_ref="$fetched_commit"
        _ruv_commit="$fetched_commit"
        _ruv_commit_timestamp="$fetched_timestamp"
    else
        echo "Checking for stable updates..."
        _ruv_target_version=$(get_latest_minor_tag "$current_version")

        if [[ "$_ruv_target_version" == "$current_version" ]]; then
            echo "Already on the latest stable minor version: $current_version"

            local latest_major
            latest_major=$(get_latest_major_tag)
            if [[ "$latest_major" != "$current_version" ]]; then
                echo ""
                echo "Upgrade available to version $latest_major"
                echo "Use 'conf.sh upgrade' command"
            fi
            return 1
        fi

        _ruv_target_ref="v$_ruv_target_version"
    fi
    return 0
}

resolve_upgrade_version() {
    local current_version="$1"
    local project_dir="$2"
    local -n _rugv_target_version="$3"
    local -n _rugv_target_ref="$4"

    if is_alpha_version "$current_version"; then
        error "Only applicable for stable versions. Alpha versions always track the latest commit from $ALPHA_BRANCH branch, use update instead"
    fi

    echo "Checking for major upgrades..."
    _rugv_target_version=$(get_latest_major_tag)

    if [[ "$_rugv_target_version" == "$current_version" ]]; then
        echo "Already on the latest major version: $current_version"
        return 1
    fi

    _rugv_target_ref="v$_rugv_target_version"
    return 0
}

# Re-invoked by install/update/upgrade commands via target version's conf.sh
cmd_apply() {
    if [[ $# -lt 1 ]]; then
        error "Usage: conf.sh apply <install|update|upgrade> [flags...]"
    fi

    local command_name="$1"
    shift

    local project_dir="" version="" commit="" commit_timestamp="" pr_flag=false
    local current_version=""
    local invocation_methods=()

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --project-dir) project_dir=$(native_path "$2"); shift 2 ;;
            --version) version="$2"; shift 2 ;;
            --commit) commit="$2"; shift 2 ;;
            --commit-timestamp) commit_timestamp="$2"; shift 2 ;;
            --current-version) current_version="$2"; shift 2 ;;
            --pr) pr_flag=true; shift ;;
            --nlia) invocation_methods+=("nlia"); shift ;;
            --nlic) invocation_methods+=("nlic"); shift ;;
            *) error "Unknown flag: $1" ;;
        esac
    done

    [[ -z "$project_dir" ]] && error "--project-dir is required"
    [[ -z "$version" ]] && error "--version is required"
    [[ "$command_name" != "install" && -z "$current_version" ]] && error "--current-version is required for update/upgrade"
    [[ ! -d "$project_dir" ]] && error "Project directory not found: $project_dir"

    local script_dir
    script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
    local source_dir
    source_dir=$(cd "$script_dir/../../.." && pwd)

    local version_string
    version_string=$(format_version_string "$version" "$commit" "$commit_timestamp")

    # Safe version to create branches
    local version_string_branch
    version_string_branch=$(format_version_string_branch "$version" "$commit" "$commit_timestamp")

    local metadata_file="$project_dir/uspecs/u/uspecs.yml"

    if [[ "$command_name" == "install" && -f "$metadata_file" ]]; then
        error "uspecs is already installed, use update instead"
    fi

    # PR: remember current branch, fast-forward default branch (may update local uspecs.yml)
    local prev_branch=""
    if [[ "$pr_flag" == "true" ]]; then
        prev_branch=$(git -C "$project_dir" symbolic-ref --short HEAD)
        (cd "$project_dir" && bash "$script_dir/_lib/pr.sh" ffdefault)
    fi

    local -A config
    if [[ "$command_name" == "install" ]]; then
        if [[ -f "$metadata_file" ]]; then
            error "uspecs is already installed, use update instead"
        fi
    else
        load_config "$project_dir" config
        if [[ "${config[version]:-}" != "$current_version" ]]; then
            error "Installed version '${config[version]:-}' does not match expected '$current_version'. Re-run the command to pick up the current installed version."
        fi
        # After ffdefault the local uspecs.yml may already reflect the incoming version
        if [[ -n "$commit" && "${config[commit]:-}" == "$commit" ]] || \
           [[ -z "$commit" && "${config[version]:-}" == "$version" ]]; then
            echo "Already up to date"
            [[ -n "$prev_branch" ]] && git -C "$project_dir" checkout "$prev_branch"
            return 0
        fi
    fi

    # Determine invocation methods string for plan display
    local plan_invocation_methods_str=""
    if [[ "$command_name" == "install" ]]; then
        plan_invocation_methods_str=$(IFS=', '; echo "${invocation_methods[*]}")
    elif [[ -f "$metadata_file" ]]; then
        plan_invocation_methods_str="${config[invocation_methods]:-}"
    fi

    # Show operation plan and confirm
    show_operation_plan "$command_name" "$current_version" "$version" "$commit" "$commit_timestamp" "$plan_invocation_methods_str" "$pr_flag" "$project_dir" "$script_dir"
    if ! confirm_action "$command_name"; then
        [[ -n "$prev_branch" ]] && git -C "$project_dir" checkout "$prev_branch"
        return 0
    fi

    # PR: create feature branch from default branch
    local branch_name="${command_name}-uspecs-${version_string_branch}"
    if [[ "$pr_flag" == "true" ]]; then
        (cd "$project_dir" && bash "$script_dir/_lib/pr.sh" prbranch "$branch_name")
    fi

    # Save existing metadata for update/upgrade
    local invocation_methods_str="" installed_at=""

    if [[ "$command_name" != "install" ]]; then
        [[ ! -f "$metadata_file" ]] && error "Installation metadata file not found: $metadata_file"
        invocation_methods_str="${config[invocation_methods]:-}"
        installed_at="${config[installed_at]:-}"
    else
        invocation_methods_str=$(IFS=', '; echo "${invocation_methods[*]}")
    fi

    if [[ "$command_name" == "install" ]]; then
        rm -f "$source_dir/uspecs/u/uspecs.yml"
        echo "Installing uspecs/u..."
        mkdir -p "$project_dir/uspecs"
        cp -r "$source_dir/uspecs/u" "$project_dir/uspecs/"
    else
        replace_uspecs_u "$source_dir" "$project_dir"
    fi

    # Write metadata
    echo "Writing installation metadata..."
    write_metadata "$project_dir" "$version" "$invocation_methods_str" "$commit" "$commit_timestamp" "$installed_at"

    # Inject NLI instructions
    echo "Injecting instructions..."
    IFS=',' read -ra inject_methods <<< "$invocation_methods_str"
    for method in "${inject_methods[@]}"; do
        method=$(echo "$method" | xargs)
        local file
        file=$(get_nli_file "$method") || continue
        inject_instructions "$source_dir/$file" "$project_dir/$file"
        echo "  - $file"
    done

    # PR: commit, push, and create pull request
    local pr_url="" pr_branch="" pr_base=""
    if [[ "$pr_flag" == "true" ]]; then
        local pr_title="uspecs ${version_string}"
        local pr_body="$pr_title"
        local pr_info_file
        pr_info_file=$(create_temp_file)

        # Capture PR info from stderr while showing normal output
        (cd "$project_dir" && bash "$script_dir/_lib/pr.sh" pr --title "$pr_title" --body "$pr_body" \
            --next-branch "$prev_branch" --delete-branch) 2> "$pr_info_file"

        # Parse PR info from temp file
        pr_url=$(_grep '^PR_URL=' "$pr_info_file" | cut -d= -f2-)
        pr_branch=$(_grep '^PR_BRANCH=' "$pr_info_file" | cut -d= -f2)
        pr_base=$(_grep '^PR_BASE=' "$pr_info_file" | cut -d= -f2)
    fi

    echo ""
    echo "${command_name^} completed successfully!"

    # Display PR summary if created
    if [[ "$pr_flag" == "true" && -n "$pr_url" ]]; then
        echo ""
        echo "=========================================="
        echo "Pull Request created"
        echo "=========================================="
        echo "URL: $pr_url"
        echo "Branch: $pr_branch -> $pr_base"
        echo "=========================================="
    fi
}

cmd_install() {
    local alpha=false
    local pr_flag=false
    local invocation_methods=()

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --alpha) alpha=true; shift ;;
            --pr) pr_flag=true; shift ;;
            --nlia) invocation_methods+=("nlia"); shift ;;
            --nlic) invocation_methods+=("nlic"); shift ;;
            *) error "Unknown flag: $1" ;;
        esac
    done

    if [[ ${#invocation_methods[@]} -eq 0 ]]; then
        error "At least one invocation method (--nlia or --nlic) is required"
    fi

    local project_dir
    project_dir=$PWD

    check_not_installed "$project_dir"

    local ref version commit="" commit_timestamp=""
    if [[ "$alpha" == "true" ]]; then
        echo "Fetching latest alpha version..."
        version=$(get_alpha_version)
        read -r commit commit_timestamp <<< "$(get_latest_commit_info)"
        ref="$commit"
        echo "Latest alpha version: $version"
    else
        echo "Fetching latest stable version..."
        version=$(get_latest_tag)
        ref="v$version"
        echo "Latest version: $version"
    fi

    local temp_dir
    temp_dir=$(create_temp_dir)

    echo "Downloading uspecs..."
    download_archive "$ref" "$temp_dir"

    local apply_args=("install" "--project-dir" "$project_dir" "--version" "$version")
    for method in "${invocation_methods[@]}"; do
        apply_args+=("--$method")
    done
    if [[ -n "$commit" ]]; then
        apply_args+=("--commit" "$commit" "--commit-timestamp" "$commit_timestamp")
    fi
    if [[ "$pr_flag" == "true" ]]; then
        apply_args+=("--pr")
    fi

    echo "Running install..."
    bash "$temp_dir/uspecs/u/scripts/conf.sh" apply "${apply_args[@]}"
}

cmd_update_or_upgrade() {
    local command_name="$1"
    shift

    local pr_flag=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --pr)
                pr_flag=true
                shift
                ;;
            *)
                error "Unknown flag: $1"
                ;;
        esac
    done

    check_installed

    local project_dir
    project_dir=$(get_project_dir)

    local -A config
    load_config "$project_dir" config
    local current_version="${config[version]:-}"

    local target_version="" target_ref="" commit="" commit_timestamp=""
    if [[ "$command_name" == "update" ]]; then
        resolve_update_version "$current_version" "$project_dir" target_version target_ref commit commit_timestamp || return 0
    else
        resolve_upgrade_version "$current_version" "$project_dir" target_version target_ref || return 0
    fi

    local temp_dir
    temp_dir=$(create_temp_dir)

    echo "Downloading uspecs..."
    download_archive "$target_ref" "$temp_dir"

    local apply_args=("$command_name" "--project-dir" "$project_dir" "--version" "$target_version")
    apply_args+=("--current-version" "$current_version")
    if [[ -n "${commit:-}" ]]; then
        apply_args+=("--commit" "$commit" "--commit-timestamp" "$commit_timestamp")
    fi
    if [[ "$pr_flag" == "true" ]]; then
        apply_args+=("--pr")
    fi

    echo "Running ${command_name}..."
    bash "$temp_dir/uspecs/u/scripts/conf.sh" apply "${apply_args[@]}"
}

cmd_im() {
    local add_methods=()
    local remove_methods=()

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --add) add_methods+=("$2"); shift 2 ;;
            --remove) remove_methods+=("$2"); shift 2 ;;
            *) error "Unknown flag: $1" ;;
        esac
    done

    if [[ ${#add_methods[@]} -eq 0 && ${#remove_methods[@]} -eq 0 ]]; then
        error "At least one --add or --remove flag is required"
    fi

    check_installed

    local project_dir
    project_dir=$(get_project_dir)

    local -A config
    load_config "$project_dir" config

    local current_methods="${config[invocation_methods]:-}"

    IFS=',' read -ra methods_array <<< "$current_methods"
    local -A methods_map
    for method in "${methods_array[@]}"; do
        method=$(echo "$method" | xargs)
        methods_map["$method"]=1
    done

    local version="${config[version]}"

    local ref
    ref=$(resolve_version_ref "$version" "${config[commit]:-}")

    local changed=false
    local temp_source=""

    for method in "${add_methods[@]}"; do
        if [[ -n "${methods_map[$method]:-}" ]]; then
            echo "Invocation method '$method' is already configured"
            continue
        fi

        if [[ -z "$temp_source" ]]; then
            temp_source=$(create_temp_file)
            echo "Downloading source file for triggering instructions..."
            local source_url="$GITHUB_RAW/$REPO_OWNER/$REPO_NAME/$ref/AGENTS.md"
            if ! curl -fsSL "$source_url" -o "$temp_source"; then
                error "Failed to download source file from $source_url"
            fi
        fi

        local file
        file=$(get_nli_file "$method") || continue
        inject_instructions "$temp_source" "$project_dir/$file"
        echo "Added invocation method: $method ($file)"
        methods_map["$method"]=1
        changed=true
    done

    for method in "${remove_methods[@]}"; do
        if [[ -z "${methods_map[$method]:-}" ]]; then
            echo "Invocation method '$method' is not configured"
            continue
        fi
        local file
        file=$(get_nli_file "$method") || continue
        remove_instructions "$project_dir/$file"
        echo "Removed invocation method: $method ($file)"
        unset "methods_map[$method]"
        changed=true
    done

    # Build new methods string preserving order from original
    local new_methods_array=()
    for method in "${methods_array[@]}"; do
        method=$(echo "$method" | xargs)
        if [[ -n "${methods_map[$method]:-}" ]]; then
            new_methods_array+=("$method")
        fi
    done
    # Add any new methods that were successfully added (present in methods_map)
    for method in "${add_methods[@]}"; do
        if [[ -z "${methods_map[$method]:-}" ]]; then
            continue
        fi
        local found=0
        for existing in "${new_methods_array[@]}"; do
            if [[ "$existing" == "$method" ]]; then
                found=1
                break
            fi
        done
        if [[ $found -eq 0 ]]; then
            new_methods_array+=("$method")
        fi
    done

    local new_methods_str
    new_methods_str=$(IFS=', '; echo "${new_methods_array[*]}")

    if [[ "$changed" == "true" ]]; then
        echo "Updating installation metadata..."
        local metadata_file="$project_dir/uspecs/u/uspecs.yml"
        local timestamp
        timestamp=$(get_timestamp)

        sed_inplace "$metadata_file" \
            -e "s/^invocation_methods: .*/invocation_methods: [$new_methods_str]/" \
            -e "s/^modified_at: .*/modified_at: $timestamp/"

        echo ""
        echo "Invocation methods updated successfully!"
    else
        echo ""
        echo "Nothing to change."
    fi
}

main() {
    if [[ $# -lt 1 ]]; then
        error "Usage: conf.sh <command> [args...]"
    fi

    local command="$1"
    shift

    case "$command" in
        install)
            cmd_install "$@"
            ;;
        update)
            cmd_update_or_upgrade "update" "$@"
            ;;
        upgrade)
            cmd_update_or_upgrade "upgrade" "$@"
            ;;
        apply)
            cmd_apply "$@"
            ;;
        im)
            cmd_im "$@"
            ;;
        *)
            error "Unknown command: $command. Available: install, update, upgrade, apply, im"
            ;;
    esac
}

main "$@"
