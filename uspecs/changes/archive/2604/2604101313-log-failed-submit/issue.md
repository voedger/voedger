# AIR-3541: VVM: log if failed to submit to processors

- **Type:** Sub-task
- **Status:** In Progress
- **Assignee:** d.gribanov@dev.untill.com

## Why

If VVM request handler failed to submit the request to processors via procbus then the client just receives 503 Service Unavailable and that's it. We do not see that in logs.

## What

Failed to submit → log an appropriate message.
