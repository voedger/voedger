---
registered_at: 2026-02-17T09:21:22Z
change_id: 2602170921-bus-timeout-race-condition
baseline: 088667a70d75f6a14fc93b8a8b475c2a0d0723d2
archived_at: 2026-02-17T12:45:12Z
---

# Change request: Replace bus send timeout with infinite wait and periodic warnings

## Why

The current 10-second bus timeout creates a race condition where the router returns HTTP 503 to the client while VVM continues processing, potentially past the "point of no return" (PLog write). Clients following standard retry patterns on 503 resend the request, causing duplicate records or data corruption.

## What

Replace the fixed bus send timeout with an indefinite wait that logs periodic warnings:

- Remove `ErrSendTimeoutExpired` error path from `SendRequest()` in `pkg/bus/impl.go`
- Continue waiting for the actual VVM response instead of timing out
- Log a WARNING after each 1 minute of waiting, including the waiting duration

Update timeout-related constants and configuration:

- Update `pkg/bus/consts.go` to remove or adjust timeout constants
- Update `pkg/router/consts.go` RouterWriteTimeout configuration to accommodate longer processing
- Update `pkg/vvm/impl_cfg.go` VVM configuration defaults as needed
