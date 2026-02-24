# Action: Archive change

## Overview

Archive a completed change request folder.

## Instructions

Rules:

- Always read `uspecs/u/concepts.md` and `uspecs/u/conf.md` before proceeding and follow the definitions and rules defined there

Parameters:

- Input
  - Active Change Folder path
- Output
  - Folder moved to `{changes_folder}/archive`
  - If on PR branch and Engineer confirms: git commit and push with message, branch and refs removed

Flow:

- Identify Active Change Folder to archive, if unclear, ask Engineer to specify folder name
- Run `bash uspecs/u/scripts/uspecs.sh status ispr`
  - If output is `yes`: present Engineer with the following options:
      1. Archive + git cleanup (commit, push, delete local branch and remote tracking ref)
      2. Archive only (no git operations)
      3. Cancel
    - On option 1: `bash uspecs/u/scripts/uspecs.sh change archive <change-folder-name> -d`
    - On option 2: `bash uspecs/u/scripts/uspecs.sh change archive <change-folder-name>`
    - On option 3: abort, no action taken
  - Otherwise: `bash uspecs/u/scripts/uspecs.sh change archive <change-folder-name>`
- Analyze output, show to Engineer and stop
