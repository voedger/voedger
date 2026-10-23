# Feature technical design: VIT restarts with reusable configurations

Proposed design for the [VIT restart framework change](../../../changes/2610221426-vit-test-restart-framework/change.md). This is a test-framework change in `pkg/vit`; production recovery and projector replay behavior remain unchanged.

## External actors

Systems:

- `*Integration test`
  - Acquires a VIT, prepares history, optionally edits storage while stopped, restarts the server, and asserts results through the current server.

## Components

### Layers

```text
Test entry points
    |
    +-- [VIT lifecycle]
    +-- [Restart test adapters]
    |
    v
Retained test state
    |
    +-- [(Shared VIT cache)]
    +-- [(Restart storage)]
    |
    v
Server execution
    |
    +-- [VVM runtime]
```

### Test entry points

- `[VIT lifecycle]`
  - A stable `*VIT` owns configuration identity, exclusive acquisition, test entities, counters, retained storage, and the current runtime. Add an explicit storage-preserving restart operation; owned/shared configuration controls acquisition and final teardown, rather than eligibility for restart.
  - impl: [pkg/vit/types.go](../../../../pkg/vit/types.go)
  - impl: [pkg/vit/impl.go](../../../../pkg/vit/impl.go)
  - impl: [pkg/vit/impl_vitcfg.go](../../../../pkg/vit/impl_vitcfg.go)

- `[Restart test adapters]`
  - Existing `TestRestartPreservingStorage` and `TestRestartPreservingStorageWithHooks` retain their signatures and hook ordering, and delegate storage retention and server replacement to the lifecycle. They accept both owned and shared acquisition configurations.
  - impl: [pkg/vit/utils.go](../../../../pkg/vit/utils.go)

### Retained test state

- `[(Shared VIT cache)]`
  - Retains the existing identity rule: the original `*VITConfig` address is the cache key. Its value is the stable `*VIT`, whose runtime may change. A second acquisition before `TearDown` fails; restart does not release that acquisition.
  - impl: [pkg/vit/consts.go](../../../../pkg/vit/consts.go)
  - impl: [pkg/vit/impl.go](../../../../pkg/vit/impl.go)

- `[(Restart storage)]`
  - The storage-factory instance actually used on first construction and its effective keyspace isolation suffix are retained from the initial launch. Each replacement VVM receives that same factory and suffix. Reuse the existing `RestartStorage` interface for stopped hooks; storage is neither cloned nor reinitialized.
  - impl: [pkg/vit/utils.go](../../../../pkg/vit/utils.go)
  - impl: [pkg/vvm/provide.go](../../../../pkg/vvm/provide.go)

### Server execution

- `[VVM runtime]`
  - Server fields directly on `VIT`: the VVM, resolved VVM configuration, problem context, HTTP client, email captor, BLOB readers, notification unsubscribe functions, and storage delay setters. Rebuild server fields in place after shutdown; the same VVM object is not relaunched and no separate runtime-holder type is needed.
  - impl: [pkg/vit/impl.go](../../../../pkg/vit/impl.go)
  - impl: [pkg/vvm/impl_orch.go](../../../../pkg/vvm/impl_orch.go)

## Scenarios

### Acquire and reuse a configuration

```text
*Integration test
  -> [VIT lifecycle]: NewVIT(tb, cfg)
      -> [(Shared VIT cache)]: reserve existing identity, or create when absent
      -> [(Restart storage)]: initialize once for this identity
      -> [VVM runtime]: construct, launch, and provision declared entities on first use
  -> [VIT lifecycle]: TearDown()
      -> [VIT lifecycle]: run after-test callbacks
      -> [VVM runtime]: release request resources
      -> [(Shared VIT cache)]: release shared acquisition; retain current running runtime
      -> [VVM runtime]: stop owned runtime when acquisition ends
```

The owned path does not use the shared cache. Shared reuse retains data and does not imply a clean database. `NewVIT` continues to advance mock time and refresh principal tokens when a test acquires an instance. Shared configurations used for restart are dedicated to restart tests and acquired sequentially, independently of unrelated shared application fixtures.

### Restart during an acquired test

```text
*Integration test
  -> [VIT lifecycle]: RestartVVM(options)
      -> [VVM runtime]: release request resources, fully shut down, then close email capture
      -> [(Restart storage)]: invoke optional AfterStop callback
      -> [VVM runtime]: construct and launch replacement over retained storage
      -> [VIT lifecycle]: refresh retained principal tokens; return ready for requests
  -> [VIT lifecycle]: continue requests using the same *VIT
```

