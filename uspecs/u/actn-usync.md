# Action: Sync Active Change Folder with actual modifications

## Overview

Sync all specs and Active Change Folder files with actual source modifications since the baseline commit.

Sources (everything outside `uspecs/`) are the source of truth. Sync direction:

sources -> technical specs -> functional specs -> Active Change Folder files

## Instructions

Rules:

- Always read `uspecs/u/concepts.md` and `uspecs/u/conf.md` before proceeding
- Exclude files inside `uspecs/` from source analysis
- Focus on high-level changes; avoid unnecessary detail
  - E.g. if a scenario was added, do not list each individual step
- Do not add more than 5 new sub-items per to-do item in a single sync
- Never remove correct items and sub-items to reduce their count

Parameters:

- Input
  - Change File frontmatter (baseline commit hash)
  - Git diff of sources since baseline
- Output
  - Updated technical spec files
  - Updated functional spec files
  - Updated Active Change Folder files (all of them)

Flow:

- Read baseline commit hash from Change File frontmatter
- Get diff of sources since baseline: `git diff <baseline> -- . ':(exclude)uspecs/'`
- Step 1 - Sync technical specs
  - Identify Technical Design Specification files related to changed sources
  - Update them to reflect what was actually built (architecture, design decisions, component interactions)
- Step 2 - Sync functional specs
  - Identify Functional Design Specification files related to changed sources
  - Update them to reflect what was actually built (scenarios, requirements, domain model)
- Step 3 - Sync Active Change Folder files
  - Using sources, updated technical specs, and updated functional specs as input
  - Update all files present in the Active Change Folder (change.md, impl.md, decs.md, how.md, issue.md and others - whichever exist)
  - Implementation Plan: add, update, or remove items and sub-items to match what was actually done
