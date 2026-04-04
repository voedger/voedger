# AIR-3517: Processors: simplify IState reuse

- **Type:** Task
- **Status:** In Progress
- **Assignee:** d.gribanov@dev.untill.com
- **URL:** https://untill.atlassian.net/browse/AIR-3517

## Why

hostState in command processor reuse looks overcomplicated: huge amount of closures is provided etc

## What

Use current workpiece instead of closures that read from workpiece - looks like the workpiece already has all necessary data to reuse
