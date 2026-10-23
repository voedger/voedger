---
change_id: 2610221426-vit-test-restart-framework
type: test
domains: [devops]
scope: [dev]
---

# Change request: VIT restart framework for reusable configurations

## Why

Recovery integration tests currently require a new owned VIT configuration and two VVM starts for each scenario. Engineers need to restart a VIT backed by either an owned or shared configuration while retaining its storage and test entities, so compatible scenarios can reuse provisioned state and calculate expected sequences from their own pre-restart history.

## What

- VIT lifecycle integration tests cover restarts that preserve storage by default, without an opt-in flag, with both owned and shared configurations, including repeated restarts and later reuse of a shared configuration.
- Tests can modify stored checkpoints while the VVM is stopped and verify client-visible recovery against the history captured before restart.
- Restart coverage verifies that logins, workspaces, and documents are retained without being provisioned again, and that requests and cleanup use the current VVM instance.
- Existing owned-configuration restart tests retain their behavior; recovery scenario isolation, fault injection, and provisioning-progress synchronization remain explicit responsibilities of their fixtures.

## Technical design

- [x] create: [dev/vit-restart--td.md](../../specs/devops/dev/vit-restart--td.md)
  - Feature Technical Design for VIT restarts with owned and shared configurations: retained storage and entity identity, shared-cache publication of the current runtime, exclusive use, stopped-storage hooks, cleanup ownership, restart failures, compatibility with existing restart helpers, and pre-restart baselines for reusable recovery fixtures.

## Construction

Implement the synchronous server restart and compatibility contracts in the [technical design](../../specs/devops/dev/vit-restart--td.md). Tests using restart use their VIT from one goroutine with no overlapping operations; shared configurations for restart are dedicated to these tests and acquired sequentially. Restart is not used in stress tests and rejects benchmarks. Keep the recovery changes already in progress intact.

### Tests

- [x] create: [vit/restart_test.go](../../../pkg/vit/restart_test.go)
  - Focused lifecycle tests using the existing VIT application builders and isolated configuration identities, with real storage and client requests rather than a second implementation of the restart lifecycle.
  - Cover owned and shared restarts with `RestartVVM()` retaining storage by default, repeated direct restarts, and shared reacquisition after teardown: stable outer `*VIT`, replacement VVM, retained storage/WSIDs/data/principal references/counter, refreshed tokens, and requests reaching the current endpoint without provisioning configured entities again.
  - Cover replacement configuration without mutation or cache-key reassignment; stopped-storage hooks using only their storage argument; rejection of `RestartVVM` when `VIT.T` is a `*testing.B`; immediate client requests after successful synchronous restart; and distinct cleanup lifetimes for clients, subscriptions, readers, storage delays, email capture, and configuration callbacks.
  - Verify that configuration replacement preserves the active after-test callbacks and supplies new callbacks for the next acquisition. Resources created after restart close on teardown; after-test panic and `runtime.Goexit` still run remaining callbacks and release resources once.
  - Cover construction/launch errors and panics, and stopped-callback panic and `FailNow` paths, using subprocess tests where Go test termination requires isolation. Verify safe teardown, cleanup of partial runtimes, preservation of the original failure, and rejection of later acquisition of a failed shared identity.
  - Cover both existing restart adapters with shared configurations, including second-phase cleanup/time boundaries and exactly one additional launch only when a shared `AfterSecondStop` hook requires final inspection.

- [x] update: [it/impl_test.go](../../../pkg/sys/it/impl_test.go)
  - Reuse and, where needed, extend `TestVITResetPreservingStorage`, `TestVITResetPreservingStorageWithAnotherConfig`, and `TestWorkspaceInheritanceUpgrade` to verify owned-adapter compatibility without duplicating the new lifecycle test matrix.
  - Preserve client-visible data, processor configuration, schema upgrade, and absence-of-reprovisioning assertions while ensuring final cleanup targets the current VVM.

- [x] update: [it/impl_recovery_test.go](../../../pkg/sys/it/impl_recovery_test.go)
  - Migrate the two `TestODocArgumentRecordIDsAreRespectedAfterRestart` cases to one shared configuration identity declared outside their case loop. Prepare a fresh workspace with a unique name in each case so its ODoc-only history and exact record-ID assertions remain valid.
  - Capture log offsets before restart and explicitly await required provisioning actualizer flushes; each case must establish its own checkpoint state and run independently of the other case.
  - Retain the existing checkpoint-presence/removal, nested logged-argument ID, and client-response assertions. Keep the remaining recovery scenarios isolated until separately migrated; do not change literal feature examples as a side effect of this demonstration.
  - Preserve and run the owned descriptor-rejection scenario that uses `AfterSecondStop`, including its independently launched server's shutdown.

### Server restart implementation

- [x] update: [vit/types.go](../../../pkg/vit/types.go)
  - Add the public `RestartOptFunc` functional-option type with private option state. Retain the acquisition flag, server cleanup status, effective configuration, storage, entities, principal references and counter on the stable outer `VIT`; no stored failure, lifecycle state enum or transition locks are needed.
  - Keep VVM/configuration/client/email/problem-context fields directly on `VIT` and rebuild them in place after shutdown. Preserve public selectors and retained fixture fields without a runtime-holder type or whole-struct assignment of mutexes.
  - Store reader closing, notification unsubscribe, and storage-delay reset functions in one `VIT.currentTestCleanups` slice, consumed on restart and teardown. Include the current HTTP client closer at release time. Keep active `currentAfterTest` separate from `configAfterTest` for the next phase or acquisition; no resource-cleanup struct is needed.

