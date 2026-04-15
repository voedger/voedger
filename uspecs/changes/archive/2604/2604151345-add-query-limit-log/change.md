---
registered_at: 2026-04-15T13:22:42Z
change_id: 2604151322-add-query-limit-log
baseline: 7469ac179a631a24474dafd5a5077121787d9b47
issue_url: https://untill.atlassian.net/browse/AIR-3575
archived_at: 2026-04-15T13:45:56Z
---

# Change request: Add log when query execs per workspace limit is reached

## Why

When the query executions per workspace limit is reached, there is currently no logging to indicate this condition. Adding a log message will improve observability and help diagnose issues related to query throttling.

See [issue.md](issue.md) for details.

## What

Add a warning log when the router rejects a query due to the per-workspace concurrent query limit being reached:

- Log level: `Warning`, stage: `routing.qp.limit`, empty message
- Move `withLogAttribs` before the limiter check so the log carries structured context (WSID, app, extension, request ID)
- Applied to both API v1 and v2 paths
