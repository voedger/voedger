#!/usr/bin/env bash
set -Eeuo pipefail

# root dir can be passed as $1, default '.'
root_dir=${1:-.}

FILE_EXT_FILTER='\.(go|vsql)$'

err=0
errstr=""

# Optional: default base ref if you want to force PR-style checks locally
# e.g. DEFAULT_BASE_REF="origin/main"
DEFAULT_BASE_REF="${DEFAULT_BASE_REF:-}"



check_file() {
  local filename=$1
  local content

  # read file once
  content=$(<"$filename") || return

  # skip generated files
  if [[ $content =~ Code[[:space:]]generated[[:space:]]by[[:space:]].*DO[[:space:]]NOT[[:space:]]EDIT ]]; then
    return
  fi

  # all allowed copyright variants in one regexp
  local copyright_re='Copyright \(c\) 202[0-9]-present ((unTill Pro|Sigma-Soft|Heeus), Ltd(., (unTill Pro|Sigma-Soft|Heeus), Ltd\.)?|unTill Software Development Group B\.V\.|Voedger Authors\.)'

  if [[ ! $content =~ $copyright_re ]]; then
    err=1
    errstr+=" $filename"
  fi
}

files_to_check() {
	# Ensure we're inside a git repo
	if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
		echo "Error: not inside a git repository" >&2
		return 1
	fi

	local added_files=""

	# When running in GitHub Actions on a pull request, try to get all files
	# added in this PR compared to its base branch.
	if [[ "${GITHUB_EVENT_NAME-}" == "pull_request" && -n "${GITHUB_BASE_REF-}" ]]; then
		local base_branch="$GITHUB_BASE_REF"
		local base_ref="origin/${base_branch}"

		# Ensure the base ref exists locally; fetch it if needed.
		if ! git rev-parse "$base_ref" >/dev/null 2>&1; then
			git fetch --no-tags --depth=1 origin "$base_branch" >/dev/null 2>&1 || true
		fi

		if git rev-parse "$base_ref" >/dev/null 2>&1; then
			local merge_base
			merge_base=$(git merge-base HEAD "$base_ref" 2>/dev/null || true)
			if [[ -n "$merge_base" ]]; then
				added_files=$(git diff --diff-filter=A --name-only "$merge_base...HEAD" || true)
			fi
		fi
	fi

	# Fallback for non-PR runs or when base cannot be determined:
	# use files added in the latest commit; if there is no parent, check all tracked files.
	if [[ -z "$added_files" ]]; then
		if git rev-parse HEAD^ >/dev/null 2>&1; then
			added_files=$(git diff --diff-filter=A --name-only HEAD^..HEAD || true)
		else
			added_files=$(git ls-tree -r --name-only HEAD || true)
		fi
	fi

	# Filter by extensions
	if [[ -n "$added_files" ]]; then
		grep -E "$FILE_EXT_FILTER" <<< "$added_files" || true
	fi
}

# Collect the result into an array
mapfile -t new_files < <(files_to_check)

# Exit 0 if empty
if ((${#new_files[@]} == 0)); then
	echo "No new files matching filter ($FILE_EXT_FILTER). Exiting."
	exit 0
fi

echo "New files to check:"
for filename in "${new_files[@]}"; do
	echo "  $filename"
done

for filename in "${new_files[@]}"; do
	check_file "$filename"
done

if (( err )); then
	echo "::error::Some new files:${errstr} - have no correct Copyright line"
	exit 1
fi
