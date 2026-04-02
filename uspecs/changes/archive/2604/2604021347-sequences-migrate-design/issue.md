# AIR-3471: Sequences: migrate and actualize design

**Type:** Sub-task
**Status:** In Progress
**Assignee:** d.gribanov@dev.untill.com
**URL:** https://untill.atlassian.net/browse/AIR-3471

## Description

### Background

The sequences feature has an actual (current) design and a proposed (more complicated) design that was considered but deemed too complex.

### Problem

The proposed design seems to be very complicated. The question is whether it can be simplified.

### What

Migrate existing documentation into the specs structure:

- Actual design → `specs/prod/apps/sequences--arch.md`
- Proposed (complicated) design → `specs/prod/apps/sequences--arch2.md`
- Draw a diagram that will show how goroutines acts between each others, how wait groups, channels and contexts works.
- Mark the place where single requested seqID is read for the workspace and stored in LRU cache as a technical dept: should always read numbres per workspace and keep in memeory without cache. Note: use industry idiomatic way to mark that as a technical dept
