# Action: Archive change

## Overview

Archive a completed change request folder.

## Instructions

Rules:

- Always read `uspecs/u/concepts.md` and `uspecs/u/conf.md` before proceeding and follow the definitions and rules defined there

Parameters:

- Input
  - Active Change Folder path (not required when `--all` is used)
  - `--all` option (optional): archive all change folders modified vs `pr_remote/default_branch`; no Active Change Folder path needed
- Output
  - Folder(s) moved to `{changes_folder}/archive`
  - If on PR branch and Engineer confirms: git commit and push with message, branch and refs removed, deleted branch hash and restore instructions reported

Flow:

- If `--all` option is provided:
  - Run `bash uspecs/u/scripts/softeng.sh change archiveall`
  - Analyze output, show to Engineer and stop
- Otherwise:
  - Identify Active Change Folder to archive, if unclear, ask Engineer to specify folder name
  - Run `bash uspecs/u/scripts/softeng.sh status ispr`
    - If output is `yes`: present Engineer with the following options:
        1. Archive + git cleanup (commit, push, delete local branch and remote tracking ref)
        2. Archive only (no git operations)
        3. Cancel
      - On option 1: `bash uspecs/u/scripts/softeng.sh change archive <change-folder-name> -d`; script output includes deleted branch hash and restore instructions - relay to Engineer
      - On option 2: `bash uspecs/u/scripts/softeng.sh change archive <change-folder-name>`
      - On option 3: abort, no action taken
    - Otherwise: `bash uspecs/u/scripts/softeng.sh change archive <change-folder-name>`
  - Analyze output, show to Engineer and stop
