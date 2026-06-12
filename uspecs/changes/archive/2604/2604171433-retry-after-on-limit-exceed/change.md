---
registered_at: 2026-04-17T13:25:28Z
change_id: 2604171325-retry-after-on-limit-exceed
baseline: d5de0204088ff817cdc678a5c88e27b65a2c0eaa
issue_url: https://untill.atlassian.net/browse/AIR-3603
archived_at: 2026-04-17T14:33:15Z
---

# Change request: Send Retry-After header on function limit exceed

## Why

When a function limit is exceeded, processors currently return only the HTTP 429 status code, leaving clients without guidance on when to retry. See [issue.md](issue.md) for details.

## What

Update processors so that limit-exceeded responses include retry guidance:

- On limit exceeded in processors, send the `Retry-After` response header containing the number of seconds the client should wait before retrying
