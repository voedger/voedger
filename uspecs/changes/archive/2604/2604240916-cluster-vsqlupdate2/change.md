---
registered_at: 2026-04-22T08:59:37Z
change_id: 2604220859-cluster-vsqlupdate2
baseline: 36c8ee68f6ca5db6f2495bf5c36d23ff652c07d3
issue_url: https://untill.atlassian.net/browse/AIR-3656
archived_at: 2026-04-24T09:16:23Z
---

# Change request: Replace c.cluster.VSqlUpdate with q.cluster.VSqlUpdate2 to avoid command processor deadlock

## Why

`c.cluster.VSqlUpdate` hangs when the internal `c.sys.CUD` it issues is routed to the same command processor, because a command processor cannot dispatch another command to itself. See [issue.md](issue.md) for details.

## What

Introduce a new query-based flow that replaces the current command and eliminates the self-routing hang:

- New `q.cluster.VSqlUpdate2` query that calls `c.cluster.LogVSqlUpdate` and then performs the CUD against the target workspace, returning WLog offsets for both
- New `c.cluster.LogVSqlUpdate` command that only logs the original request parameters
- Router changes that reroute `c.cluster.VSqlUpdate` to `q.cluster.VSqlUpdate2` and adapt the response to the command response format

Migration steps:

- Ask frontend to switch to `q.cluster.VSqlUpdate2`
- Wait until Live uses `q.cluster.VSqlUpdate2`
- Remove `c.cluster.VSqlUpdate` and its special routing
