# AIR-3603: voedger: send Retry-After header on func limit exceed

- **Key**: AIR-3603
- **Type**: Sub-task
- **Status**: In Progress
- **Assignee**: d.gribanov@dev.untill.com
- **URL**: https://untill.atlassian.net/browse/AIR-3603

## Why

If a limit is exceeded then just `429` status code is sent to the HTTP client. The client does not know how much time to wait.

## What

In processors: limit exceeded → send `Retry-After` header with amount of seconds to retry after.
