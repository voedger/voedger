# Action: Sync Implementation Plan with actual modifications

Always read `uspecs/u/concepts.md` and `uspecs/u/conf.md` before proceeding any instructions.

## Flow

- Read baseline commit hash from Change File frontmatter
- Get diff since baseline using command: `git diff <baseline>`
- Update Implementation Plan with what was actually done, including changes in files and file contents

## Rules

- Track only files related to the current Change Request
- Exclude files in the Change Request folder from analysis
- Add items and sub-items for changes not yet reflected in the Implementation Plan
- Update or remove wrong or outdated items and sub-items
- Do not add more than 5 new sub-items per to-do item in a single sync
- Never remove correct items and sub-items to reduce their count
- Avoid adding unnecessary details, focus on high-level changes
  - E.g. if scenario was added do not mention each step added
