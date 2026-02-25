# Action: Create pull request

## Overview

Create a pull request from the current change branch by squash-merging it into a dedicated PR branch and submitting via GitHub CLI.

## Instructions

Rules:

- Strictly follow the definitions from `uspecs/u/concepts.md` and `uspecs/u/conf.md`
- Always call `uspecs.sh` for git/PR operations — never call `_lib/pr.sh` directly
- Read `change.md` frontmatter to determine `issue_url` and `issue_id`

Parameters:

- Input
  - Active Change Folder (change.md for issue_url)
  - change_branch: current git branch
- Output
  - PR created on GitHub
  - pr_branch: `{change_branch}--pr` with squashed commits
  - change_branch deleted

Flow:

- Merge default branch into change_branch:
  - Run `bash uspecs/u/scripts/uspecs.sh pr mergedef`
  - If script exits with error: report the error and stop
  - Parse `change_branch`, `default_branch`, and `change_branch_head` from script output
- Read Active Change Folder (change.md) to determine `issue_url` (may be absent) and derive `issue_id` from the URL (last path segment)
- Present Engineer with the following options:
    1. Create PR (squash-merge `change_branch` into `{change_branch}--pr`, delete `change_branch`, create PR on GitHub)
    2. Cancel
  - On option 2: stop
- Get specs diff to derive PR title and body:
  - Run `bash uspecs/u/scripts/uspecs.sh diff specs`
  - From the diff output identify `draft_title` and `draft_body`; construct `pr_title` and `pr_body` per `{templates_folder}/tmpl-pr.md`
- Create PR:
  - Pass `pr_body` via stdin to `bash uspecs/u/scripts/uspecs.sh pr create --title "{pr_title}"`
  - Note: `pr_title` is passed on the command line; ensure it contains no shell-special characters (`<`, `>`, `$`, backticks)
  - If script exits with error: report the error and stop
  - Parse `pr_url` from script output
- Report `pr_url` and `change_branch_head` to Engineer; inform that Engineer is now on `pr_branch` to address review comments and that the deleted change branch can be restored with `git branch {change_branch} {change_branch_head}`