Proposed public API:

```go
func (vit *VIT) RestartVVM(opts ...RestartOptFunc) error
func WithAfterStop(callback func(RestartStorage)) RestartOptFunc
func WithNewConfig(config *VITConfig) RestartOptFunc
```

`RestartVVM` requires an active acquisition and a running runtime. It supports repeated calls under the same acquisition, including `RestartVVM()` without options. It is synchronous: a successful return means the replacement VVM is running, retained principal tokens are refreshed, and client requests can use the current endpoint. Storage is preserved by default. `WithAfterStop` supplies the stopped-storage callback; `WithNewConfig` supplies a replacement configuration. Omitting `WithNewConfig` or supplying nil retains the current effective configuration. Options apply in argument order; later options for the same setting replace earlier ones.

Tests using restart use their VIT from one goroutine and make no overlapping calls. `RestartVVM` is not used in stress tests. If `VIT.T` is a `*testing.B`, the method returns an error before stopping the server or invoking any callback. No lifecycle state machine or per-VIT transition locking is needed under this contract.

The callback runs synchronously after all old VVM services have stopped and before any replacement services start. It receives only retained storage and must not call VIT operations. Old app-structure objects, engines, subscriptions, and borrowed partitions must not be used as live runtime dependencies there.

The method returns ordinary shutdown/construction/startup errors with their phase. Rejected usage before entering the restart transition leaves the current runtime untouched. Existing construction or startup panics retain their panic behavior, with cleanup guaranteed. A stopped callback may fail the Go test, including through `FailNow`; deferred resource handling must work on error, panic, and `runtime.Goexit`.

Each successful restart preserves the original configuration identity, storage, workspace/principal maps and object identities, mock-time source, and `NextNumber` counter. It creates no configured logins, subjects, workspaces, or documents again. Workspace templates and runtime resources are registered against every new VVM. Existing `WithInit` runtime-construction callbacks run on reconstruction; `WithPostInit` fixture provisioning runs only on initial creation. Tests must keep any custom initialization side effects suitable for repeated runtime construction.

An explicit replacement configuration changes the effective runtime configuration for this acquisition and subsequent reuse. It does not mutate either `VITConfig`, change owned/shared ownership, or create a second cache entry under the replacement configuration. Acquiring that replacement configuration independently retains its normal identity semantics. Its storage settings cannot redirect this restart away from the retained factory and suffix, matching the existing storage-preserving override behavior. Existing upgrade tests can continue changing schemas and processor configuration in their second run.

### Run existing restart-hook tests

```text
*Integration test
  -> [Restart test adapters]: cfg, hooks, optional second configuration
      -> [VIT lifecycle]: acquire; invoke FirstRun
      -> [VIT lifecycle]: restart; invoke AfterFirstStop while stopped
      -> [VIT lifecycle]: invoke SecondRun against replacement runtime
      -> [VIT lifecycle]: finish test phases and release acquisition
```

The adapters reuse existing shared state when the acquisition configuration is shared. Their `FirstRun` callback is a preparation phase, not a guarantee of empty storage or first-ever startup. The optional second configuration remains supported with either ownership mode.

Preserve the existing test-phase cleanup and mock-time boundaries of the two-run adapters. Their phase-transition operations and the public restart operation share one internal stop/build/install path; there must not be a separate shared-config restart implementation. A direct `RestartVVM` within a phase refreshes tokens without advancing mock time; adapter entry to its second test phase retains the existing day advancement.

Acquire once with the original configuration pointer and keep the same `*VIT` through both phases. Consume the first phase's config callbacks, call `RestartVVM` with `AfterFirstStop`, then initialize the second phase's callbacks and advance mock time. Do not call `TearDown` between phases or clone the cache-key configuration. One deferred `TearDown` releases the acquisition when the helper finishes. With no explicit second configuration, restart retains the current effective configuration, including on a reused shared VIT.

If `AfterSecondStop` is present, fully stop the second runtime before invoking it. Owned configurations finish stopped. For shared configurations, resume over the same retained storage without provisioning before releasing the acquisition, so the next test receives a running runtime. This explicit final inspection therefore costs one extra VVM launch for shared configurations. Do not introduce an implicit extra launch when this hook is absent. If inspection itself launches a server, it must shut that server down before returning, as the existing descriptor-rejection test does.

### Handle a failed restart

```text
*Integration test
  -> [VIT lifecycle]: RestartVVM(options)
      -> [VVM runtime]: stop old generation; replacement fails
      -> [VIT lifecycle]: close partial replacement resources; propagate error or panic
  -> [VIT lifecycle]: TearDown(); release acquisition without starting another server
  -> [(Shared VIT cache)]: retain failed identity; reject later normal acquisition
```

