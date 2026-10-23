---
change_id: 2609291131-improve-partition-workspace-recovery
type: feat
issue_url: https://untill.atlassian.net/browse/AIR-5009
domains: [prod]
scope: [apps, storage]
---

# Change request: Improved partition and workspace recovery

Refs:

- [AIR-5009: voedger: imrpove partition and workspace recovery](./issue-AIR-5009.md)

## Why

Partition and workspace sequence state must survive a VVM restart without rebuilding every counter from the complete log on demand. Existing deployments can already contain PLog and WLog history while having no recovery checkpoints, so VVM startup must backfill the initial checkpoint set before normal request-driven recovery begins.

## What

In the production application-processing context:

- Store only the last handled PLog offset in the partition checkpoint.
- Store one WLog ID-bearing offset: the offset of the most recent ID-bearing event found by initial recovery or observed by the checkpoint projector. Do not persist the last WLog offset or record IDs themselves.
- During VVM bootstrap, check every declared application partition. Scan missing-checkpoint PLogs concurrently under a per-application CPU bound of `runtime.GOMAXPROCS(0)`, compute workspace checkpoints for every WSID encountered, persist those workspace checkpoints, and persist the partition checkpoint last.
- Deploy only partitions whose initial recovery succeeded or whose completion checkpoint was already present. A non-cluster application's partition-scoped failure does not prevent server readiness, even when every partition attempt for that application fails; the affected partition remains undeployed, with no asynchronous projectors or scheduled jobs running. The cluster application's sole partition is required: initialization or deployment failure stops bootstrap.
- Keep the deployment descriptor's total `NumParts` unchanged in `c.cluster.DeployApp`, app registration, and routing. Failed checkpoint initialization changes only the runtime deployment-ID list; it must not reduce the declared count or remap workspaces to healthy partitions.
- Return `400 Bad Request` with `application not found` for a missing application, and `500 Internal Server Error` with `partition not found` for a missing or undeployed partition of a registered application. Keep application QName and partition ID in structured command-error log fields rather than repeating them in these messages. Return initial-recovery failures to bootstrap as a partition-ID-to-error map for logging; do not propagate their causes to app partitions or command requests. Requests do not retry initial recovery or deploy the partition; a later bootstrap retries partitions with absent completion markers.
- Represent a successfully initialized empty PLog with a present partition checkpoint whose `lastPLogOffset` is zero. Represent an encountered workspace with no ID-bearing event with a present workspace checkpoint whose `wlogIDBearingOffset` is zero.
- Recover a partition by reading the PLog from its saved handled offset through the end, using the final event to set the next PLog offset, and reapplying that final event.
- After partition recovery, let the requesting command continue to the workspace stage, which starts workspace recovery on demand.
- Recover a workspace with an inclusive WLog tail scan from the WLog ID-bearing offset through the log end. Use the actual final event to set the next WLog offset and the latest ID-bearing event encountered in the complete scan to restore record-ID allocation.
- Treat a stored checkpoint as invalid when its inclusive log tail scan invokes no callback, its first callback has a different offset, or a workspace checkpoint event did not allocate a non-singleton record ID. Log an error and recover from `FirstOffset` instead of accepting counters derived from an empty or incomplete scan.
- If a PLog is empty, all WLogs in that partition are also empty; exclude fixtures that empty only the PLog while retaining WLog events. Start empty-log offsets at `FirstOffset` and an empty WLog's next record ID at `FirstUserRecordID`.
- Return `503 Service Unavailable` while recovery of the requested partition or workspace is already running or if the partition already has four workspace recoveries running. Log a corresponding message for each case.
- Emit partition recovery lifecycle logs with `vapp=sys/voedger`, `extension=sys._Recovery`, and `partid`; emit workspace recovery lifecycle logs with those attributes plus `wsid`.
- Use the command processor's fixed workspace-recovery limit of four per partition.

## How

### Key concepts

- **Recovery checkpoint** (or **checkpoint**): a persisted set of log offsets used as inclusive recovery-scan starts. A checkpoint accelerates recovery but is not an authoritative copy of recovered sequence state.
- **Partition checkpoint**: `lastPLogOffset`, the latest PLog offset selected by initial recovery or subsequently stored by the checkpoint projector for a partition. A present zero value is an initialization sentinel marking an initialized empty PLog; it is not an event offset to validate.
- **Workspace checkpoint**: `wlogIDBearingOffset`, the offset of the latest ID-bearing WLog event found by initial recovery or subsequently seen by the checkpoint projector for a workspace. A present zero value is an initialization sentinel marking an encountered workspace for which no ID-bearing event was found; it is not an event offset to validate.
- **ID-bearing event**: a WLog event that allocates at least one new non-singleton record ID.
- **Initial recovery**: bootstrap-time reconstruction of missing recovery checkpoints from complete PLogs before an application's partitions and asynchronous projectors are started. It initializes checkpoint storage only; it does not restore or publish runtime partition or workspace sequence state.
- **Known partition**: every partition ID in the application deployment descriptor's contiguous declared range `0..NumPartitions-1`, whether or not that partition has a PLog event or allocated storage yet.
- **Initial-recovery worker limit**: `runtime.GOMAXPROCS(0)`, sampled once when an application's initial recovery starts. It is the Go runtime's effective available-processor limit and bounds the number of that application's missing partitions whose checkpoints may be computed concurrently.
- **Initial-recovery completion marker**: the presence of a partition checkpoint. Initial recovery writes all workspace checkpoints derived from that partition before writing its partition checkpoint, so an absent partition checkpoint means that the partition must be scanned again.
- **Undeployed partition**: a declared application partition for that is not deployed. Initial-recovery failure prevents the partition deployment for the current server run; app registration and the declared partition count remain available for request routing and error reporting.
- **Inclusive log tail scan**: a log read that starts with the event identified by a nonzero checkpoint and continues through the actual log end. The first returned event validates the checkpoint; the complete scan supplies the recovered values. An absent or zero checkpoint instead selects a direct full scan from `FirstOffset`.
- **Checkpoint invalid**: a nonzero checkpoint exists but any of:
  - `Read*Log(storedOffset, ReadToTheEnd)` produces no callback (e.g. beyond the end)
  - offset of the event on the first callback of `Read*Log(storedOffset, ReadToTheEnd)` is not equal to `storedOffset` (e.g. points to an unoccupied offset in the middle of the log)
  - for `Workspace checkpoint`: event on the first callback of `Read*Log(storedOffset, ReadToTheEnd)` has no new IDs (e.g. `storedOffset` incorrectly recorded)
- **Checkpoint fallback**: one inclusive scan from `FirstOffset` when a stored checkpoint is invalid. Recovery does not rewrite the checkpoint.
- **Partition recovery**: Command-processor-scoped recovery that restores the next PLog offset and republishes the application partition.
- **Workspace recovery**: Command-prcoessor-scoped recovery that restores one workspace's next WLog offset and record ID after its partition has recovered.

### Decisions

