#!/usr/bin/env bash
set -Eeuo pipefail

# uspecs automation
#
# Usage:
#   uspecs change frontmatter <path-to-change-md> [--issue-url <url>]
#   uspecs change archive <path-to-change-folder>
#
# change frontmatter:
#   Adds frontmatter to existing change.md with auto-generated metadata:
#     - registered_at: YYYY-MM-DDTHH:MM:SSZ
#     - change_id: ymdHM-<change-name-kebab-case>
#     - baseline: <commit-hash> (if git repository)
#     - issue_url: <url> (if --issue-url provided)
#
# change archive:
#   Archives change folder to <changes-folder>/archive/yymm/ymdHM-<change-name>
#   Adds archived_at metadata and updates folder date prefix

error() {
    echo "Error: $1" >&2
    exit 1
}

get_timestamp() {
    date -u +"%Y-%m-%dT%H:%M:%SZ"
}

get_baseline() {
    if git rev-parse --git-dir > /dev/null 2>&1; then
        git rev-parse HEAD 2>/dev/null || echo ""
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
    count=$(grep -r "^\s*-\s*\[ \]" "$folder"/*.md 2>/dev/null | wc -l)
    echo "${count:-0}" | tr -d ' '
}

extract_change_name() {
    local folder_name="$1"
    echo "$folder_name" | sed 's/^[0-9]\{10\}-//'
}

move_folder() {
    local source="$1"
    local destination="$2"
    if git rev-parse --git-dir > /dev/null 2>&1; then
        git mv "$source" "$destination" 2>/dev/null || mv "$source" "$destination"
    else
        mv "$source" "$destination"
    fi
}

cmd_change_frontmatter() {
    local path_to_change_md=""
    local issue_url=""

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --issue-url)
                if [[ $# -lt 2 || -z "$2" ]]; then
                    error "--issue-url requires a URL argument"
                fi
                issue_url="$2"
                shift 2
                ;;
            *)
                if [ -z "$path_to_change_md" ]; then
                    path_to_change_md="$1"
                    shift
                else
                    error "Unknown argument: $1"
                fi
                ;;
        esac
    done

    if [ -z "$path_to_change_md" ]; then
        error "path-to-change-md is required"
    fi

    if [ ! -f "$path_to_change_md" ]; then
        error "File not found: $path_to_change_md"
    fi

    local change_folder
    change_folder=$(dirname "$path_to_change_md")

    local folder_name
    folder_name=$(basename "$change_folder")

    if [[ ! "$folder_name" =~ ^[0-9]{10}- ]]; then
        error "Change folder must follow format ymdHM-change-name: $folder_name"
    fi

    local content
    content=$(cat "$path_to_change_md")

    if [ -z "$content" ]; then
        error "File is empty: $path_to_change_md"
    fi

    local timestamp baseline
    timestamp=$(get_timestamp)
    baseline=$(get_baseline)

    local metadata="---"$'\n'
    metadata+="registered_at: $timestamp"$'\n'
    metadata+="change_id: $folder_name"$'\n'

    if [ -n "$baseline" ]; then
        metadata+="baseline: $baseline"$'\n'
    fi

    if [ -n "$issue_url" ]; then
        metadata+="issue_url: $issue_url"$'\n'
    fi

    metadata+="---"

    local temp_file
    temp_file=$(mktemp)
    {
        echo "$metadata"
        echo ""
        echo "$content"
    } > "$temp_file"

    mv "$temp_file" "$path_to_change_md" || {
        rm -f "$temp_file"
        error "Failed to update change.md"
    }

    echo "Added frontmatter to: $path_to_change_md"
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
        if ! sed -i.bak -E 's#\]\(\.\./#](../../../#g' "$file"; then
            error "Failed to convert links in file: $file"
        fi
        rm -f "${file}.bak"
    done <<< "$md_files"

    return 0
}

cmd_change_archive() {
    local path_to_change_folder="$1"

    if [ -z "$path_to_change_folder" ]; then
        error "path-to-change-folder is required"
    fi

    if [ ! -d "$path_to_change_folder" ]; then
        error "Folder not found: $path_to_change_folder"
    fi

    local change_file="$path_to_change_folder/change.md"
    if [ ! -f "$change_file" ]; then
        error "change.md not found in folder: $path_to_change_folder"
    fi

    local folder_name
    folder_name=$(basename "$path_to_change_folder")

    if [[ "$path_to_change_folder" == */archive/* ]]; then
        error "Folder is already in archive: $path_to_change_folder"
    fi

    local uncompleted_count
    uncompleted_count=$(count_uncompleted_items "$path_to_change_folder")

    if [ "$uncompleted_count" -gt 0 ]; then
        echo "Cannot archive: $uncompleted_count uncompleted todo item(s) found"
        echo ""
        echo "Uncompleted items:"
        grep -rn "^\s*-\s*\[ \]" "$path_to_change_folder"/*.md 2>/dev/null | sed 's/^/  /'
        echo ""
        echo "Complete or cancel todo items before archiving"
        exit 0
    fi

    local timestamp
    timestamp=$(get_timestamp)

    # Insert archived_at into YAML front matter (before closing ---)
    local temp_file
    temp_file=$(mktemp)
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
        { print }
    ' "$change_file" > "$temp_file"
    if mv "$temp_file" "$change_file"; then
        :  # Success, continue
    else
        rm -f "$temp_file"
        return 1
    fi

    # Add ../ prefix to relative links for archive folder depth
    if ! convert_links_to_relative "$path_to_change_folder"; then
        error "Failed to convert links to relative paths"
    fi

    local changes_folder
    changes_folder=$(dirname "$path_to_change_folder")

    local archive_folder="$changes_folder/archive"

    local date_prefix
    date_prefix=$(date -u +"%y%m%d%H%M")

    # Extract yymm for subfolder
    local yymm_prefix="${date_prefix:0:4}"

    local archive_subfolder="$archive_folder/$yymm_prefix"
    mkdir -p "$archive_subfolder"

    local change_name
    change_name=$(extract_change_name "$folder_name")

    local archive_path="$archive_subfolder/${date_prefix}-${change_name}"

    if [ -d "$archive_path" ]; then
        error "Archive folder already exists: $archive_path"
    fi

    move_folder "$path_to_change_folder" "$archive_path"

    echo "Archived change: $archive_path"
}

main() {
    if [ $# -lt 1 ]; then
        error "Usage: uspecs <command> [args...]"
    fi

    local command="$1"
    shift

    case "$command" in
        change)
            if [ $# -lt 1 ]; then
                error "Usage: uspecs change <subcommand> [args...]"
            fi
            local subcommand="$1"
            shift

            case "$subcommand" in
                frontmatter)
                    cmd_change_frontmatter "$@"
                    ;;
                archive)
                    cmd_change_archive "$@"
                    ;;
                *)
                    error "Unknown change subcommand: $subcommand"
                    ;;
            esac
            ;;
        *)
            error "Unknown command: $command"
            ;;
    esac
}

main "$@"
