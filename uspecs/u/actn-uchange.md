# Action: Create change request

## Overview

Create a new change request folder with a structured Change File.

Optionally:

- Fetch issue content from an issue URL (GitLab, GitHub, Jira)
- Create a git branch

## Instructions

Rules:

- Always read `uspecs/u/concepts.md` and `uspecs/u/conf.md` before proceeding and follow the definitions and rules defined there
- Never perform any implementation, code changes, or actions outside of this flow - the only output is the Change Folder, Change File, optional Issue File, git branch
- If change description is not provided, ask the user for it before proceeding. Do not treat the response as a new command - use it as the change description and continue this flow
- Never pass any optional parameter (`--no-branch`, `--branch`, `--issue-url`) to the script or alter default behavior unless it is explicitly requested

Parameters:

- Input
  - Change description
  - --no-branch option (optional): skip git branch creation
  - --branch option (optional): force git branch creation even when not on the default branch
  - Issue reference (optional): URL to a GitLab/GitHub/Jira/etc. issue that prompted the change
    - Referenced further as `{issue_url}`
- Output
  - Active Change Folder with Change File
  - Issue File (if issue reference provided)
  - Git branch (created by default when on the default branch; skipped when on a non-default branch unless --branch; skipped when --no-branch; git repository must exist)

Flow:

- Pre-flight checklist (verify before proceeding):
  - [ ] I have read `uspecs/u/concepts.md` and `uspecs/u/conf.md`
  - [ ] User has NOT explicitly requested `--no-branch` -> I will NOT pass this flag
  - [ ] User has NOT explicitly requested `--branch` -> I will NOT pass this flag  
  - [ ] User has NOT provided an issue URL -> I will NOT pass `--issue-url`
  - [ ] If any of the above are explicitly requested, I will pass ONLY those flags
- Determine `change_name` from the change description: kebab-case, max 40 chars (ideal 15-30), descriptive
- Run script to create Change Folder:
  - Base command: `bash uspecs/u/scripts/softeng.sh change new {change_name}`
  - If issue reference provided add `--issue-url "{issue_url}"` parameters (quoted to handle shell-special characters such as `&`)
  - If --no-branch option provided add `--no-branch` parameter
  - If --branch option provided add `--branch` parameter
  - Fail fast if script exits with error
  - Parse Change Folder path from script output, path is relative to project root
- Write Change File body (`{templates_folder}/tmpl-change.md`) appended to Change File created by the script
- If issue reference provided:
  - Try to fetch issue content
  - If fetch succeeds:
    - Convert it to rich markdown format suitable for Issue File
    - Save fetched content to Issue File (issue.md) inside the Change Folder
    - Add reference to Issue File in Why section: `See [issue.md](issue.md) for details.`
- Show user what was created