Keep the existing acquisition flag and server cleanup status. When construction or restart fails, deferred cleanup closes the current server and the original error or panic propagates to the current test. Do not store the failure on `VIT`. Reject subsequent requests, restarts and shared acquisition when the server is closed. Deferred cleanup also handles `runtime.Goexit`; `TearDown` remains safe and avoids shutting down an already closed server again.

A shared identity whose server is closed is not silently evicted and recreated: doing so would replace its database. Later acquisition reports that the server is closed; the original failure belongs to the test that attempted restart. Intentional startup-failure scenarios should use their own configuration identity. Recovery of a closed identity is outside this design.

### Prepare relative recovery assertions

```text
*Integration test
  -> [VIT lifecycle]: prepare scenario history in fresh or explicitly selected workspaces
      -> [VVM runtime]: finish required provisioning and persist projector progress
      -> [(Restart storage)]: capture pre-restart history and expected next values
  -> [VIT lifecycle]: restart with scenario-specific checkpoint edits
      -> [VVM runtime]: issue commands and compare responses with captured expectations
```

The framework retains history; each scenario establishes its own checkpoint and runtime preconditions. PLog offsets are scoped to a partition and WLog offsets and ordinary record IDs to a workspace. Capture the actual relevant log tails and allocation high-water mark before restart; never obtain expected values from the newly recovered generator. Record IDs can originate in logged arguments as well as CUDs, and synchronized events can introduce lower IDs, so neither the latest record nor the last event alone always establishes the allocation high-water mark.

Use fresh workspaces for histories requiring no previously allocated user IDs. Restore deliberately deleted log entries before reusing affected state, or isolate that scenario in its own configuration. Fault injection and log assertions must be scoped and reset by the fixture. Current literal Gherkin examples remain exact unless the specification and matching step comments are deliberately revised together. Migrating the whole recovery file is separate from implementing the framework; a small compatible group demonstrates shared reuse first.

## Cross-cutting concerns

### Runtime ownership and cleanup

- Every runtime generation is fully shut down before the next generation uses its storage or leadership identity.
- Every cached shared identity keeps the same outer `*VIT` across restarts, so deferred `vit.TearDown()` acts on its current runtime.
- Every restart rebuilds only server fields on the existing `VIT`; it never copies a live `sync.Mutex`, the whole `VIT`, or entity maps by whole-struct assignment.
- Every tracked resource is released once before its generation is discarded or its test ends; after-test callbacks run only when the test phase ends.

Store resource cleanup functions in a single `VIT.currentTestCleanups` slice. Register reader closing, notification unsubscribe, and storage-delay reset callbacks there. `releaseResources()` consumes the slice, includes the current HTTP client's idle-connection closer, and defers every callback in registration order so panic or `runtime.Goexit` cannot skip remaining releases. Both restart and teardown call it. The HTTP client remains reusable across shared acquisitions without a separate stored cleanup function.

`TearDown()` selects `cleanupOwnVIT()` or `cleanupSharedVIT()` and ends the acquisition, including when a callback fails. Both paths consume the active config-cleanup snapshot through `runConfigCleanups()` before releasing runtime resources. Owned cleanup also stops the server; shared cleanup validates the retained server and email capture. `RestartVVM()` calls `cleanupOnRestart()`, which releases runtime resources and stops the server while preserving config cleanups and the acquisition. All server shutdown paths share `shutdownVVM()`; it calls VVM shutdown when `vvmProblemCtx` is available. Failure cleanup reports shutdown errors without replacing the original failure.

`WithAfterTest` registers callbacks such as resetting test mocks. `currentAfterTest` holds the current phase's callbacks; `configAfterTest` holds the effective configuration's definitions for the next phase or acquisition. A configuration-changing direct restart preserves the active callbacks and updates only the definitions. `TearDown()` consumes the active callbacks in registration order before releasing resources, including when a callback panics or calls `FailNow`. The adapters also consume callbacks at their explicit phase boundaries. Direct restarts do not run these callbacks. Owned teardown also stops the current runtime; shared teardown releases the acquisition while retaining the runtime. Both ownership modes use the same callback and resource handling.

Keep server fields directly on the stable `VIT`, including its embedded `VoedgerVM` and `VVMConfig`. Keep test entities, counter, acquisition flag, effective configuration and retained storage across restarts. The original configuration address remains the cache key. Since calls are exclusive and synchronous, construction updates the current server fields in place after the previous server has fully stopped. Install deferred cleanup before construction so it can close the current partially initialized server on error, panic or `runtime.Goexit`. Constructors and adapters use the same server constructor and cleanup functions.

