# AIR-3517: Processors: simplify IState reuse

- **Type:** Task
- **Status:** In Progress
- **Assignee:** d.gribanov@dev.untill.com
- **URL:** https://untill.atlassian.net/browse/AIR-3517

## Why

hostState reuse in the command processor looks overcomplicated: a huge number of closures are provided

## What

Use current workpiece instead of closures that read from workpiece - looks like the workpiece already has all necessary data to reuse
