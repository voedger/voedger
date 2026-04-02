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
- Draw a diagram that shows how goroutines interact with each other, and how wait groups, channels, and contexts work.
- Mark the place where a single requested `seqID` is read for a workspace and stored in an LRU cache as technical debt: the implementation should always read numbers per workspace and keep them in memory without using a cache. Note: use a standard, industry-idiomatic way to mark this as technical debt.
