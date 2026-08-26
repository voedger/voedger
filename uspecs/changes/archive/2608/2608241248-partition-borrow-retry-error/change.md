---
change_id: 2608241226-partition-borrow-retry-error
type: fix
issue_url: https://untill.atlassian.net/browse/AIR-4768
domains: [prod]
scope: [apps]
---

# Change request: Correct partition borrow retry error handling

Refs:

- [AIR-4768: voedger, appparts: fix wrong error checked in partition borrow retry handler](./issue-AIR-4768.md)

## Why

`WaitForBorrow` must keep waiting while an extension engine is temporarily unavailable, but it must return every other borrow error. Its retry handler checked the named return variable `err` instead of the received operation error `opErr`. Because `err` was always `nil`, the handler classified every failure as retryable. Permanent errors such as an unknown application or partition could therefore be suppressed and retried until the caller's context was cancelled.

### Why existing tests passed

The existing test suite did not expose the incorrect retry classification because:

- integration tests exercised `WaitForBorrow` only with deployed applications and valid partitions; in that state, `Borrow` either succeeds or returns `ErrNotAvailableEngines`, which is supposed to be retried
- there was no focused coverage passing a non-engine-availability error through the retry policy
- processor-focused tests either mock `IAppPartitions` or call `Borrow` directly, so they do not exercise the real `WaitForBorrow` error-classification branch

## What

Symptom: A processor keeps waiting when partition borrowing fails with a non-retryable error.

```text
processor requests an application partition through WaitForBorrow
      |
      v
Borrow returns an application or partition lookup error
      |
      v
partBorrowRetryCfg.OnError receives opErr
      |
      v
OnError checks named return variable err (nil) instead of opErr   <-- fault
      |
      v
!errors.Is(nil, ErrNotAvailableEngines) is true
      |
      v
OnError requests another retry
      |
      v
WaitForBorrow retries a permanent error until context cancellation   (symptom)
```

Corrected behavior: `ErrNotAvailableEngines` remains retryable because it represents temporary engine-pool contention. Every other borrow error aborts the retry loop and is returned to the processor.

## How

Decisions:

- Keep borrow-error classification in the shared application-partition retry policy so command, query, actualizer, and scheduler processors receive consistent behavior without processor-specific handling
- Continue classifying engine exhaustion with `errors.Is` against the public sentinel so processor-specific wrapped errors retain their identity and diagnostics
- Verify the retry contract at the application-partition boundary across every processor kind rather than duplicating coverage in individual processor packages

Assumptions:

- None

Out of scope:

- Preventing scheduler or job re-entry
- Changing extension-engine pool sizes or partition borrow retry delays

References:

- [shared partition borrowing and retry policy](../../../../../pkg/appparts/impl.go)
- [processor-specific engine exhaustion errors](../../../../../pkg/appparts/errors.go)
- [engine borrow and release lifecycle](../../../../../pkg/appparts/impl_app.go)
- [existing application-partition borrowing coverage](../../../../../pkg/appparts/impl_test.go)

## Construction

- [x] create: [appparts/impl_internal_test.go](../../../../../pkg/appparts/impl_internal_test.go)
  - provide package-internal regression coverage for the application-partition borrow retry policy
  - verify that every processor-specific wrapped `ErrNotAvailableEngines` error enables retry without an abort error
  - verify that a non-engine-availability error disables retry and is returned as the abort error

- [x] update: [appparts/impl.go](../../../../../pkg/appparts/impl.go)
  - classify the operation error received by `partBorrowRetryCfg.OnError` instead of its named return variable
  - retry engine-exhaustion errors and abort with the operation error for all other borrow failures
