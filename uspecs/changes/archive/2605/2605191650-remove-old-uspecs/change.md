---
registered_at: 2026-05-19T16:34:26Z
change_id: 2605191634-remove-old-uspecs
type: chore
baseline: 4184d623f617f80924fec35b0351252c59686887
archived_at: 2026-05-19T16:50:14Z
---

# Change request: Remove old uspecs framework files

## Why

The uspecs framework is now delivered via the `uspecs-dev` plugin, which provides actions, skills, templates, and scripts. The legacy in-repo copy under `uspecs/u/` is obsolete, duplicates plugin content, and risks drift between the two sources.

## What

Remove the legacy in-repo uspecs framework while keeping the working data (changes, version):

- Delete the `uspecs/u/` directory (action docs, concepts, conf, examples, prompts, scripts, templates, uspecs.yml)
- Update `AGENTS.md` and `CLAUDE.md` to drop references to `uspecs/u/` files
- Preserve `uspecs/changes/` and `uspecs/version.txt`
- Fix the name for the prod domain file
