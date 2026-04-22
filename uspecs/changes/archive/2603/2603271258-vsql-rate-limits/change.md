---
registered_at: 2026-03-26T12:20:26Z
change_id: 2603261220-vsql-rate-limits
baseline: 43fecfd41a6c0244a700f0579fd4370e522e60fa
issue_url: https://untill.atlassian.net/browse/AIR-3410
archived_at: 2026-03-27T12:58:38Z
---

# Change request: Implement rate limits using VSQL

## Why

The application currently uses appconfig to apply rate limits. This approach needs to be improved by declaring limits using VSQL for better maintainability and consistency.

See [issue.md](issue.md) for details.

## What

Replace the existing appconfig-based rate limiting with VSQL-based implementation:

- Implement rate limits declaration using VSQL
- Replace the existing appconfig method