Email capture follows the runtime generation. Keep its channel open while old services can still send; stop services before closing that channel, check for unexpected mail, and install a fresh captor for the replacement. Retain existing retry policies, dynamic ports, schema-cache behavior, and goroutine/problem-context checks, but apply them to the appropriate current generation. Refresh the existing principal objects so previously retained workspace-owner references use current tokens.

### Concurrency

- Every test using restart uses its VIT from one goroutine, with no overlapping acquisition, requests, restart or teardown.
- Every `NewVIT` call is sequential across the process, including acquisitions of different configurations; the shared-cache map has no lock.
- Every stopped callback uses only its supplied storage access; VIT operations, including nested restart and acquisition, are unsupported there.
- Every retained raw runtime/service pointer and notification channel is invalid after restart.

Keep existing request/counter synchronization needed by stress tests that do not restart. Restart and teardown do not need per-VIT transition locks or an explicit state enum. Shared state reuse must not require a preceding scenario to have run.

### Alternatives and scope

- Every restart entry point uses the same retained-storage and generation-ownership implementation.
- Every owned/shared distinction affects acquisition and final release only; it does not select a different restart algorithm.

Removing only the owned-config assertions leaves `TearDown` unable to stop a shared runtime and the cache holding an obsolete server. Replacing the outer `*VIT` and updating the cache still leaves saved references and deferred cleanup bound to the old object. Retaining a stable facade and replacing one runtime generation resolves both problems. Relaunching the existing VVM object would reuse canceled contexts and completed shutdown resources, so reconstruction follows the existing restart strategy.

Storage snapshots, automatic rollback of test mutations, automatic waiting for all projectors, and production changes to make provisioning replay idempotent are outside this change. A fixture that requires completed provisioning must explicitly await persisted actualizer progress; restart must also remain usable by tests deliberately exercising unfinished work.

### Validation

- Every restart test asserts retained state through client requests and independently captured pre-restart expectations; internal assertions supplement those results for lifecycle ownership.
- Every shared-reuse test exercises reacquisition after teardown, rather than checking only the immediate post-restart callback.
- Every failure-path test verifies that no stopped or partially initialized runtime can be reused as healthy.

Cover these distinct cases with focused VIT lifecycle tests and existing integration fixtures:

| Case | Required evidence |
| --- | --- |
| Owned restart | Existing workspace, data, and configured processor settings survive; no entity provisioning repeats. |
| Shared restart and reacquisition | Same `*VIT`, new VVM generation, retained data, usable HTTP endpoint and tokens after later `NewVIT` with the original configuration. |
| Repeated restart | Two direct restarts retain counters, entities, storage identity, and current cleanup ownership. |
| Changed effective configuration | Processor/schema changes take effect without mutating the configuration objects or changing cache ownership. |
| Stopped hook | Old services are stopped and checkpoint edits affect the next launch; the callback uses only its storage argument. |
| Cleanup lifetimes | Old subscriptions/readers/client resources close once; resources created after restart close on teardown. `WithAfterTest` callbacks run at their phase boundary, including after configuration replacement, and callback failures cannot skip remaining callbacks or resource release. Final teardown targets the last VVM. |
| Exclusive acquisition | Acquiring the same shared configuration before `TearDown` fails. |
| Benchmark rejection | `VIT.T` being a `*testing.B` makes `RestartVVM` return an error before stopping or changing the server. |
| Synchronous restart | Client requests work immediately after `RestartVVM` returns successfully. |
| Failed restart | Construction error, launch failure/panic, and a stopped-callback failure leave the server closed and unavailable for reuse. Use an isolated subprocess for `FailNow`/panic paths when needed. |
| Final stopped hook | Existing owned descriptor-rejection coverage passes; a shared adapter resumes once after inspection and supports later acquisition. |
| Compatible recovery cases | Reused storage supports captured PLog/WLog/ID baselines, with fresh workspaces where required; each case also runs alone. |

Retain the existing `TestVITResetPreservingStorage`, `TestVITResetPreservingStorageWithAnotherConfig`, `TestWorkspaceInheritanceUpgrade`, and focused recovery-scenario checks. Use `-count` and shuffled case order for shared reuse and race-enabled checks for shared cache and existing request helpers when supported. Run only the affected packages/test functions; this design introduces no dependencies or required provisioning changes.
