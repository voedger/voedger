# Action: Create change

## Overview

Create a new change request folder with a structured Change File. Optionally fetch issue content from an issue URL (GitLab, GitHub, Jira) and create a git branch.

## Instructions

Rules:

- Always read `uspecs/u/concepts.md` and `uspecs/u/conf.md` before proceeding and follow the definitions and rules defined there

Parameters:

- Input
  - Change description
  - --branch option (optional): create git branch for the change
  - Issue reference (optional): URL to a GitLab/GitHub/Jira/etc. issue that prompted the change
    - Referenced further as `{issue_url}`
- Output
  - Active Change Folder with Change File
  - Issue File (if issue reference provided)
  - Git branch (if --branch option provided and git repository exists)

Flow:

- Determine `change_name` from the change description: kebab-case, 15-30 chars, descriptive
- Run script to create Change Folder:
  - Base command: `bash uspecs/u/scripts/uspecs.sh change new {change_name}`
  - If issue reference provided add `--issue-url "{issue_url}"` parameters (quoted to handle shell-special characters such as `&`)
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
- Show user what was created and stop, do not proceed to implementation or other steps
