---
registered_at: 2026-04-02T11:41:31Z
change_id: 2604021141-sequences-migrate-design
baseline: cde799cbe15807fbd753b5c4a00c53efc1b8d4af
issue_url: https://untill.atlassian.net/browse/AIR-3471
archived_at: 2026-04-02T13:47:03Z
---

# Change request: Sequences: migrate and actualize design

## Why

The current sequences design documentation needs to be migrated into the proper specs structure and actualized. The proposed design appeared too complicated and needs to be simplified before being recorded.

See [issue.md](issue.md) for details.

## What

Migrate and actualize sequences architecture documentation:

- Move actual design to `specs/prod/apps/sequences--arch.md`
- Move proposed (complicated) design to `specs/prod/apps/sequences--arch2.md`
- Separate active command processor behavior from ISequencer package (implemented, not yet integrated)
- Add deep code links, data structures, flows, diagrams, and synchronization primitives
- Remove `reqmd` tags from 13 Go source files
- Fix LRU terminology error in sequences--arch2.md
