# voedger, workspaces: let the actualizer retry workspace creation after federation retry exhaustion

- URL: https://untill.atlassian.net/browse/AIR-5285
- ID: AIR-5285
- State: In Progress⚒️
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov
- Linked issues: AIR-4959 — voedger: partition recovery performance

## Description

**Why**

Workspace creation currently records an owner error when federation retries expire, even if the destination is temporarily unavailable during recovery. This prematurely fails creation instead of allowing the actualizer to retry later.

**What**

* Use `errors.Is(err, httpu.ErrRetryExhausted)` in `ApplyInvokeCreateWorkspaceID` and `InvokeCreateWorkspace`
* Return retry-exhaustion errors to the actualizer without updating the owner’s `WSID` or `WSError`
* Use the actualizer’s existing backoff and event replay to retry creation later
* Preserve existing handling of successful responses and permanent business failures
* Add regression tests proving that creation remains pending after retry exhaustion and completes after recovery
* Verify that replay after a committed command with a lost response does not create a duplicate workspace
