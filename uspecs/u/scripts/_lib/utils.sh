#!/usr/bin/env bash

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

# get_pr_info <pr_sh_path> <map_nameref>
# Calls pr.sh info and parses the key=value output into the given associative array.
# Keys populated: pr_remote, default_branch
# Returns non-zero if pr.sh info fails.
get_pr_info() {
    local pr_sh="$1"
    local -n _pr_info_map="$2"
    local output
    output=$(bash "$pr_sh" info) || return 1
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
