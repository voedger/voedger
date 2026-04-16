---
registered_at: 2026-04-16T11:26:19Z
change_id: 2604161126-log-dropped-queries
baseline: 03a20266199ecd140225eba0de2d3321521c20d3
issue_url: https://untill.atlassian.net/browse/AIR-3575
---

# Change request: Log dropped queries per workspace limit

## Why

When the per-workspace query execution limit is reached, the router silently returns HTTP 503 without any server-side logging. Operators have no way to know which workspace hit the limit or how frequently it happens.

See [issue.md](issue.md) for details.

## What

Log the amount of queries dropped during the last 10 seconds per [app, wsid, extension] key:

- Log level: `Warning`, stage: `routing.qp.limit`, msg: `droppedInLast10Seconds=X`
- LogCtx: taken from the last dropped query for each key
- Applied to both API v1 and v2 paths

Accumulate dropped query counts with deferred logging:

- On each query rejection: bump counter for key [app, wsid, extension], store the LogCtx from the last query
- 10 seconds after the first unlogged drop: log one message per key, then purge
- On server shutdown: log all pending entries as if 10 seconds had elapsed

Accepted limitation:

- If queries are dropped only in a short burst with no further drops, the log entry may be lost (optimistic case assumes frequent enough query activity)
