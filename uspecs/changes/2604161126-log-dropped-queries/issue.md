# AIR-3575: voedger: add log if query execs per workspace limit is reached

- **Type:** Sub-task
- **Status:** In Progress
- **Assignee:** d.gribanov@dev.untill.com

## Description

**Why:**

When the per-workspace query execution limit is reached, the router silently returns HTTP 503 without any server-side logging. This makes it difficult to diagnose query throttling issues in production — operators have no way to know which workspace hit the limit or how frequently it happens.

**What:**

Log amount of queries dropped during the last 10 seconds per pair app+wsid:

- Log level: Warning
- Stage attribute: `routing.qp.limit`
- Msg: `droppedInLast10Seconds=X`
- LogCtx: take from the last query
- Applied to both API v1 and v2 paths

Accumulate amounts of dropped queries:

- On each query, if limit is reached:
  - Bump counter of dropped queries for key [app, wsid, extension]
  - Store the LogCtx in the key
- 10 seconds passed from the first dropped unlogged query:
  - Log one message for each key [app, wsid, extension]
  - LogCtx is ctx from the key (the ctx of the last query)
  - Purge
- Log the entire contents of dropped queries on server shutdown like after 10 seconds since the first drop

Assume optimistic case when queries will be requested often enough.

Acceptable bad case:

- 1000 queries were dropped during 1 second
- No queries then
- Nothing in the log, we do not know about this DDoS-like case
