---
registered_at: 2026-04-04T10:33:39Z
change_id: 2604041033-simplify-istate-reuse
baseline: 9f5dc79dd02b7738d79555554c885341a09c4aa1
issue_url: https://untill.atlassian.net/browse/AIR-3517
---

# Change request: Simplify IState reuse in processors

## Why

The hostState reuse mechanism in the command processor is overcomplicated, relying on a large number of closures. See [issue.md](issue.md) for details.

## What

Simplify IState reuse by using the current workpiece directly instead of closures that read from the workpiece:

- Remove excessive closure-based state passing in command processor
- Use workpiece fields directly since the workpiece already contains all necessary data for state reuse
