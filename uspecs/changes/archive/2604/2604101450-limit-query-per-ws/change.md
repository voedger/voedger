---
registered_at: 2026-04-10T13:24:52Z
change_id: 2604101324-limit-query-per-ws
baseline: 2d707b6e4fd9a3b4427356077ea26a1731da85ff
issue_url: https://untill.atlassian.net/browse/AIR-3542
archived_at: 2026-04-10T14:50:46Z
---

# Change request: Limit concurrent query executions per workspace

## Why

During a production outage (AIR-3536), 93 goroutines simultaneously processed webhook requests for a single workspace, starving all other workspaces and blocking query processors entirely. A per-workspace concurrency limit is needed to prevent this resource exhaustion.

See [issue.md](issue.md) for details.

## What

Implement a per-workspace concurrent query execution limiter in the router layer (`pkg/router`):

- Add a concurrent-safe per-WSID counter map in the router
- Before sending a query request, check the counter for the target WSID; return HTTP 503 if at or above the limit
- Increment the counter before processing and decrement after `reply_v1`/`reply_v2` returns (via defer)
- Apply the limit to query requests only (QP-bound APIPath in V2, resource prefix `q.` in V1)
- Make the limit configurable via `RouterMaxQueriesPerWS` in `VVMConfig` (default 10)
