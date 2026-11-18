---
change_id: 2611180850-workspace-creation-retry
type: fix
issue_url: https://untill.atlassian.net/browse/AIR-5285
domains: [prod]
scope: [apps]
---

# Change request: Workspace creation retry after federation exhaustion

Refs:

- [AIR-5285: Let the actualizer retry workspace creation after federation retry exhaustion](./issue-AIR-5285.md)

## Why

Workspace creation can encounter a temporarily unavailable destination during recovery. Recording federation retry exhaustion as an owner error prematurely fails creation instead of allowing the actualizer to retry after the destination recovers.

## What

Symptom: Workspace creation records an owner error after federation retries expire during temporary destination unavailability.

```text
Workspace creation is requested
        |
        v
ApplyInvokeCreateWorkspaceID or InvokeCreateWorkspace sends a creation command
        |
        v
Federation retries expire while the destination is temporarily unavailable
        |
        v
pkg/sys/workspace/impl.go error branches call updateOwner
        <-- fault: retry exhaustion is handled as a permanent creation failure
        |
        v
updateOwner writes a null WSID and WSError and returns nil when the update succeeds
        |
        v
The actualizer receives no error to trigger replay; creation is reported as failed
```

Corrected behavior: `ApplyInvokeCreateWorkspaceID` and `InvokeCreateWorkspace` detect `errors.Is(err, httpu.ErrRetryExhausted)` and return the error to the actualizer without updating the owner's `WSID` or `WSError`, allowing existing backoff and event replay to keep creation pending and complete it after recovery while preserving successful-response, conflict-response, and permanent-business-failure handling.

## How

Decisions:

- Keep error classification at the Workspace creation boundary, where the choice to update the owner is made, without changing federation retry windows or introducing another retry loop
- Retain command-level deduplication and the existing acceptance of conflict responses as the basis for safe replay, without adding a new idempotency protocol or persisted retry state
- Verify the internal creation branches with local Workspace package tests: error propagation, absence of owner updates, successful re-invocation after exhaustion, and unchanged business-failure handling

Assumptions:

- Existing actualizer replay and command deduplication remain sufficient for recovery after a committed command loses its response; the local tests do not verify this end-to-end behavior

Out of scope:

- Reclassifying federation failures that do not carry the retry-exhaustion marker
- Changing Workspace initialization or partition recovery policies
- Dedicated integration regression tests for actualizer-driven recovery and persisted deduplication

References:

- [Workspace creation, owner updates, and command deduplication](../../../../../pkg/sys/workspace/impl.go)
- [Federation retry and error classification contract](../../../../../pkg/coreutils/federation/interface.go)
- [Actualizer retry lifecycle](../../../../../pkg/processors/actualizers/async.go)
- [Workspace lifecycle integration coverage](../../../../../pkg/sys/it/impl_childworkspace_test.go)

## Construction

- [x] Create: [sys/workspace/impl_test.go](../../../../../pkg/sys/workspace/impl_test.go)
  - Add focused regression coverage for `ApplyInvokeCreateWorkspaceID` and the `InvokeCreateWorkspace` projector through `invokeCreateWorkspaceProjector`
  - Verify direct, wrapped, and joined retry-exhaustion errors propagate unchanged and cause no owner-update request
  - Invoke each creation path again with the same input after exhaustion and verify that it retries the command successfully without updating the owner
  - Verify successful responses remain successful and non-exhaustion failures retain owner-update behavior, including propagation of an owner-update failure
  - Reuse shared state and row test doubles from `pkg/coreutils/mock.go` and `pkg/coreutils/test_utils.go`; keep the federation mock local to avoid the `coreutils` import cycle
- [x] Update: [sys/workspace/impl.go](../../../../../pkg/sys/workspace/impl.go)
  - Check `errors.Is(err, httpu.ErrRetryExhausted)` in both creation error branches and return the original error before `updateOwner` can run
  - Preserve existing expected HTTP status codes and handling of all other results without changing federation or actualizer retry policy
  - Run the Workspace package tests and verify the exhaustion cases fail against the original error branches and pass with the fix
