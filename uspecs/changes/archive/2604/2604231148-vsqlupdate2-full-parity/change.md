---
registered_at: 2026-04-22T14:48:41Z
change_id: 2604221448-vsqlupdate2-full-parity
baseline: f91d1438f6fbf47ffd247bb8d06e47cfdff55e6e
issue_url: https://untill.atlassian.net/browse/AIR-3665
archived_at: 2026-04-23T11:48:36Z
---

# Change request: Make q.cluster.VSqlUpdate2 cover all c.cluster.VSqlUpdate features

## Why

The frontend must call `q.cluster.VSqlUpdate2` as the single entry point for every DML flavor that `c.cluster.VSqlUpdate` supports today, so the legacy command resource can be retired from the public API. See [issue.md](issue.md) for details.

## What

Make `q.cluster.VSqlUpdate2` a full functional replacement for `c.cluster.VSqlUpdate`:

- Accept every DML kind currently dispatched by `c.cluster.VSqlUpdate` (including kinds previously rejected by the query: `unlogged update`, `update corrupted`, and any others handled by `dispatchDML`)
- Inside the query, dispatch to `c.cluster.LogVSqlUpdate` (new companion command that only records the VSql audit log entry) and then execute the DML-specific side effects in the query processor

Reshape the router to the new contract:

- Transform every `c.cluster.LogVSqlUpdate` command response into the `q.cluster.VSqlUpdate2` query response shape (offsets, `NewID`, etc.) so existing API v1 / v2 callers of `c.cluster.VSqlUpdate` keep observing the legacy response payload
- Retire the current AIR-3656 / AIR-3661 shim that reroutes `c.cluster.VSqlUpdate` to `q.cluster.VSqlUpdate2` once the inverse transformation is in place
