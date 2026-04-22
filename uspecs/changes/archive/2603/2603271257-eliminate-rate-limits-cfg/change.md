---
registered_at: 2026-03-27T11:28:08Z
change_id: 2603271128-eliminate-rate-limits-cfg
baseline: 4594ae92a387f53b38c12ff7db5fab388a41dce0
issue_url: https://untill.atlassian.net/browse/AIR-3417
archived_at: 2026-03-27T12:57:16Z
---

# Change request: Eliminate AppConfig rate limits infrastructure

## Why

The rate limits infrastructure in AppConfig is no longer needed and adds unnecessary complexity to the codebase. See [issue.md](issue.md) for details.

## What

Remove all AppConfig infrastructure related to rate limits:

- Rate limit configuration fields and types from AppConfig
- Rate limit initialization and setup logic
- Related helper functions and utilities
- Associated tests and references