- Store only `lastPLogOffset` per partition and `wlogIDBearingOffset` per workspace. Checkpoints are inclusive scan starts, not authoritative sequence state; do not store the last WLog offset or record-ID high-water marks.
- `sys.ApplyRecoveryCheckpoint` advances the partition checkpoint every 100 PLog events and replaces the workspace checkpoint on each ID-bearing event.
- Bootstrap orchestrates initial recovery through `checkpoints.IInitialRecovery`; `pkg/checkpoints` owns event scanning and checkpoint calculation.
- A present partition checkpoint skips initial recovery. For each missing checkpoint in `0..NumPartitions-1`, scan the complete PLog and retain its end plus the latest ID-bearing WLog offset, or zero, for every encountered WSID.
- Run partition scans concurrently with a per-application limit sampled from `runtime.GOMAXPROCS(0)`. Persist all computed workspace checkpoints before the partition checkpoint.
- A partition-scoped checkpoint-read, scan, or storage failure is returned in the per-partition error map and excludes that partition from the deployable IDs. Bootstrap logs each returned error once with app QName and partition ID. Such failures do not cancel sibling scans; for non-cluster applications they do not fail bootstrap or degrade readiness. A failed reconstruction writes no completion marker; a checkpoint-read error never authorizes deployment. Setup failure or cancellation is fatal to an initial-recovery invocation; failure to initialize or deploy the cluster application's sole partition is also fatal to bootstrap.
- Initial recovery returns eligible partition IDs and a map of failed partition IDs to operation-specific errors. Bootstrap passes the IDs directly to `DeployAppPartitions` and logs the map entries; it starts no asynchronous actualizers or schedulers for excluded partitions. `IAppPartitions` uses its existing unknown-partition error without storing initial-recovery failures.
- Initial recovery initializes checkpoint storage only. Runtime partition and workspace states remain request-recovered.
- The descriptor's partition count is routing metadata, not an availability count. `c.cluster.DeployApp` receives that full count and preserves compatibility checks against the persisted descriptor. Runtime deployment uses successful IDs without changing the count or renumbering partitions.
- An absent or zero checkpoint starts one scan at `FirstOffset` without validation or invalid-checkpoint logging. A nonzero checkpoint must identify the first returned event, and a workspace checkpoint must identify an ID-bearing event; otherwise recovery logs the error and performs one fallback scan from `FirstOffset`.
- Partition recovery restores the next PLog offset from the actual PLog end and reapplies the final event. A later request starts workspace recovery, which restores the next WLog offset and next non-singleton record ID from the actual WLog end.
- On a deployed partition, runtime recovery admission remains `503` while work runs. A retained runtime failure is returned as `500` while one retry starts; an in-progress runtime retry returns `503`. An undeployed partition instead returns `500` on every command without entering that lifecycle.
- Workspace recovery is bounded by the fixed `numWSRecoverers = 4` limit per partition key, deduplicated by WSID, and tied to the service context. Before a recovered partition is published, its workspace manager shares both the partition key's stable recovery-slot pool and the partition manager's worker wait group, so reset cannot create fresh capacity while detached workers run and shutdown joins those workers.
- Recovery logs identify the application, partition, workspace when applicable, failed operation, and underlying error.
- `sequences.feature` covers successful user-visible issuance; `sequences--td.md` keeps issuance internals and failure-induced gaps. All checkpoint and recovery behavior belongs in `recovery.feature` and `recovery--td.md`.

Notes:

- Workspace chekpoints existance for each WSID is not checked on Intial recovery
- The last event in the plog is not applied on the Inital Recovery. It is done on the first command to the partition
- Query does not initiate partition or workspace recovery. Only a command does.

### Initial-recovery outcomes and deployment gate

`IInitialRecovery.InitializeCheckpoints(ctx, appQName, numPartitions)` returns `([]istructs.PartitionID, map[istructs.PartitionID]error, error)`: IDs eligible for deployment, operation-specific errors keyed by failed partition ID, and an optional fatal setup/context error. A present checkpoint, including zero, is eligible without another scan. A missing checkpoint becomes eligible only after the scan and every checkpoint write succeed. Each failed partition is omitted from the ID slice and appears once in the error map with its failed operation and wrapped underlying cause. When the fatal error is nil, the slice and map are disjoint and together cover every declared partition. An empty ID slice with a nonempty error map and a nil fatal error is a valid initialization result. For non-cluster applications it is not a bootstrap error; the cluster application's sole partition is required. A setup or cancellation error prevents deployment from that invocation, even if some attempts already completed; any collected partition errors can still be logged, while context cancellation itself is reported through the fatal error.

Scans of distinct missing partitions run concurrently, with at most the `runtime.GOMAXPROCS(0)` value sampled once for that application's call in flight. Each partition reads its own PLog sequentially through the actual log end. All started attempts are joined before returning; a partition failure does not cancel healthy siblings. Applications remain initialized in bootstrap order.

Bootstrap registers the app's definition and full declared partition count, invokes `InitializeCheckpoints`, logs each returned map entry once through `bootstrap.initial_recovery.error` with app QName and partition ID, and passes only the returned IDs to `DeployAppPartitions` when the fatal error is nil. It skips that call when the slice is empty. `InitializeCheckpoints` does not also log partition failures. The map is used only for bootstrap logging; there is no deployment-error registry or new `IAppPartitions` method. Excluded IDs have no `appPartitionRT`, actualizer, scheduler, or recovery worker. The existing actualizer and scheduler startup in `DeployAppPartitions` therefore cannot start asynchronous projectors (including the checkpoint projector) or jobs for excluded IDs. Healthy partitions continue to start normally.

`Borrow` keeps its existing `appparts.ErrNotFound` contract. On that error, the command processor uses the existing `AppPartsCount` registration lookup to distinguish a missing application (`400 Bad Request`, `application not found`) from a missing partition of a registered application (`500 Internal Server Error`, `partition not found`), without parsing error text or introducing new app-partitions errors. These command errors omit application QName and partition ID; command error logging preserves the request context and supplies `vapp=<appQName>` and `partid=<partitionID>` even when borrowing fails. `ErrNotAvailableEngines` remains `503`. Every subsequent command to an excluded partition receives the same not-found outcome, including after the underlying storage recovers. It starts no runtime recovery, executes no command, and writes no log event. Other runtime-recovery admission conditions for deployed partitions retain `503`; query response policy is unchanged by this change.

The same deployment gate applies to cluster, built-in, and sidecar partitions. The cluster application has one required partition: bootstrap returns an error if its workspace, checkpoints, or partition cannot initialize or deploy, before registering other apps or calling `c.cluster.DeployApp`. It does not use the earlier read-only compatibility fallback or defer other apps while reporting ready. Partition failures in non-cluster applications remain isolated. Missing completion markers are retried on a later bootstrap; automatic in-process redeployment is outside this change.

### Descriptor count and selective partition deployment

`AppDeploymentDescriptor.NumParts` supplies the full count to `c.cluster.DeployApp`, local app registration, `AppPartsCount`, and routing. The cluster command validates that descriptor against `cluster.App` and persists it on first deployment. It does not inspect checkpoint outcomes or runtime deployment counts. A genuine configured count change still produces the existing `409 Conflict`; a matching configured and persisted count remains compatible despite excluded partitions.

Checkpoint initialization returns a separate list of eligible IDs for `deployAppPartitions`. For a descriptor count of `3`, partition `1` may fail while only IDs `[0, 2]` deploy. The count remains `3`, so WSID `10` still maps to partition `1` and receives `500` with `partition not found`. Replacing the count with `2` would map that workspace to partition `0`, permitting writes against a different partition history and risking overwritten data. Preserve the count and original IDs even when no partition of a non-cluster application deploys.