- [x] update: [vit/consts.go](../../../pkg/vit/consts.go)
  - Retain stable `*VIT` values under the original configuration addresses, including failed identities. All `NewVIT` calls are sequential across configurations, so the shared-cache map needs no lock.
  - Preserve exclusive acquisition rather than adding parallel use of a single shared configuration.

- [x] update: [vit/impl.go](../../../pkg/vit/impl.go)
  - Separate retained fixture construction, server construction/launch, and acquisition initialization so first use captures the actual resolved storage factory and keyspace suffix and replacement launches skip entity provisioning.
  - Reuse the captured factory and suffix on every replacement, apply the effective configuration without mutating its options, register templates/runtime resources on each generation, and preserve established `WithInit`/`WithPostInit` behavior.
  - Update `NewVIT`, `NewVITLocalCassandra`, request entry points, token refresh, resource registration, and `TearDown` to use the current server, acquisition flag and server cleanup status. Keep direct restart time-neutral while acquisition and adapter phase transitions retain their existing mock-time behavior.
  - Run after-test callbacks only on teardown and preserve the current test's callbacks across configuration-changing restarts. Stop services before closing their email channel, report unexpected mail, and ensure failed or partially constructed runtimes are cleaned without masking the original error. Shared teardown releases the acquisition with the current runtime retained; owned teardown stops it.

- [x] update: [vit/impl_vitcfg.go](../../../pkg/vit/impl_vitcfg.go)
  - Rename `WithCleanup` to `WithAfterTest`, registering callbacks that run before resource release on teardown, independently of direct server restarts.

- [x] update: [vit/shared_cfgs.go](../../../pkg/vit/shared_cfgs.go)
  - Register shared fixture mock resets through `WithAfterTest`.

- [x] create: [vit/restart.go](../../../pkg/vit/restart.go)
  - Implement the public synchronous `(*VIT).RestartVVM(...RestartOptFunc) error` operation with `WithAfterStop` and `WithNewConfig` for either configuration ownership mode and the common restart orchestration used by the adapters, delegating to the single server constructor and cleanup operations extracted from `impl.go`.
  - Reject `VIT.T` being a `*testing.B` before stopping the server or invoking hooks. Keep the acquisition throughout restart, preserve fixture identity and storage, and return successfully only after launch and token refresh. Stopped hooks use only their storage argument.
  - Propagate phase-specific errors and panics to the current test, with cleanup on ordinary error, panic and `runtime.Goexit`. Reject requests, restarts and shared acquisition when the server is closed, without storing the original failure or silently creating fresh storage.
  - Run resource cleanup functions through `releaseResources()` on restart and teardown. Defer each release so a failure cannot skip the others; consume tracked resources before releasing them to avoid duplicate disposal.
  - Provide `cleanupOwnVIT`, `cleanupSharedVIT`, and `cleanupOnRestart` as explicit cleanup paths. Share config-callback execution, resource release, and server shutdown helpers. Only owned/shared teardown consumes config callbacks and ends the acquisition; restart preserves them. Keep email capture open until services stop and preserve cleanup of partial construction.

### Compatibility adapters

- [x] update: [vit/utils.go](../../../pkg/vit/utils.go)
  - Reimplement `TestRestartPreservingStorageWithHooks` and `TestRestartPreservingStorage` over the common lifecycle, removing owned-configuration eligibility guards and the separate two-instance storage-retention implementation.
  - Preserve signatures, stopped-storage access, optional second configuration, callback ordering, and existing test-phase cleanup/time boundaries. A shared `FirstRun` means preparation against current state, not initial provisioning.
  - Preserve `AfterSecondStop`: leave owned instances stopped, and resume shared instances without provisioning before releasing their acquisition. Avoid another shutdown of closed runtimes or an extra launch when final inspection is absent.
  - Keep one acquisition under the original config pointer and defer teardown only for final release. End config callbacks at phase boundaries and initialize the next phase after `RestartVVM`; reuse the existing effective configuration when a second one is omitted.
  - Remove the obsolete `restartVIT` constructor and its entity-provisioning flag; replacement VVM construction is handled by `RestartVVM` on the stable VIT.

## Quick start

Proposed API after construction: acquire a VIT with either an owned or shared configuration using `NewVIT`, and retain the returned pointer through repeated restarts. Shared reuse requires passing the same configuration address to later acquisitions and calling `TearDown` at the end of every acquisition.

With `vit` and `ws` already prepared by a fixture that has completed provisioning and awaited its required actualizer flushes:

```go
defer vit.TearDown()

body := `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.category","name":"recovery example"}}]}`
before := vit.PostWS(ws, "c.sys.CUD", body)

require.NoError(t, vit.RestartVVM())

after := vit.PostWS(ws, "c.sys.CUD", body)
require.Equal(t, before.CurrentWLogOffset+1, after.CurrentWLogOffset)
require.Equal(t, before.NewID()+1, after.NewID())
```

The fixture must exclude unrelated commands that would advance the same workspace's sequences. For checkpoint edits, supply `it.WithAfterStop(func(storage it.RestartStorage) { ... })`; its storage argument is valid while the old VVM is fully stopped and before the replacement starts. Use `it.WithNewConfig(&replacement)` to change the effective configuration while retaining storage. Both options can be passed together. Existing two-run test helpers remain available for callback-based scenarios.
