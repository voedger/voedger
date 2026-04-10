---
registered_at: 2026-04-10T12:00:09Z
change_id: 2604101200-log-failed-submit
baseline: 909db58170f1b8e223a534062ef0d68041d369ea
issue_url: https://untill.atlassian.net/browse/AIR-3541
archived_at: 2026-04-10T13:13:54Z
---

# Change request: Log failed submit to processors

## Why

When VVM request handler fails to submit a request to processors via procbus, the client receives a 503 Service Unavailable response but no information is logged on the server side, making it difficult to diagnose issues.

See [issue.md](issue.md) for details.

## What

Log an appropriate message when a request fails to be submitted to processors via procbus.