The initializer and deployment helper remain separate. The original count is never reconstructed from the successful-ID slice, its maximum ID, runtime partitions, or the failure map. See the [routing-safety decision](./decisions.md#safety-declared-partition-count-is-independent-of-checkpoint-initialization).

### Checkpoint rationale

Store the offset of the last-seen wlog ID-bearing event

#### Why?

- Store the last handled offset + last IDs: wrong, IDs may be recorded incorrectly
- Store the last handled offset: wrong, we do not know last IDs
- Store both the latest ID-bearing offset seen by the projector and the last handled offset, perform a log tail scan from the last handled offset, then read the ID-bearing event separately if that scan finds no IDs: wrong, because the asynchronous projector may not yet have seen the actual latest ID-bearing event in the WLog.
  - we have to prove that IDs are really last so we need a log tail scan anyway

Problematic example:

```text
                     ID-bearing                         ID-bearing
                      ID 1001                         IDs 1002..1005           wlog end
                         |                                  |                      |
                         v                                  v                      v
WLog: ---- 1 ---- 2 ---- 3 ---- 4 ---- 5 ---- 6 ---- 7 ---- 8 ---- 9 ---- 10 ---- 11
                         ^                                  ^                      ^
                         |                                  |                      |
          latest ID-bearing offset = 3        actual latest ID-bearing    stored last offset = 10
          seen by checkpoint projector        event, not yet seen by      not correct but easily recovered by
                                              checkpoint projector        ReadWLog(10, ReadToTheEnd)

ReadWLog(10, ReadToTheEnd)  -> offset 10 exists; actual log end is 11; no ID-bearing event found
ReadWLog(3, 1)              -> offset 3 exists and is ID-bearing; validation passes
nextRecordID                -> 1002 <- WRONG
correct nextRecordID        -> 1006

The checkpoint correctly identifies event 3 as the latest ID-bearing event seen by the projector, but nothing in either read proves that the projector had reached event 7. Event 7 remains invisible to this recovery path.
```

#### Recovery flow

The safe recovery path must perform a log tail scan from `wlogIDBearingOffset` through the actual log end:

```text
                     ID-bearing                         ID-bearing
                      ID 1001                         IDs 1002..1005           wlog end
                         |                                  |                      |
                         v                                  v                      v
WLog: ---- 1 ---- 2 ---- 3 ---- 4 ---- 5 ---- 6 ---- 7 ---- 8 ---- 9 ---- 10 ---- 11
                         ^                                  ^                      ^
                     checkpoint                             |                      |
                         |                            sees last IDs      sees last wlog offset
                         |----------- ReadWLog(3, ReadToTheEnd) ------------------>|


latest ID-bearing event found = 7     -> nextRecordID   = 1006
actual final event found      = 11    -> nextWLogOffset = 12
```

So:

- Store exactly one `wlogIDBearingOffset`: the latest ID-bearing event seen by the checkpoint projector.
- Require it to point to an ID-bearing event.
- Treat it only as an inclusive scan start, not as proof that the event is the latest ID-bearing event in the WLog.

The remaining tradeoff is recovery time when the projector lags or the WLog has many non-ID-bearing events after the checkpoint:

```text
                     checkpoint
                   IDs 1001..1003                                                            WLog end
                          |                                                                      |
                          v                                                                      v
WLog: ------------------ 41 ---- non-ID ---- non-ID ---- ... 1,000,000 non-ID events ... ---- 1000041
                          |<--------------- ReadWLog(41, ReadToTheEnd) ------------------------->|

events read       = 1,000,001
nextRecordID      = 1004
nextWLogOffset    = 1000042
```

#### Bad cases on recovery

What if the PLog offset is recorded incorrectly?

- `lastPLogOffset` is absent or zero -> this is not an invalid checkpoint -> `ReadPLog(FirstOffset, ReadToTheEnd)` once, with no checkpoint-error log
- `ReadPLog(lastPLogOffset, ReadToTheEnd)`
  - no callback -> checkpoint invalid -> `ReadPLog(FirstOffset, ReadToTheEnd)`
  - first callback offset differs from `lastPLogOffset` -> checkpoint invalid -> `ReadPLog(FirstOffset, ReadToTheEnd)`

What if the WLog ID-bearing offset is recorded incorrectly?

- `wlogIDBearingOffset` is absent or zero -> this is not an invalid checkpoint -> `ReadWLog(FirstOffset, ReadToTheEnd)` once, with no checkpoint-error log
- `ReadWLog(wlogIDBearingOffset, ReadToTheEnd)`
  - no callback e.g. the `wlogIDBearingOffset` is beyond the end) -> checkpoint invalid -> `ReadWLog(FirstOffset, ReadToTheEnd)`
  - first callback offset differs from `wlogIDBearingOffset` -> checkpoint invalid -> `ReadWLog(FirstOffset, ReadToTheEnd)`
  - event in the first callback contains no new non-singleton IDs -> checkpoint invalid -> `ReadWLog(FirstOffset, ReadToTheEnd)`

Out of scope:

- Changing workspace-ID allocation or Sequence Trust Level write policy.

References (internal):

