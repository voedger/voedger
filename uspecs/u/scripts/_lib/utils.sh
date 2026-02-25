#!/usr/bin/env bash
set -Eeuo pipefail

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