- [command/impl.go](../../../../../pkg/processors/command/impl.go)
- [command/provide.go](../../../../../pkg/processors/command/provide.go)
- [checkpoint contracts](../../../../../pkg/checkpoints/types.go)
- [checkpoint projector](../../../../../pkg/checkpoints/impl.go)
- [checkpoint registration](../../../../../pkg/checkpoints/provide.go)
- [storage/impl_recoverycheckpoint.go](../../../../../pkg/vvm/storage/impl_recoverycheckpoint.go)
- [istructs/events-types.go](../../../../../pkg/istructs/events-types.go)
- [vvm/types.go](../../../../../pkg/vvm/types.go)
- [recovery logging contract](../../../../specs/prod/apps/logging--td.md#command-processor)
- [AIR-5009 recovery requirements](./issue-AIR-5009.md)

References (external):

- [post-removal sequence-storage baseline in PR #4649](https://github.com/voedger/voedger/pull/4649)
- [partition-recovery performance and shared-storage scope in AIR-4959](https://untill.atlassian.net/browse/AIR-4959)

## Functional design

- [x] update: [apps/recovery.feature](../../../../specs/prod/apps/recovery.feature)
  - specify unchanged descriptor and advertised partition counts after one or every non-cluster application partition fails checkpoint initialization; routed commands keep their original partition and cannot write elsewhere
  - retain `409 Conflict` for a genuinely changed configured descriptor after partial deployment
  - specify that failure of the cluster application's sole partition stops bootstrap before other application registration or provisioning
  - align admission examples with the implemented four-worker limit; keep zero-capacity behavior as a generic-manager technical test rather than a selectable VVM configuration
  - specify that partition reset preserves the four-worker limit across detached and replacement workspace managers

- [x] create: [apps/sequences.feature](../../../../specs/prod/apps/sequences.feature)
  - Feature specification for successful, externally observable allocation of PLog offsets, WLog offsets, ordinary record IDs, singleton IDs, and workspace IDs; exclude failed-command behavior, implementation-level persistence policy, restart, checkpoint, and recovery behavior

- [x] rename: [apps/recovery-checkpoints.feature](../../../../specs/prod/apps/recovery-checkpoints.feature) -> [apps/recovery.feature](../../../../specs/prod/apps/recovery.feature)
  - rename the feature to partition and workspace recovery
  - add a rule for bootstrap-time initial recovery when a declared partition has no checkpoint
  - define known partitions as the complete declared `0..NumPartitions-1` range and include empty partitions in initialization
  - specify that every missing partition checkpoint is computed from the complete PLog and persisted after the workspace checkpoints derived from that partition
  - specify that every WSID encountered in the scan receives a workspace checkpoint, using zero when no ID-bearing event was found
  - specify empty-PLog initialization as a present zero-valued partition checkpoint
  - original failure scenarios are superseded by the completed undeployed-partition update below
  - specify that successful initial recovery initializes checkpoint storage only and that the first later request still enters normal partition recovery followed by lazy workspace recovery
  - retain request-driven partition/workspace recovery, checkpoint validation, fallback, empty-log defaults, admission limits, and retry behavior under recovery-focused rules
  - remove normal sequence-allocation behavior; it belongs in `sequences.feature`

- [x] update: [apps/recovery.feature](../../../../specs/prod/apps/recovery.feature)
  - replace the initial-failure `503`/runtime-retry scenario with one-partition and all-partitions failure cases: the server becomes ready, successful partitions are deployed, and failed partitions are not deployed
  - specify that excluded partitions run no asynchronous projectors or scheduled jobs, including after other partitions begin handling commands
  - verify every repeated command routed to an excluded partition returns `500 Internal Server Error` with `partition not found`, even if storage later becomes healthy; initial-recovery causes appear only in logs and no request starts recovery or deployment
  - cover retry on a later bootstrap, retaining completed markers and deploying previously failed partitions only after successful initialization
  - retain the existing runtime-recovery admission/retry scenarios for deployed partitions and cross-reference them rather than duplicating them

## Technical design

- [x] update: [apps/arch-deployment.md](../../../../specs/prod/apps/arch-deployment.md)
  - separate descriptor count from runtime deployment IDs; keep `c.cluster.DeployApp` independent of checkpoint outcomes and preserve compatibility validation
  - document modulo-routing safety and the unchanged count for partially or completely undeployed non-cluster applications
  - replace the unavailable-cluster fallback with the required cluster partition and fatal bootstrap error

- [x] update: [apps/recovery--td.md](../../../../specs/prod/apps/recovery--td.md)
  - add technical flows for preserved routing, descriptor mismatch after partial deployment, and fatal cluster-partition initialization failure
  - use the implemented `sys.ApplyRecoveryCheckpoint` projector name and locate the storage interface in `checkpoints/interface.go`
  - document the fixed four-worker production limit, admission-before-attempt construction, and retained failures waiting for an available retry slot
  - distinguish the generic manager's own worker join from the missing command-service join of workspace workers; track that integration gap as pending construction
  - make the workspace-recovery slot pool stable by partition key so reset does not multiply admission capacity
  - identify existing test coverage and explain the real WSID used to represent the feature's logical workspace 10 through API v1

- [x] update: [apps/logging--td.md](../../../../specs/prod/apps/logging--td.md)
  - distinguish nonfatal per-partition logging from the required cluster partition's bootstrap failure

- [x] delete: [apps/arch-sequences.md](../../../../specs/prod/apps/arch-sequences.md)
  - replace its current-behavior content with the focused sequence-allocation and recovery feature technical designs below; retain historical material only through the existing historical references

- [x] create: [apps/sequences--td.md](../../../../specs/prod/apps/sequences--td.md)
  - Feature Technical Design whose Key concepts define sequence state, next values, partition/workspace/application allocation scopes, ordinary versus singleton record IDs, and successful-write advancement; cover the components, persistence boundaries, Sequence Trust Level policy, and failure-induced gaps needed to explain normal PLog-offset, WLog-offset, record-ID, and workspace-ID issuance, cite existing technical-test evidence and the implemented VIT allocation tests, and explicitly exclude checkpointing and recovery

- [x] create: [apps/recovery--td.md](../../../../specs/prod/apps/recovery--td.md)
  - Feature Technical Design whose Key concepts define recovery checkpoints, partition and workspace checkpoints, their zero-value initialization sentinels, ID-bearing events, initial recovery, the initial-recovery completion marker, inclusive log tail scans, checkpoint fallback, partition recovery, and workspace recovery; cover bootstrap orchestration, checkpoint production and storage, request-driven recovery, admission and retry lifecycles, failure ordering, logging, and shutdown

- [x] update: [apps/recovery--td.md](../../../../specs/prod/apps/recovery--td.md)
  - document the deployable-ID slice, failed-partition error map, fatal setup/context error, and bootstrap-owned per-partition error logging; retain the deployment gate and distinction between undeployed-partition not-found `500` and deployed-partition runtime recovery
  - distinguish missing-application `400` from missing-partition `500` using the existing app-registration lookup; use concise request errors with identity in structured log fields and leave `IAppPartitions` errors unchanged
  - retain CPU-bounded concurrent PLog scans, per-application `GOMAXPROCS(0)` sampling, waiting for all attempts, and workspace-before-partition persistence
  - update the initial-failure flows and invariants to exclude failed partitions from deployment, actualizers, and schedulers; reference the deployment architecture for the unavailable cluster-partition path

- [x] update: [apps/arch.md](../../../../specs/prod/apps/arch.md)
  - remove `arch-sequences.md` as the canonical Sequences subsystem architecture
  - link `sequences.feature` with `sequences--td.md` and `recovery.feature` with `recovery--td.md` as the focused feature designs

- [x] update: [apps/arch2-sequences.md](../../../../specs/prod/apps/arch2-sequences.md)
  - replace the link to the deleted canonical sequence architecture with links to the new sequence-allocation and recovery specifications while retaining this file as historical context only

- [x] update: [apps/arch-deployment.md](../../../../specs/prod/apps/arch-deployment.md)
  - document that bootstrap invokes the checkpoint initial-recovery component after app registration and before partition deployment
  - document bounded per-application partition initialization, waiting for every scheduled attempt, and deployment of only partitions whose returned outcome is ready
  - document the deployable-ID/error-map/fatal-error return contract and bootstrap logging each failed partition once before deploying only eligible IDs
  - document app registration without partition runtime creation and suppression of actualizers and schedulers for excluded partitions, using the existing app-partitions contract and errors
  - document the unavailable cluster-partition branch: return an error before other app registration or command-dependent provisioning; no retrying deployment commands or successful readiness
  - define readiness as independent of non-cluster partition-scoped initial-recovery results, including the all-partitions-failed case
  - distinguish fatal setup, cancellation, and cluster-partition failures from isolated failures inside non-cluster partition attempts

- [x] update: [apps/logging--td.md](../../../../specs/prod/apps/logging--td.md)
  - document: partition and workspace recovery admission and failure messages emitted through `cp.error`
  - document: partition and workspace recovery lifecycle stages, context attributes, and completion messages
  - document: malformed-checkpoint error stages, messages, recovery context attributes, and fallback to `FirstOffset`
  - align malformed-checkpoint messages with the exact `retrying from FirstOffset` suffix emitted by command recovery

- [x] update: [apps/logging--td.md](../../../../specs/prod/apps/logging--td.md)
  - document initial-recovery start, per-partition completion, failure, and app completion stages under the bootstrap/recovery context
  - require app QName and partition ID on per-partition initial-recovery logs, distinguish scan failures from checkpoint-storage failures, and state that non-cluster errors are partition-isolated while cluster-partition failure stops bootstrap

- [x] update: [apps/logging--td.md](../../../../specs/prod/apps/logging--td.md)
  - document concise `cp.error` messages: `application not found` with `400` for missing applications and `partition not found` with `500` for missing partitions; keep identity in `vapp` and `partid` log attributes, including borrow failures, and initial-recovery failure details in bootstrap logs, distinct from runtime-recovery start, failure, and retry messages
  - identify bootstrap as the emitter of one `bootstrap.initial_recovery.error` per returned partition-error map entry, with app QName, partition ID, failed operation, and underlying cause; the initializer returns these errors without logging them again

## Construction

Gherkin scenario test placement:

- Implement the normal allocation scenarios in [apps/sequences.feature](../../../../specs/prod/apps/sequences.feature) as VIT integration tests in [sys/it/impl_sequences_test.go](../../../../../pkg/sys/it/impl_sequences_test.go); reuse or move exact existing coverage instead of duplicating it.
- Implement the scenarios in [apps/recovery.feature](../../../../specs/prod/apps/recovery.feature) as VIT integration tests in [sys/it/impl_recovery_test.go](../../../../../pkg/sys/it/impl_recovery_test.go).
- Implement the empty-PLog and empty-WLog scenarios as mocked command-processor tests because VIT cannot create those states.
- Cover "Missing checkpoints are computed for every declared partition" in [checkpoints/initial_recovery_test.go](../../../../../pkg/checkpoints/initial_recovery_test.go): its empty partition must have no WLog history, which VIT provisioning cannot produce. Do not delete a provisioned PLog while retaining its WLogs to simulate this case.
- Use a mocked command-processor test for any other scenario only when its VIT implementation proves impossible or disproportionately complicated. Document the concrete reason next to that fallback test; otherwise, do not duplicate Gherkin scenario coverage in command-processor tests.

### Tests

- [x] update: [command/impl_test.go](../../../../../pkg/processors/command/impl_test.go)
  - align the existing four-workspace admission test's scenario name and Given comments with the synchronized feature; retain zero-capacity coverage in generic recovery-manager tests
  - verify command-service shutdown cannot finish while a canceled workspace recovery is still exiting, including workers belonging to a partition detached by reset; reuse the existing recovery gates
  - both shutdown regression cases pass after the `command/impl.go` worker-lifecycle fix, including the early-service-exit assertion
  - verify that four workspace recoveries detached by partition reset still occupy all capacity seen by the replacement partition until one finishes

- [x] update: [sys/it/impl_test.go](../../../../../pkg/sys/it/impl_test.go)
  - resolve the preserved profile workspace from the second-run VIT instead of referencing the first callback's out-of-scope `ws`; retain the no-reprovisioning assertion

- [x] create: [checkpoints/export_test.go](../../../../../pkg/checkpoints/export_test.go)
  - keep projector implementation access confined to test builds for external-package checkpoint tests
  - expose `applyRecoveryCheckpoint` to external-package tests without exporting the production implementation

- [x] update: [coreutils/mock_test.go](../../../../../pkg/coreutils/mock_test.go)
  - verify the shared PLog-event mock returns its configured handling partition and satisfies the extended event contract

- [x] update: [sys/it/impl_bootstrap_test.go](../../../../../pkg/sys/it/impl_bootstrap_test.go)
  - reuse the bootstrap operation recorder and a delegating federation recorder to verify `c.cluster.DeployApp` receives the descriptor's full `NumParts`, including first deployment and repeated startup with one or all non-cluster application partitions excluded
  - assert local registration and `AppPartsCount` retain the full count while `DeployAppPartitions` receives only successful original IDs, or is skipped when no IDs succeed
  - align the existing cluster-failure test with the new feature scenario, retaining checks for the returned error, error logging, no later app registration, and no cluster deployment commands

- [x] update: [sys/it/impl_recovery_test.go](../../../../../pkg/sys/it/impl_recovery_test.go)
  - implement both Examples rows of `Checkpoint failures preserve the descriptor partition count` using existing storage-failure and operation-recording helpers; preserve the feature's exact rows and step comments
  - prepare existing workspace data before restart; use HTTP requests through the router and a WSID whose modulo result differs for descriptor count 3 versus successful count 2; verify the original failed partition receives the request, the response is `500` with `partition not found`, and no PLog/WLog write or runtime recovery occurs in any partition
  - implement `A changed descriptor partition count is rejected after partial deployment`; verify `409 Conflict`, unchanged persisted descriptor, and no writes or remapped app startup
  - keep descriptor mismatch coverage distinct from availability-only failure coverage: a genuine configured count change must still be rejected

- [x] create: [sys/it/impl_sequences_test.go](../../../../../pkg/sys/it/impl_sequences_test.go)
  - VIT scenario tests for `sequences.feature`
  - verify successful partition-scoped PLog-offset issuance, workspace-scoped WLog-offset and record-ID issuance, singleton-ID behavior, and persistent application-scoped WSID allocation; keep failed-command and recovery cases out of these feature scenarios
  - move or rename exact existing tests where possible; use `sequences: scn:` names matching the feature scenarios, verbatim step comments at the code that realizes each step, and exact Examples header/row comments and placeholder mappings for every Scenario Outline row

- [x] update: [checkpoints/initial_recovery_test.go](../../../../../pkg/checkpoints/initial_recovery_test.go)
  - unit tests for bootstrap-time checkpoint reconstruction over multiple declared partitions and workspaces
  - cover every partition in the declared `0..NumPartitions-1` range, present and absent checkpoints, empty PLogs with no allocated storage, workspaces with and without ID-bearing events, and selection of the latest ID-bearing WLog offset
  - verify returned deployable partition IDs and the partition-error map are disjoint and exhaustive when the fatal error is nil; verify continuation after one scan or storage failure, workspace-before-partition persistence order, retry on the next bootstrap, no writes for a completed partition, context cancellation, and absence of any runtime recovery/publication side effect
  - assert one operation-specific wrapped error per failed partition in the returned map, preserving the underlying cause; leave error-log assertions to bootstrap tests
  - cover present-zero checkpoints as deployable, checkpoint-read errors in the error map, an empty ID slice and a complete error map when all attempts fail, and no deployment from partial results after a fatal setup/cancellation error

- [x] update: [sys/it/impl_bootstrap_test.go](../../../../../pkg/sys/it/impl_bootstrap_test.go)
  - use a recording wrapper around real initial recovery to verify registration, completed initialization, and selective deployment in order; inject failures through dependencies rather than reproducing logging or recovery logic in the wrapper
  - record the partition IDs passed to `DeployAppPartitions`, verify failed IDs are excluded while healthy IDs deploy, and verify the deployment call is absent when every partition fails
  - capture one `bootstrap.initial_recovery.error` log per returned map entry with app QName, partition ID, failed operation, and underlying error; verify errors are not logged a second time by the initializer
  - verify successful readiness after non-cluster application partition failures; failure of the cluster application's sole partition returns an error before processing other apps or sending deployment commands
  - retain logging and no-deployment assertions for the failed cluster partition; the earlier read-only validation and deferred-provisioning fallback is superseded
  - preserve setup/cancellation and descriptor-incompatibility failure coverage; leave feature scenarios and checkpoint-value assertions in `impl_recovery_test.go`

- [x] update: [appparts/impl_test.go](../../../../../pkg/appparts/impl_test.go)
  - verify omitted partition IDs have no runtime partition and invoke neither the asynchronous actualizer runner nor the scheduler/job runner
  - verify selected healthy IDs still start both runners and that declared partition counts/routing stay unchanged; include the all-excluded case
  - verify `Borrow` for an omitted partition returns the existing unknown-partition `ErrNotFound` and standard message without any initial-recovery state

- [x] update: [command/impl_test.go](../../../../../pkg/processors/command/impl_test.go)
  - add focused technical coverage for borrow-error translation: missing application returns `400` with `application not found`, missing partition of a registered application returns `500` with `partition not found`, and engine exhaustion remains `503`; use the existing app-registration lookup rather than error-text matching or new app-partitions errors
  - assert command-error logs retain request context and include `vapp=<appQName>` and `partid=<partitionID>` even on borrow failure, while not-found messages omit identifiers and initial-recovery causes
  - verify repeated rejected commands never start runtime recovery, allocate offsets, or write events; retain existing `503` and retry coverage for deployed partitions

- [x] create: [cluster/impl_deployapp_test.go](../../../../../pkg/cluster/impl_deployapp_test.go)
  - focused tests for the cluster command's read-only deployment-descriptor validator
  - cover matching descriptors, incompatible partition/workspace counts, absent descriptors, and lookup errors; assert validation performs no provisioning or log writes

- [x] update: [sys/it/impl_recovery_test.go](../../../../../pkg/sys/it/impl_recovery_test.go)
  - replace the initial-failure scenario's runtime `503`/`500`/retry assertions with the revised feature's immediate and repeated undeployed-partition `500` and `partition not found` assertions; preserve verbatim step comments for the revised message
  - verify healthy partitions accept commands and excluded partitions have no asynchronous projector or scheduler/job instances, including an application with all partitions excluded
  - release the injected storage failure while the server runs and prove commands still receive `500` with `partition not found`; after a later restart prove successful initialization deploys the partition
  - remove `initialRecoveryReadFailure` runtime caller-stack detection, retry gates, and counters when they become unused; retain only dependency failure injection needed by the revised scenarios

- [x] update: [sys/it/impl_recovery_test.go](../../../../../pkg/sys/it/impl_recovery_test.go)
  - rename feature traceability from `recovery-checkpoints: scn:` to `recovery: scn:` and align step comments with `recovery.feature`
  - replace request-time absent-partition-checkpoint scenarios with bootstrap initial-recovery scenarios
  - retain direct technical coverage for defensive request-time handling of a checkpoint removed after bootstrap
  - simulate a pre-checkpoint deployment by creating PLog histories in shared storage and removing their partition completion markers before the next VVM launch
  - verify that bootstrap initializes every missing declared partition, persists the actual PLog end offset, creates a workspace checkpoint for every WSID encountered, and stores zero for empty/non-ID-bearing histories
  - original initial-failure assertions are superseded by the completed selective-deployment VIT update above
  - verify across shared-storage VVM runs that initial recovery creates checkpoints once and later bootstrap runs preserve completed partition markers
  - verify that checkpoint initialization does not eagerly recover runtime state: the first command still receives partition-recovery `503`, the next reaches lazy workspace recovery, and only the later retry succeeds
  - give every feature scenario an exact `recovery: scn:` subtest name, place each verbatim step comment immediately above the code that realizes it, and preserve exact Examples header/row comments and placeholder mappings

- [x] update: [command/impl_test.go](../../../../../pkg/processors/command/impl_test.go)
  - rename retained feature scenario tests to the `recovery: scn:` prefix
  - keep request-time missing-partition-checkpoint behavior as a technical fallback test rather than an initial-recovery feature scenario
  - verify that absent and zero-valued partition and workspace checkpoints each cause exactly one read from `FirstOffset`, no validation attempt, no fallback read, and no invalid-checkpoint log

- [x] update: [checkpoints/impl_test.go](../../../../../pkg/checkpoints/impl_test.go)
  - exercise the shared ID-bearing-event predicate used by both initial recovery and the asynchronous checkpoint projector

- [x] update: [sys/it/impl_recovery_test.go](../../../../../pkg/sys/it/impl_recovery_test.go)
  - realign every VIT scenario subtest, exact Examples row, and step comment with the updated feature specification
  - for every partition-checkpoint case, assert the first request returns partition-recovery `503`, the next request returns workspace-recovery `503`, and only the request after workspace recovery succeeds
  - use `200001` through `200006` in scenario traceability comments and verify that the dynamically allocated IDs equal those values
  - assert the complete invalid-checkpoint log messages ending with `FirstOffset`, and assert their absence for absent, zero, and valid checkpoints
  - implement every deterministic non-empty-log history in VIT, including the WLog-end boundary, all non-ID-bearing event kinds after the checkpoint, a WLog without user record IDs, and checkpoints pointing to each non-ID-bearing event kind
  - leave recovery admission and failure scenarios to focused command-processor tests because pausing or failing recovery through a full VVM would require test-only lifecycle controls
  - make recovery-state observations deterministic: use dedicated untouched workspaces, disable transparent request retries for expected `503` responses, and wait for explicit VIT subscriptions or recovery lifecycle completion before issuing the next scenario request
  - verify the successful post-recovery command by reading from its returned PLog offset through the end, matching the event to the target workspace, and asserting that no later PLog event exists instead of checking only that the log end advanced
  - continue reading from the successful insert's returned WLog offset through the end and assert that exactly that insert event exists there, with no later WLog event

- [x] update: [sys/it/impl_recovery_test.go](../../../../../pkg/sys/it/impl_recovery_test.go)
  - replace semantic Scenario Outline disambiguators such as `zero`, `valid but outdated`, `valid`, `beyond end`, `missing event`, and event-kind abbreviations with the exact distinguishing Examples cell value, so all 20 non-empty-log Examples rows are directly discoverable from their final `recovery: scn:` names
  - preserve the existing exact Examples header/row comments, placeholder mappings, distinct checkpoint kinds, fixture construction, and behavioral assertions; do not duplicate the existing scenario tests or expand the grouped technical tests
  - correct the seven workspace-checkpoint step comments from `Given the stored workspace checkpoint is <checkpoint>` to the feature's verbatim `And the stored workspace checkpoint is <checkpoint>`
  - explicitly retain the already exact `updates of existing records` case and its `checkpointExistingRecordSuffix` path, and the CUD-free and existing-record-update invalid-checkpoint paths at offsets 12 and 14

- [x] update: [command/impl_test.go](../../../../../pkg/processors/command/impl_test.go)
  - keep the grouped `TestRecoveryCheckpoints_RecordIDsRecovery` subtests as supplementary technical coverage rather than treating their generic names as feature-scenario identities
  - implement absent, zero, and invalid checkpoints over empty PLogs and WLogs with mocked logs, the only log state that VIT cannot construct
  - drive each empty-log scenario through the partition and workspace recovery managers' request-equivalent `503` stages and verify the next PLog offset, WLog offset, and record ID; direct manager tests are required because provisioning a command-capable fixture makes the logs non-empty
  - use a mocked test for another Gherkin scenario only when deterministic VIT construction is impossible or disproportionately complicated, and document that concrete reason next to the test
  - implement the four-worker workspace admission limit, duplicate suppression, and retained-failure/report/retry here with documented VIT-complexity reasons; keep zero-capacity behavior in generic-manager tests
  - prove that an excess workspace-recovery request is not queued by observing no start before a later request is submitted after a slot is released
  - verify the retained-failure/report/retry scenario through its externally observable `503`, `500`, `503`, and success sequence
  - assert complete invalid-checkpoint messages, including the `FirstOffset` fallback wording, rather than matching a proper prefix of a longer message

- [x] update: [command/impl_test.go](../../../../../pkg/processors/command/impl_test.go)
  - replace the `zero` aliases in the empty-PLog and empty-WLog Scenario Outline subtest names with the exact `lastPLogOffset 0` and `wlogIDBearingOffset 0` Examples values
  - preserve the existing exact Examples header/row comments, placeholder mappings, mocked recovery flow, and assertions; do not duplicate the four empty-log scenario tests

- [x] update: [checkpoints/impl_test.go](../../../../../pkg/checkpoints/impl_test.go)
  - verify that one projector stores `lastPLogOffset` when `PLogOffset % 100 == 0`
  - verify that each event allocating non-singleton record IDs replaces `wlogIDBearingOffset`, so it identifies the latest such event seen by the projector, while singleton, existing-record, and CUD-free events do not write a workspace checkpoint
  - verify storage-error propagation and registration of the single standard asynchronous projector

- [x] update: [command/impl_test.go](../../../../../pkg/processors/command/impl_test.go)
  - implement the empty-PLog and empty-WLog scenarios from the renamed [apps/recovery.feature](../../../../specs/prod/apps/recovery.feature) with mocked logs
  - implement another Gherkin scenario here only if its VIT implementation is impossible or disproportionately complicated, and document the concrete reason for the fallback
  - verify one inclusive PLog read from the saved `lastPLogOffset` to the end
  - verify that partition recovery does not start workspace recovery and that a subsequent request starts it from the command stage
  - verify one inclusive WLog read from the workspace's saved `wlogIDBearingOffset` to the end
  - verify that the final WLog event determines the next WLog offset
  - verify that a later ID-bearing event found by the log tail scan, rather than the stored checkpoint event, determines the next record ID
  - verify that singleton-only and CUD-free events after the checkpoint still contribute to the actual WLog end
  - verify that a history without non-singleton CUDs starts allocation at `FirstUserRecordID`
  - verify that empty PLog and WLog scans start offsets at `FirstOffset` and an empty WLog starts record allocation at `FirstUserRecordID`
  - verify one recovery attempt per WSID, `503` while it is running, and retained failure/retry behavior
  - verify that the production concurrency limit admits at most four workspace recoveries, returns `503` without queueing excess work, logs the limit condition, and permits a later retry after a slot is released
  - verify distinct log messages for partition-in-progress and workspace-in-progress responses
  - verify that a partition recovery admission-limit result is translated to `503 Service Unavailable` rather than a panic
  - retain zero-capacity admission coverage in `recovery_manager_test.go`, since the command service always constructs four slots
  - verify that an invalid checkpoint over an empty log logs an error, performs at most one fallback scan, and retains the empty-log defaults

- [x] create: [command/recovery_manager_test.go](../../../../../pkg/processors/command/recovery_manager_test.go)
  - directly verify generic recovery-manager startup, ready lookup, duplicate suppression, retained failures, retries, admission limits, hooks, cancellation, reset, stale-completion rejection, and shutdown clearing
  - relocate stale-completion coverage from the partition-manager integration-style test

- [x] create: [testutils/storage_test.go](../../../../../pkg/checkpoints/testutils/storage_test.go)
  - verify shared checkpoint-test-storage key isolation, missing versus present-zero values, last-write-wins replacement, and concurrent access
  - verify fault injection, reentrant hooks, forced setup bypassing observations, delegated backend behavior, successful-write ordering, and rejection of unexpected strict-mock calls

- [x] create: [command/recovery_test_utils_test.go](../../../../../pkg/processors/command/recovery_test_utils_test.go)
  - centralize deterministic test support for asynchronous partition and workspace recovery
  - provide recovery-attempt gates, injected failures, completion waits, and inclusive PLog/WLog tail scan observations
  - provide a retrying request sender for command tests whose subject is unrelated to lazy recovery

- [x] delete: [command/test_utils.go](../../../../../pkg/processors/command/test_utils.go)
  - replace the partition-only recovery test controls with the generic test-only utilities in `recovery_test_utils_test.go`

- [x] update: [storage/impl_recoverycheckpoint_test.go](../../../../../pkg/vvm/storage/impl_recoverycheckpoint_test.go)
  - verify single-field PLog and WLog JSON values named `lastPLogOffset` and `wlogIDBearingOffset`, including missing and malformed values and last-write-wins replacement
  - verify application/partition/workspace isolation through partition-key suffixes and the fixed singleton clustering column `[]byte{1}`
  - construct the test subject through `NewRecoveryCheckpointStorage` and the `IRecoveryCheckpointStorage` contract so the storage tests compile
  - compare decoded JSON numbers using the representation returned by `encoding/json` so the isolation cases pass

- [x] update: [storage/consts_test.go](../../../../../pkg/vvm/storage/consts_test.go)
  - preserve fixed sequence-storage prefix values

- [x] update: [actualizers/impl_helpers_test.go](../../../../../pkg/processors/actualizers/impl_helpers_test.go)
  - extend PLog event mocks with handling-partition and PLog-offset accessors used by the checkpoint projector

- [x] update: [istructsmem/event-types_test.go](../../../../../pkg/istructsmem/event-types_test.go)
  - use the exported WLog storage-key helper in the event-read failure test

- [x] update: [sys/it/impl_recovery_test.go](../../../../../pkg/sys/it/impl_recovery_test.go)
  - implement every non-empty-log scenario in the renamed [apps/recovery.feature](../../../../specs/prod/apps/recovery.feature) as a VIT integration test
  - cover absent and zero checkpoints, earlier valid offsets, and current PLog and WLog checkpoints from the scenario examples
  - cover PLog and WLog checkpoints beyond the log end, checkpoints whose exact event is missing, and a WLog checkpoint pointing to a non-ID-bearing event
  - verify `wlogIDBearingOffset` across sequential VVM instances sharing storage
  - verify through the next successful CUD response that restart preserves each workspace's next WLog offset and record ID
  - verify through the next successful CUD response that a singleton event at the WLog end advances the WLog offset but not the record ID
  - verify through the next successful CUD response that an event at the PLog end for another workspace in the same partition does not advance the first workspace's WLog offset or record ID
  - overwrite persisted recovery checkpoints between VVM runs with PLog and WLog offsets beyond the log end, zero offsets, and a WLog offset of a non-ID-bearing update event
  - record the final pre-restart insert's WLog offset and NewID, then verify through the next successful CUD response and its persisted WLog event that every overwritten-checkpoint case returns both values incremented by one
  - verify dedicated malformed-checkpoint error logs for checkpoints beyond the log end and non-ID-bearing checkpoints, and no such errors for zero checkpoints
  - use one unchanged VIT config for both VVM runs and resolve the same named workspaces through `vit.WS` after restart
  - remove the second-run-only configuration and any manual reconstruction of an `AppWorkspace`

- [x] update: [sys/it/impl_test.go](../../../../../pkg/sys/it/impl_test.go)
  - extend the storage-preserving VIT restart test to record the owner profile WLog end before restart
  - assert at the beginning of the second run that the log end is unchanged, proving that config-declared child workspaces were not provisioned again

- [x] update: [vit/utils.go](../../../../../pkg/vit/utils.go)
  - retain the shared-storage two-VVM lifecycle helper and its hooks for inspecting stable storage after each VVM stops
  - use the same `VITConfig` by default and retain the existing optional second-run configuration for upgrade tests
  - after the first-run hook, retain `appWorkspaces` and `principals` as two explicit maps
  - launch the second VIT with config-entity provisioning disabled, assign both retained maps, and then run common VIT initialization so principal tokens are refreshed
  - keep the existing optional second-run configuration argument without adding a configuration callback or an aggregate type for the retained maps

- [x] update: [vit/impl.go](../../../../../pkg/vit/impl.go)
  - extract common VIT initialization performed after VVM launch so normal creation and storage-preserving restart apply options, validate the email captor, advance test time, refresh principal tokens, and capture the initial goroutine count consistently
  - replace the ambiguous launch-only switch with an explicit `createEntities` decision
  - when `createEntities` is false, still apply the VVM config, register workspace templates, and launch services, but skip sign-up, workspace initialization, configured document and subject creation, and post-initialization fixture actions
  - keep `appWorkspaces` and `principals` as explicit inputs on the restart path rather than defining a generic or vaguely named holder type

### Descriptor partition count

- [x] update: [btstrp/impl.go](../../../../../pkg/btstrp/impl.go)
  - make the descriptor source explicit for `c.cluster.DeployApp` request `NumPartitions` and local app registration; never derive it from initialized or deployed IDs
  - preserve separate `initializeAppCheckpoints` and `deployAppPartitions` operations; pass only eligible original IDs to runtime deployment without changing or renumbering the descriptor's declared partition range
  - retain fatal cluster-partition handling and the normal cluster command's compatibility/provisioning path; non-cluster partition failures must not alter command arguments or descriptor validation

- [x] update: [appparts/types.go](../../../../../pkg/appparts/types.go)
  - document `AppDeploymentDescriptor.NumParts` as the full declared routing count, independent of checkpoint success and runtime availability; changing that count requires compatibility validation and cannot be used to hide excluded partitions

### Initial recovery

- [x] create: [btstrp/utils.go](../../../../../pkg/btstrp/utils.go)
  - extract `SplitBuiltInApps` for shared production and test bootstrap preparation
  - preserve application order, exclude sidecars from the remaining built-ins, and reject a missing cluster application

- [x] update: [coreutils/mock.go](../../../../../pkg/coreutils/mock.go)
  - extend `MockPLogEvent` with its PLog offset and handling partition, and provide strict application-structures/provider mocks reused by checkpoint tests

- [x] consolidate: [coreutils/mock_plogevent.go](../../../../../pkg/coreutils/mock_plogevent.go) into [coreutils/mock.go](../../../../../pkg/coreutils/mock.go)
  - keep the handling-partition accessor beside the existing PLog-event mock methods
  - remove the separate file without changing configured expectations or interface coverage

- [x] create: [checkpoints/interface.go](../../../../../pkg/checkpoints/interface.go)
  - declare `IRecoveryCheckpointStorage` with partition/workspace reads returning value, presence, and error
  - define unconditional partition/workspace writes over their respective application-scoped keys

- [x] create: [testutils/mock.go](../../../../../pkg/checkpoints/testutils/mock.go)
  - provide strict `MockCheckpointStorage` expectations for all checkpoint-storage operations
  - verify expectations during cleanup and never fall back to a memory implementation on unexpected calls

- [x] create: [testutils/storage.go](../../../../../pkg/checkpoints/testutils/storage.go)
  - provide shared thread-safe memory checkpoint storage and a delegating backend wrapper for initial, command, and bootstrap recovery tests
  - isolate keys by application and partition/workspace, preserve present-zero values, and overwrite unconditionally
  - inject faults by full operation key, run hooks outside locks, record successful typed writes, and expose semantic presence/order assertions
  - provide forced fixture writes that bypass faults, hooks, and recorded production writes

- [x] create: [checkpoints/initial_recovery.go](../../../../../pkg/checkpoints/initial_recovery.go)
  - bootstrap-time `IInitialRecovery` implementation backed by the app-structures provider and recovery-checkpoint storage
  - `NewInitialRecovery` constructor and `InitializeCheckpoints(ctx, appQName, numPartitions) ([]istructs.PartitionID, map[istructs.PartitionID]error, error)` operation that enumerates `0..numPartitions-1`, includes already completed partitions in the deployable IDs, and scans each missing partition from `FirstOffset` through the PLog end
  - per-partition accumulation of the final PLog offset and latest ID-bearing WLog offset for every encountered WSID, including zero-valued checkpoints for empty or non-ID-bearing histories
  - production worker limit sampled from `runtime.GOMAXPROCS(0)` at the start of each `InitializeCheckpoints` call
  - bounded worker scheduling that waits for every partition attempt and does not cancel siblings after a partition-scoped failure
  - ordered persistence of all computed workspace checkpoints before the partition completion marker; collect each checkpoint-read/scan/write failure in the returned error map with its operation and wrapped cause, omit that partition from deployable IDs, and return only application-level setup/context failures through the fatal error result
  - collect concurrent outcomes safely and do not emit partition-failure logs here; bootstrap owns logging the returned errors

- [x] update: [checkpoints/types.go](../../../../../pkg/checkpoints/types.go)
  - define `IInitialRecovery` with `InitializeCheckpoints(ctx, appQName, numPartitions) ([]istructs.PartitionID, map[istructs.PartitionID]error, error)` for injection into bootstrap
  - document the deployable-ID slice, failed-partition error map, their disjoint/exhaustive membership when the fatal error is nil, the valid empty slice when every partition fails, and no deployment from partial results when the fatal error is non-nil
  - document checkpoint presence, rather than a nonzero offset, as the initial-recovery completion marker
  - document zero-valued partition and workspace checkpoints produced for empty and non-ID-bearing histories and identify both zero values as non-validating full-scan sentinels

- [x] update: [checkpoints/impl.go](../../../../../pkg/checkpoints/impl.go)
  - extract one ID-bearing-event predicate shared by initial recovery and `sys.ApplyRecoveryCheckpoint`
  - keep the asynchronous projector as the normal post-bootstrap checkpoint producer

- [x] update: [btstrp/impl.go](../../../../../pkg/btstrp/impl.go)
  - accept the service context and injected initial-recovery contract
  - invoke initial recovery after each app is registered with app structures and before `DeployAppPartitions` starts its partition workers
  - apply the same ordering to the cluster app, built-in apps, and sidecar apps
  - log each returned partition-error map entry once with app QName and partition ID, then pass only the returned deployable IDs to `DeployAppPartitions` if the fatal error is nil; skip deployment when the slice is empty and continue to server readiness after any number of partition-scoped failures, without retaining their causes in app partitions or command handling
  - require successful initialization and deployment of the cluster application's sole partition; return an appropriate error and stop before processing other applications when it fails

- [x] update: [cluster/impl_deployapp.go](../../../../../pkg/cluster/impl_deployapp.go)
  - retain read-only deployment-descriptor lookup/compatibility checks within the normal cluster command path; preserve existing conflict messages and keep first-deployment writes/provisioning there; checkpoint outcomes must not participate in these checks

- [x] update: [command/impl.go](../../../../../pkg/processors/command/impl.go)
  - on `appparts.ErrNotFound`, use the existing `AppPartsCount` registration lookup to distinguish missing application (`400 Bad Request`, `application not found`) from missing partition of a registered application (`500 Internal Server Error`, `partition not found`), without parsing error text or changing the app-partitions error contract; keep `ErrNotAvailableEngines` at `503`
  - omit application QName and partition ID from these request-error messages; structured command-error log fields provide that identity
  - reject before runtime partition/workspace recovery, command execution, offset allocation, or log writes; preserve `503` for deployed-partition engine exhaustion and runtime recovery admission

- [x] update: [command/provide.go](../../../../../pkg/processors/command/provide.go)
  - enrich command error logging with `vapp=<appQName>` and `partid=<partitionID>` from the command message while retaining the request context, including when `Borrow` fails before runtime partition state is available

- [x] update: [vvm/provide.go](../../../../../pkg/vvm/provide.go)
  - construct `checkpoints.IInitialRecovery` from the app-structures provider and recovery-checkpoint storage
  - inject it and the bootstrap operator context into `btstrp.Bootstrap`

- [x] regenerate: [vvm/wire_gen.go](../../../../../pkg/vvm/wire_gen.go)
  - reflect construction and bootstrap injection of the checkpoint initial-recovery component

### Checkpoint contracts and storage

- [x] update: [istructs/events-types.go](../../../../../pkg/istructs/events-types.go)
  - expose handling partition and PLog offset on persisted PLog events for checkpoint projection
  - keep `IIDGenerator` free of checkpoint-specific record-ID accessors

- [x] update: [istructsmem/utils.go](../../../../../pkg/istructsmem/utils.go)
  - export the PLog and WLog storage-key helpers used to construct exact missing-event checkpoint cases

- [x] update: [istructsmem/impl.go](../../../../../pkg/istructsmem/impl.go)
  - use the exported PLog and WLog storage-key helpers for event persistence and reads
  - complete the WLog helper rename at the three remaining `WLogKey` calls; those undefined references currently prevent compilation

- [x] update: [storage/consts.go](../../../../../pkg/vvm/storage/consts.go)
  - retain the sequence-storage prefix values and document partition ID and WSID as partition-key suffixes rather than clustering columns
  - replace the `lastWLogOffsetWithNewRecordIDs` JSON field constant with `wlogIDBearingOffset`

- [x] update: [storage/impl_recoverycheckpoint.go](../../../../../pkg/vvm/storage/impl_recoverycheckpoint.go)
  - append partition ID or WSID to the corresponding application partition key and use `[]byte{1}` as the clustering column
  - encode only `lastPLogOffset` for PLog checkpoints and only `wlogIDBearingOffset` for WLog checkpoints
  - report missing values as absent, reject malformed values, and overwrite unconditionally
  - decode both checkpoint kinds through one required-offset decoder without unused map results or a generic scalar helper

- [x] update: [storage/provide.go](../../../../../pkg/vvm/storage/provide.go)
  - construct the recovery-checkpoint adapter over shared system-VVM storage

- [x] create: [checkpoints/consts.go](../../../../../pkg/checkpoints/consts.go)
  - define the 100-event partition-checkpoint interval and `QNameApplyRecoveryCheckpoint` for `sys.ApplyRecoveryCheckpoint`

- [x] update: [checkpoints/types.go](../../../../../pkg/checkpoints/types.go)
  - define a PLog checkpoint containing only `lastPLogOffset` and a WLog checkpoint containing only `wlogIDBearingOffset`
  - name the Go workspace-checkpoint field `WLogIDBearingOffset`; its documentation must state that it identifies the latest ID-bearing event found by its producer but need not identify the latest ID-bearing event in the WLog
  - keep the partition and workspace checkpoint storage interface in `checkpoints/interface.go`

- [x] update: [checkpoints/impl.go](../../../../../pkg/checkpoints/impl.go)
  - implement one recovery-checkpoint projector for both checkpoint types
  - replace `wlogIDBearingOffset` whenever the projector sees an ID-bearing event, making it the latest such offset seen by the projector without making recovery depend on the projector having reached the WLog end
  - persist the PLog offset when `PLogOffset % 100 == 0`

- [x] create: [checkpoints/provide.go](../../../../../pkg/checkpoints/provide.go)
  - register the single built-in asynchronous recovery-checkpoint projector with stateless resources

- [x] update: [sys/sys.vsql](../../../../../pkg/sys/sys.vsql)
  - replace both built-in recovery-checkpoint declarations with one projector using `AFTER EXECUTE ON ALL COMMANDS` together with the CUD and ODoc triggers

- [x] update: [pkg/sys/sys.vsql](../../../../../pkg/sys/it/testdata/apps/test2.app1/image/pkg/sys/sys.vsql)
  - mirror the single built-in recovery-checkpoint projector declaration in the generated integration-test schema

- [x] update: [sys/sysprovide/provide.go](../../../../../pkg/sys/sysprovide/provide.go)
  - register the single checkpoint projector with stateless resources

### Command recovery

- [x] update: [command/impl.go](../../../../../pkg/processors/command/impl.go)
  - join workspace recovery workers during command-service shutdown, including workers owned by partitions detached after reset; do not rely on service-context cancellation alone
  - preserve the existing admission, retry, partition-resource ownership, and stale-completion behavior while completing the worker lifecycle

- [x] update: [command/consts.go](../../../../../pkg/processors/command/consts.go)
  - define shared errors for recovery-in-progress, concurrency-limit, and retained-failure outcomes
  - define the fixed per-partition workspace recovery limit `numWSRecoverers = 4`

- [x] create: [command/recovery_manager.go](../../../../../pkg/processors/command/recovery_manager.go)
  - implement the generic keyed recovery lifecycle shared by partitions and workspaces
  - reserve optional concurrency slots before creating attempts, retain failures for one reporting request, and publish successful recovered values
  - share the admission and startup path between new attempts and retries while preserving the distinct returned errors
  - reject stale completions after reset and join each manager's own workers before clearing its state during shutdown
  - define the generic recoverable value, attempt factory, hooks, and worker/slot bookkeeping; create request-owned resources only after admission

- [x] update: [command/types.go](../../../../../pkg/processors/command/types.go)
  - define partition and workspace keys and manager wrappers over the shared generic recovery manager
  - retain application identity and structures in recovered partitions for asynchronous WLog recovery

- [x] update: [command/impl.go](../../../../../pkg/processors/command/impl.go)
  - recover the partition with one inclusive PLog tail scan and use only its final event
  - publish the recovered partition without starting workspace recovery from the partition-recovery goroutine
  - start workspace recovery on demand from the command pipeline's workspace stage
  - recover the WLog end offset and latest non-singleton record ID with one inclusive log tail scan from `wlogIDBearingOffset` through the actual log end
  - initialize empty PLog and WLog offsets with `FirstOffset` and an empty WLog record-ID generator with `FirstUserRecordID`
  - ignore singleton IDs and retain `FirstUserRecordID` when the complete WLog contains no newly allocated non-singleton CUD
  - let the generic manager deduplicate attempts, enforce admission, run workers and hooks, retain failures, reject stale completions, publish results, and account for worker shutdown
  - expose the lifecycle through one generic `getOrStart` operation returning only the recovered value or an error, without a separate state enum or decision object
  - inject the partition recovery function once into the partition manager instead of forwarding it through each partition lookup
  - keep recovery startup, logging, hooks, and HTTP error policy in the thin partition and workspace managers
  - reserve a slot before starting a goroutine and return `503` both when full and for an in-progress WSID
  - translate a generic partition recovery-limit result to a logged `503 Service Unavailable` response rather than panicking
  - prefix recovery admission and failure messages with the affected partition ID or workspace ID
  - attach `vapp=sys/voedger`, `extension=sys._Recovery`, and `partid` to partition recovery lifecycle logs, and add `wsid` to workspace recovery lifecycle logs
  - log workspace recovery start, initial WLog tail scan failure, and completion using dedicated `cp.workspace_recovery.*` stages
  - remove recovery-time checkpoint writes and all persisted record-ID handling
  - validate that a stored checkpoint identifies the exact first event returned by its inclusive log tail scan, and that a WLog checkpoint event is ID-bearing
  - abort an invalid log tail scan and retry once from `FirstOffset`, using the full fallback scan instead of a separate first-event probe
  - recover empty fallback scans with `FirstOffset` and `FirstUserRecordID` defaults and never loop beyond two total scans
  - log malformed checkpoints through the dedicated partition or workspace error stage with existing recovery context attributes
  - leave checkpoint repair to the asynchronous projector rather than writing checkpoint storage during recovery
  - propagate PLog write errors so the command pipeline can schedule partition recovery
  - keep the four-slot workspace recovery pool in the partition manager by partition key and attach it before publishing recovered partition state, preserving capacity across resets

- [x] update: [command/provide.go](../../../../../pkg/processors/command/provide.go)
  - inject checkpoint storage and partition/workspace recovery hooks into command-service construction; production uses the fixed four-worker limit
  - run rate limiting, command lookup, execute authorization, and request-body validation after partition admission but before workspace admission; only requests that pass these gates may start a WLog scan
  - log command errors with application and partition identity
  - preserve service-context cancellation and the partition-manager shutdown call, which joins the shared partition/workspace worker wait group

### VVM wiring

- [x] update: [vvm/provide.go](../../../../../pkg/vvm/provide.go)
  - wire shared checkpoint storage into stateless resources and the command service; use `btstrp.SplitBuiltInApps` for bootstrap preparation

- [x] regenerate: [vvm/wire_gen.go](../../../../../pkg/vvm/wire_gen.go)
  - reflect the updated Wire provider graph
