# Feature technical design: Partition and workspace recovery

Technical design for [Partition and workspace recovery](./recovery.feature).
Normal offset and identifier issuance is specified separately in
[Sequence allocation](./sequences--td.md).

## Key concepts

- **Recovery checkpoint**: a persisted log offset used as an inclusive recovery
  scan start. It accelerates recovery but is not authoritative sequence state.
- **Partition checkpoint**: `lastPLogOffset`, the latest handled PLog offset
  selected by initial recovery or the asynchronous checkpoint projector.
- **Workspace checkpoint**: `wlogIDBearingOffset`, an ID-bearing WLog offset
  from a locally generated event. Such an event allocates beyond all earlier
  IDs, so scanning from it preserves coverage of the maximum allocated ID.
  Synchronized inserts (`c.sys.Init`)may contain lower IDs and do not advance the checkpoint.
  The offset is never treated as the WLog end.
- **Zero-value initialization sentinel**: a present zero partition checkpoint
  marks an initialized empty PLog; a present zero workspace checkpoint marks an
  encountered workspace with no locally generated ID-bearing event, including
  sync-only history. Either zero selects one scan
  from `FirstOffset` without validation or invalid-checkpoint logging.
- **ID-bearing event**: an event that allocates at least one new non-singleton
  record ID at or above `FirstUserRecordID`, including logged argument roots
  and descendants as well as newly created CUD rows.
- **Initial recovery**: bootstrap-time computation and persistence of missing
  checkpoints for every declared application partition. It does not publish
  runtime partition or workspace state.
- **Initial-recovery completion marker**: presence of a partition checkpoint,
  including a zero-valued one. Workspace checkpoints derived from a partition
  are stored before this marker.
- **Deployable partition**: a declared partition whose existing completion marker was
  read successfully or whose missing checkpoints were reconstructed and
  persisted successfully by the current initial-recovery invocation.
- **Undeployed partition**: a declared partition for which no runtime partition
  has been created. An initial-recovery failure excludes that ID from deployment
  for the current server run. The application remains registered for routing,
  but no asynchronous actualizers or schedulers exist for the excluded ID.
- **Inclusive log tail scan**: `ReadPLog` or `ReadWLog` begins at the checkpointed
  event and continues through the actual log end.
- **Checkpoint fallback**: one scan from `FirstOffset` after a nonzero checkpoint
  fails validation. Recovery does not repair the stored checkpoint.
- **Partition recovery**: service-scoped, request-triggered restoration of the
  next PLog offset followed by reapplication of the final event.
- **Workspace recovery**: lazy, request-triggered restoration of one workspace's
  next WLog offset and ordinary record-ID generator.

Every WLog event originates in its partition's PLog. An empty PLog therefore
implies empty WLogs for all workspaces in that partition. Empty-partition
fixtures must preserve both logs; clearing only a provisioned PLog produces an
impossible history and is not a recovery scenario.

## External actors

Systems:

- `*Client`
  - Submits commands whose partition and workspace state may require recovery.

## Components

### Layers

```text
Bootstrap
    |
    +-- [Bootstrap operator]
    +-- [Checkpoint initial recovery]
    |
    v
Request recovery
    |
    +-- [Command processor]
    +-- [Partition recovery manager]
    +-- [Workspace recovery manager]
    +-- [Offsets bookkeeping]
    +-- [Record ID generator]
    |
    v
Checkpointing
    |
    +-- [Recovery checkpoint projector]
    +-- [(Recovery checkpoint storage)]
    |
    v
Application operations
    |
    +-- [Application events]
    +-- [Application records]
    +-- [App partitions engine]
```

### Bootstrap

- `[Bootstrap operator]`
  - Registers each application with `[App partitions engine]`, invokes
    `[Checkpoint initial recovery]`, logs each returned partition-error map
    entry once with app QName and partition ID, and passes only the returned
    deployable IDs to `DeployAppPartitions`. An empty ID slice skips that call.
    The descriptor's full `NumParts` remains available for routing and is supplied
    unchanged as `NumPartitions` to `c.cluster.DeployApp`. Checkpoint outcomes
    affect only the IDs passed to partition deployment.
  - A returned setup or cancellation error stops bootstrap without deploying
    partitions from that invocation. Partition-scoped failures exclude only
    the affected IDs; for non-cluster applications readiness remains successful
    even when no IDs are returned.
  - Applies the same gate to cluster, built-in, and sidecar applications. The
    bootstrap dependency path when the cluster partition is unavailable belongs
    to [Apps deployment](./arch-deployment.md#bootstrap-failure-and-readiness);
    bootstrap returns an error if that sole partition cannot initialize or deploy,
    before processing other applications.
  - impl: [pkg/btstrp/impl.go](../../../../pkg/btstrp/impl.go)

- `[Checkpoint initial recovery]`
  - `checkpoints.IInitialRecovery.InitializeCheckpoints(ctx, appQName, numPartitions)` returns
    `([]istructs.PartitionID, map[istructs.PartitionID]error, error)`: deployable
    IDs, operation-specific errors keyed by failed partition ID, and an optional
    fatal app-structures setup or context error. IDs belong to `0..numPartitions-1`.
    When the fatal error is nil, the slice and map are disjoint and cover every
    declared partition exactly once. An empty slice and a complete error map
    with a nil fatal error are valid. A non-nil fatal error prevents deployment
    from the returned IDs; collected partition errors can still be logged.
  - Obtains the application's event storage and samples `runtime.GOMAXPROCS(0)`
    once per call. A present partition checkpoint, including zero, makes that
    partition deployable without scanning or rewriting it. Each missing partition
    is scheduled concurrently under the sampled CPU bound and reads its PLog
    sequentially from `FirstOffset` through the actual log end. The attempt
    retains the final PLog offset and latest locally generated ID-bearing WLog offset for every
    WSID encountered.
  - It stores a workspace checkpoint for every encountered WSID, using zero when
    no locally generated ID-bearing event was found, and stores the partition checkpoint last,
    using zero for an empty PLog. The partition is added to the deployable IDs
    only after all these writes succeed. A failed reconstruction writes no
    completion marker and is repeated on a later bootstrap.
  - Every partition checkpoint-read, PLog-read, or checkpoint-write error omits
    that ID from the deployable slice and adds it to the error map, retaining
    the failed operation and wrapping the underlying cause. Concurrent attempts
    collect their outcomes safely; the returned map is complete once all
    scheduled attempts finish. A checkpoint-read failure never authorizes
    deployment, even if a stored marker might exist. Partition-failure logging
    belongs to `[Bootstrap operator]`;
    the initializer does not duplicate those logs. No retained cause or
    error-registration method is added to `[App partitions engine]` or
    `[Command processor]`.
  - Waits for all scheduled attempts without canceling siblings after a
    partition-scoped failure. Only app-structures setup failure before scheduling
    or bootstrap-context cancellation is returned as the fatal error.
  - decl: [pkg/checkpoints/types.go#IInitialRecovery](../../../../pkg/checkpoints/types.go)
  - impl: [pkg/checkpoints/initial_recovery.go](../../../../pkg/checkpoints/initial_recovery.go)

### Recovery on command request

- `[Command processor]`
  - Borrows the runtime partition before authentication and workspace checks.
    When `Borrow` returns `appparts.ErrNotFound`, uses the existing
    `IAppPartitions.AppPartsCount` registration lookup to distinguish a missing
    application from a missing partition, without parsing error text or adding
    a new app-partitions error contract. A missing application returns
    `400 Bad Request` with `application not found`; a missing partition of a
    registered application returns `500 Internal Server Error` with
    `partition not found`. Neither message repeats application or partition
    identifiers: command error logs carry `vapp=<appQName>` and
    `partid=<partitionID>`, including when borrowing fails. Engine exhaustion
    (`ErrNotAvailableEngines`) remains `503`.
  - Repeated commands to an undeployed partition return the same not-found
    error, even after its storage becomes healthy, and never reach runtime
    recovery, command execution, offset allocation, or event writes. A later
    bootstrap must successfully initialize the partition before it can deploy.
  - For a deployed partition, runs partition recovery admission after
    authentication/workspace checks and workspace recovery admission after a
    recovered partition is available. A command does not execute on the request
    that starts either recovery.
  - impl: [pkg/processors/command/provide.go](../../../../pkg/processors/command/provide.go)
  - impl: [pkg/processors/command/impl.go](../../../../pkg/processors/command/impl.go)

- `[Partition recovery manager]`
  - Owns states keyed by application and partition. Concurrent requests share
    one attempt; a failure is retained for the next request, which receives the
    failure while starting exactly one retry when capacity is available. A full
    manager returns the concurrency-limit error without consuming the retained
    failure. Production partition recovery has no concurrency bound.
  - Uses the service context. Reset detaches old state so stale completion cannot
    publish into a replacement. The caller cancels the context; the generic
    manager's shutdown waits for its registered workers and then clears its state.
    New attempts and retries share one capacity check and startup path.
  - Constructs an attempt only after admission reserves a slot. For partition
    recovery this is when the request's borrowed partition is transferred to
    the recovery workpiece, which releases it when the attempt finishes.
  - impl: [pkg/processors/command/recovery_manager.go](../../../../pkg/processors/command/recovery_manager.go)
  - impl: [pkg/processors/command/impl.go](../../../../pkg/processors/command/impl.go)

- `[Workspace recovery manager]`
  - Lives within a recovered partition, owns states keyed by application,
    partition, and WSID. It uses a four-slot pool owned by the partition manager,
    keyed by application and partition, and set by the command processor's
    `numWSRecoverers` constant. Replacement partition state reuses the same pool,
    so workers detached by reset continue to consume capacity until they finish.
    No VVM configuration field changes this limit.
    Saturation returns `503` without creating state or queueing work.
  - Reuses the generic manager's retained-failure, retry, stale-completion, and
    service-context behavior. The generic constructor also supports zero
    capacity for technical tests, where no attempt is admitted.
  - Before a recovered partition is published, its workspace manager shares the
    partition manager's worker wait group. The existing shutdown call therefore
    joins workspace workers even when reset has detached their partition.
  - impl: [pkg/processors/command/recovery_manager.go](../../../../pkg/processors/command/recovery_manager.go)
  - impl: [pkg/processors/command/impl.go](../../../../pkg/processors/command/impl.go)

- `[Offsets bookkeeping]`
  - Holds `appPartition.nextPLogOffset` and `workspace.NextWLogOffset`. Recovery
    sets each value from the actual last callback plus one, or `FirstOffset` for
    an empty complete scan.
  - impl: [pkg/processors/command/types.go](../../../../pkg/processors/command/types.go)

- `[Record ID generator]`
  - Starts at `FirstUserRecordID`; workspace recovery advances it from the maximum
    newly allocated non-singleton ID encountered in the complete WLog tail scan.
    A later synchronized insert with a lower ID must not lower this maximum.
    A stored checkpoint pointing at a synchronized event is unsafe and triggers
    the existing full-scan fallback, covering checkpoints written by older producers.
  - impl: [pkg/istructsmem/idgenerator.go](../../../../pkg/istructsmem/idgenerator.go)

### Checkpointing

- `[Recovery checkpoint projector]`
  - The single built-in asynchronous `sys.ApplyRecoveryCheckpoint` stores a
    partition checkpoint every 100 PLog events and replaces the workspace
    checkpoint whenever it sees a locally generated ID-bearing event. Both initial
    recovery and this projector use the same checkpoint eligibility predicate.
    Synchronized events leave the earlier checkpoint in place; a workspace with
    only synchronized events is recovered with a full scan. No allocated IDs or
    in-memory maximum-ID cache are added to checkpoint storage.
  - impl: [pkg/checkpoints/impl.go](../../../../pkg/checkpoints/impl.go)
  - impl: [pkg/checkpoints/provide.go](../../../../pkg/checkpoints/provide.go)

- `[(Recovery checkpoint storage)]`
  - Last-write-wins storage for independent partition and workspace JSON values.
    Partition keys include application and partition; workspace keys include
    application and WSID. Recovery reads but never rewrites checkpoints.
  - decl: [pkg/checkpoints/interface.go](../../../../pkg/checkpoints/interface.go)
  - impl: [pkg/vvm/storage/impl_recoverycheckpoint.go](../../../../pkg/vvm/storage/impl_recoverycheckpoint.go)

### Application operations

- `[Application events]`
  - Supplies complete PLog/WLog reads or inclusive log tail scans. Initial
    recovery reads PLogs only; partition recovery reads a PLog; workspace
    recovery reads a WLog.
  - decl: [pkg/istructs/interface.go](../../../../pkg/istructs/interface.go)
  - impl: [pkg/istructsmem/impl.go](../../../../pkg/istructsmem/impl.go)

- `[Application records]`
  - Reapplies the final recovered partition event through the event-reapply path;
    record and WLog writes are unconditional and consume no new PLog offset.
  - impl: [pkg/istructsmem/impl.go](../../../../pkg/istructsmem/impl.go)

- `[App partitions engine]`
  - Registers application structures and the full declared partition count
    before initial recovery. `DeployAppPartitions` creates runtime partitions
    and starts actualizers and schedulers only for the supplied deployable IDs.
    Excluded IDs therefore start no asynchronous projectors, checkpoint
    projectors, or scheduled jobs, including after healthy partitions begin work.
  - `Borrow` for an absent runtime partition keeps returning the existing
    `ErrNotFound` error. The contract and errors are unchanged; no initial-recovery
    failure state is stored here. Query response policy is unchanged.
  - Invokes synchronous projectors when runtime partition recovery reapplies
    the final event for a deployed partition.
  - decl: [pkg/appparts/interface.go](../../../../pkg/appparts/interface.go)
  - impl: [pkg/appparts/impl.go](../../../../pkg/appparts/impl.go)

## Scenarios

### Initial recovery

#### Missing checkpoints are computed for every declared partition

```text
[Bootstrap operator]
  -> [App partitions engine]: register application with partitions 0..2
  -> [Checkpoint initial recovery]: InitializeCheckpoints(application, 3)
      -> [(Recovery checkpoint storage)]: partition 0 present -> skip
      -> [Application events]: scan complete PLogs for missing partitions 1 and 2
      -> [(Recovery checkpoint storage)]: partition 1 workspace checkpoints 10=6, 11=0
      -> [(Recovery checkpoint storage)]: partition 1 checkpoint=50 (completion marker)
      -> [(Recovery checkpoint storage)]: partition 2 checkpoint=0 (empty completion marker)
      - return deployable IDs [0, 1, 2], empty error map, nil
  -> [App partitions engine]: DeployAppPartitions(application, [0, 1, 2])
```

Workspace writes for one partition complete before its partition marker. The
relative ordering of different partition attempts is unconstrained.

#### Initial recovery initializes checkpoints but not runtime sequence state

```text
[Bootstrap operator]
  -> [Checkpoint initial recovery]: persist missing checkpoints only
  -> [App partitions engine]: deploy partition workers; server ready
*Client
  -> [Command processor]: first command
      -> [Partition recovery manager]: start recovery -> 503
  -> [Command processor]: retry after partition recovery
      -> [Workspace recovery manager]: start lazy workspace recovery -> 503
  -> [Command processor]: retry after workspace recovery -> command succeeds
```

#### Initial recovery failures do not prevent server readiness

```text
[Checkpoint initial recovery]
  -> scheduled missing partitions, bounded by runtime.GOMAXPROCS(0)
      -> [Application events]: one or every PLog scan fails
      - collect each partition-scoped failure in the error map; write no marker for it
      - let every sibling attempt finish; wait for all scheduled attempts
  -> [Bootstrap operator]: return deployable IDs, partition-error map, nil
      - log one bootstrap.initial_recovery.error per map entry with app QName and partition ID
      -> [App partitions engine]: deploy only returned IDs; skip call if empty
      - failed IDs have no runtime partitions, actualizers, or schedulers
      - server ready even if every partition was excluded
*Client
  -> [Command processor]: command for excluded partition
      -> [App partitions engine]: Borrow -> existing ErrNotFound
      - return 500: partition not found
  -> [Command processor]: repeat command after storage becomes healthy
      -> [App partitions engine]: Borrow -> same ErrNotFound -> 500
      - no runtime recovery, deployment, or event writes
```

If setup fails before partition scheduling or the bootstrap context is canceled,
`InitializeCheckpoints` returns a fatal error and `[Bootstrap operator]` does not deploy that
application's partitions. Readiness is otherwise independent of how many
scheduled non-cluster application partition attempts failed, including all of
them. Failure of the cluster application's sole partition stops bootstrap.

#### Checkpoint failures preserve the descriptor partition count

```text
[Bootstrap operator]
  - application descriptor NumParts=3; workspace 10 belongs to partition 1
  -> [App partitions engine]: register full declared count 3
  -> [Checkpoint initial recovery]: partition 1, or every partition, fails
  -> [App partitions engine]: deploy eligible IDs [0, 2], or skip deployment
  - preserve descriptor and advertised partition count 3
*Client
  -> [Command processor]: routed command for workspace 10, using 10 % 3 = 1
      -> [App partitions engine]: Borrow(partition 1) -> ErrNotFound
      - return 500: partition not found
      - do not reroute using the successful count or append events in any partition
```

Bootstrap's `c.cluster.DeployApp` request supplies the descriptor's full count
before initial recovery; the command has no checkpoint-outcome input. Its
compatibility check remains independent of the runtime deployment set. The
source-of-count contract and overwrite risk are specified in
[Preserve declared partition count after partial initialization](./arch-deployment.md#preserve-declared-partition-count-after-partial-initialization).

#### A changed descriptor partition count is rejected after partial deployment

```text
[Bootstrap operator]
  - previous run: descriptor count 3, runtime partitions [0, 2]
  - configured descriptor on restart: count 2
  - c.cluster.DeployApp validates the configured descriptor
      - persisted count 3 differs -> existing 409 Conflict
  - stop bootstrap; preserve persisted count and workspace placement
```

The descriptor-validation implementation and components belong to
[Reject incompatible redeployment](./arch-deployment.md#reject-incompatible-redeployment).

#### Cluster partition failure stops bootstrap

```text
[Bootstrap operator]
  -> [Checkpoint initial recovery]: cluster partition fails; no eligible ID
  - log the partition error and return a cluster partition error
  - stop before other application registration, provisioning, or partition deployment
  - server does not become ready
```

#### Healthy partitions continue while a failed partition remains undeployed

```text
[Bootstrap operator]
  -> [Checkpoint initial recovery]: partition 1 fails; return [0, 2], {1: error}, nil
  - log partition 1 error once with app QName and partition ID
  -> [App partitions engine]: DeployAppPartitions(test1/app1, [0, 2])
      - start configured actualizers and schedulers only for partitions 0 and 2
*Client
  -> [Command processor]: commands for deployed partitions 0 and 2
      - follow existing runtime recovery scenarios; later retries succeed
  -> [Command processor]: command for partition 1
      -> [App partitions engine]: Borrow -> ErrNotFound -> 500
      - partition 1 still has no asynchronous projectors or scheduled jobs
```

#### A later bootstrap deploys a previously failed partition after successful initial recovery

```text
[Bootstrap operator]
  - restart after partition 1 storage becomes healthy
  -> [App partitions engine]: register test1/app1 in the new server run
  -> [Checkpoint initial recovery]: InitializeCheckpoints(test1/app1, 3)
      -> [(Recovery checkpoint storage)]: partitions 0 and 2 present -> preserve
      -> [Application events]: scan partition 1 from FirstOffset through PLog end
      -> [(Recovery checkpoint storage)]: workspace checkpoints, then partition 1 marker
      - return [0, 1, 2], empty error map, nil
  -> [App partitions engine]: deploy partitions 0..2; server ready
*Client
  -> [Command processor]: command for deployed partition 1
      - follow existing runtime recovery scenarios; later retry succeeds
```

### Partition checkpoint

#### Partition recovery starts from a checkpoint

```text
*Client
  -> [Command processor]: command
      -> [Partition recovery manager]: start -> 503
          -> [(Recovery checkpoint storage)]: absent, zero, or valid lastPLogOffset
          -> [Application events]: ReadPLog(checkpoint or FirstOffset, through log end)
          -> [Offsets bookkeeping]: nextPLogOffset = actual log end + 1
          -> [Application records]: reapply final event when present
  -> [Command processor]: later request enters workspace recovery
```

Absent and zero checkpoints select `FirstOffset` directly. A nonzero checkpoint
must equal the first returned event offset.

#### Invalid partition checkpoint falls back to the PLog beginning

```text
[Partition recovery manager]
  -> [(Recovery checkpoint storage)]: nonzero lastPLogOffset
  -> [Application events]: inclusive log tail scan returns no callback or a different first offset
  - log cp.partition_recovery.checkpoint.invalid
  -> [Application events]: one fallback ReadPLog(FirstOffset, through log end)
  -> [Offsets bookkeeping]: restore nextPLogOffset from fallback's actual log end
```

#### Empty PLog recovery starts at FirstOffset

```text
[Partition recovery manager]
  -> [(Recovery checkpoint storage)]: absent or zero checkpoint
  -> [Application events]: ReadPLog(FirstOffset, through log end) -> empty
  -> [Offsets bookkeeping]: nextPLogOffset = FirstOffset
  - publish partition without reapplication or invalid-checkpoint log
```

#### Invalid partition checkpoint over an empty PLog

```text
[Partition recovery manager]
  -> [Application events]: read nonzero checkpoint -> no callback
  - log invalid checkpoint
  -> [Application events]: fallback from FirstOffset -> empty
  -> [Offsets bookkeeping]: nextPLogOffset = FirstOffset
```

### Workspace checkpoint

#### Workspace recovery starts from a checkpoint

```text
*Client
  -> [Command processor]: command after partition recovery
      -> [Workspace recovery manager]: reserve slot; start -> 503
          -> [(Recovery checkpoint storage)]: absent, zero, or valid wlogIDBearingOffset
          -> [Application events]: ReadWLog(checkpoint or FirstOffset, through log end)
          -> [Offsets bookkeeping]: NextWLogOffset = actual log end + 1
          -> [Record ID generator]: advance from maximum allocated ID found by the log tail scan
  -> [Command processor]: later retry succeeds with following offset and record ID
```

#### Workspace recovery starts from an ID-bearing checkpoint at the WLog end

```text
[Workspace recovery manager]
  -> [Application events]: ReadWLog(ID-bearing checkpoint at log end, through log end)
      - first and final callback are the checkpoint event
  -> [Offsets bookkeeping]: NextWLogOffset = checkpoint offset + 1
  -> [Record ID generator]: next ID = greatest new ordinary ID + 1
```

#### Non-ID-bearing log tail events advance only the WLog offset

```text
[Workspace recovery manager]
  -> [Application events]: scan from ID-bearing checkpoint through CUD-free,
                           singleton-only, or existing-record-update events at log end
  -> [Offsets bookkeeping]: advance from the actual final event
  -> [Record ID generator]: advance only from the earlier ID-bearing event
```

#### Workspace without user record IDs starts allocation at FirstUserRecordID

```text
[Workspace recovery manager]
  -> [Application events]: complete WLog scan finds no ID-bearing event
  -> [Offsets bookkeeping]: NextWLogOffset = actual log end + 1
  -> [Record ID generator]: retain FirstUserRecordID
```

#### Invalid workspace checkpoint falls back to the WLog beginning

```text
[Workspace recovery manager]
  -> [Application events]: inclusive log tail scan from nonzero checkpoint
      - no callback, different first offset, or first event is not ID-bearing
  - log cp.workspace_recovery.checkpoint.invalid
  -> [Application events]: one fallback ReadWLog(FirstOffset, through log end)
  -> [Offsets bookkeeping] + [Record ID generator]: restore from complete fallback
```

### Empty WLog

#### Empty WLog recovery starts at FirstOffset and FirstUserRecordID

```text
[Workspace recovery manager]
  -> [(Recovery checkpoint storage)]: absent or zero checkpoint
  -> [Application events]: ReadWLog(FirstOffset, through log end) -> empty
  -> [Offsets bookkeeping]: NextWLogOffset = FirstOffset
  -> [Record ID generator]: next ID = FirstUserRecordID
```

#### Invalid workspace checkpoint over an empty WLog

```text
[Workspace recovery manager]
  -> [Application events]: read nonzero checkpoint -> no callback
  - log invalid checkpoint
  -> [Application events]: fallback from FirstOffset -> empty
  -> [Offsets bookkeeping]: NextWLogOffset = FirstOffset
  -> [Record ID generator]: next ID = FirstUserRecordID
```

### Workspace recovery admission

#### Duplicate requests share one workspace recovery

```text
*Client
  -> [Workspace recovery manager]: first request; reserve slot and start -> 503
  -> [Workspace recovery manager]: duplicate request; share attempt -> 503
  - exactly one workspace-recovery start is logged
  -> [Workspace recovery manager]: later request after completion -> recovered value
```

#### Recovery limit rejects without queueing and permits a later retry

```text
*Client
  -> [Workspace recovery manager]: workspaces 44..46 occupy three of four slots
  -> [Workspace recovery manager]: workspace 42; reserve final slot -> 503
  -> [Workspace recovery manager]: workspace 43; no slot -> 503
      - create no state and queue no worker for workspace 43
  -> [Workspace recovery manager]: after workspace 42 releases slot,
                                   a new request starts workspace 43 -> 503
```

#### Partition reset preserves workspace recovery capacity

```text
*Client
  -> [Workspace recovery manager]: workspaces 3, 5, 7, and 9 reserve all four slots
  -> [Command processor]: command failure resets partition state
  -> [Partition recovery manager]: recover replacement partition
      - reuse the partition key's existing workspace-recovery slot pool
  -> [Workspace recovery manager]: workspace 11; old-generation workers still occupy all slots -> 503
      - create no state and queue no worker for workspace 11
  -> [Workspace recovery manager]: one detached worker finishes and releases its shared slot
  -> [Workspace recovery manager]: workspace 11; reserve returned slot and start -> 503
```

#### Failed workspace recovery is reported and retried once

```text
*Client
  -> [Workspace recovery manager]: start attempt -> 503
      -> [Application events]: WLog read fails; retain failure
  -> [Workspace recovery manager]: next request gets retained failure;
                                   reserve slot and start one retry -> 500
  -> [Workspace recovery manager]: request while retry runs -> 503
  -> [Workspace recovery manager]: request after completion -> recovered value
```

## Sequence diagrams

### Request-driven partition then workspace recovery

```mermaid
sequenceDiagram
    participant Client as *Client
    participant CP as [Command processor]
    participant Part as [Partition recovery manager]
    participant WS as [Workspace recovery manager]

    Client->>CP: Command for deployed but unrecovered partition/workspace
    CP->>Part: getOrStart(partition)
    Part-->>CP: 503; recover partition asynchronously
    CP-->>Client: 503
    Client->>CP: Retry after partition recovery
    CP->>Part: recovered partition
    CP->>WS: getOrStart(workspace)
    WS-->>CP: 503; recover workspace asynchronously
    CP-->>Client: 503
    Client->>CP: Retry after workspace recovery
    WS-->>CP: recovered workspace
    CP-->>Client: Command result
```

## Cross-cutting concerns

### Failure isolation and shutdown

- Every scheduled `[Checkpoint initial recovery]` partition attempt owns its
  read and writes; its failure is returned in the partition-error map, excludes
  its ID from deployment, and does not cancel sibling attempts. Failures in
  non-cluster applications do not prevent readiness; an unavailable cluster
  partition stops bootstrap. A failed reconstruction writes no partition
  completion marker.
- Every `[Bootstrap operator]` invocation deploys only IDs returned without a
  fatal initial-recovery error. No excluded ID has runtime actualizers or
  schedulers; an empty ID slice starts no partition workers for that application.
- Every command to an excluded partition fails at `[App partitions engine]`
  borrowing with the existing not-found error, mapped to `500` before runtime
  recovery starts. Commands never retry initial recovery or deploy a partition.
- Every bootstrap-context cancellation stops scheduling, cancels in-flight
  initialization, waits for started work, and is returned to `[Bootstrap operator]`.
- Every request-recovery attempt uses the VVM service context rather than request
  cancellation. Command-service shutdown joins the shared partition/workspace
  worker wait group before clearing partition state, including workers belonging
  to partitions detached by reset.
- Every recovery-manager key has at most one current attempt; reset may detach
  an attempt, whose later stale completion cannot overwrite replacement state.

### Observability

- Every request-driven partition log uses `vapp=sys/voedger`,
  `extension=sys._Recovery`, and `partid`; workspace logs add `wsid`.
- Every request admission or retained failure is emitted through `cp.error` with
  the affected partition or workspace in the message.
- Every malformed checkpoint is logged before its single `FirstOffset` fallback;
  read, reapply, start, and completion stages follow
  [Logging](./logging--td.md#command-processor).
- Every initial-recovery error map entry is logged once by `[Bootstrap operator]`
  through `bootstrap.initial_recovery.error`, identifying the application and
  partition. The initializer returns partition errors without logging them again.
  Each error identifies a partition-checkpoint-read, PLog-read,
  workspace-checkpoint-write, or partition-checkpoint-write failure.
  The application completion log is emitted only after every scheduled attempt
  has finished.
- Every undeployed-partition rejection uses the standard not-found message
  through `cp.error`; the initial-recovery cause remains only in the attempt log.

### Performance and persistence

- Every application's initial recovery has at most the `runtime.GOMAXPROCS(0)`
  value sampled for that `InitializeCheckpoints` call in flight; applications are initialized
  in bootstrap order.
- Every partition admits at most four concurrent workspace recovery attempts
  across both its current state and any state detached by reset, and never queues
  rejected work.
- Every nonzero recovery checkpoint is an inclusive scan start; correctness
  requires a log tail scan through the actual log end. Many non-ID-bearing WLog
  events after the checkpoint therefore increase workspace recovery time without
  changing record-ID state.
- Every checkpoint-storage write is last-write-wins. Initial recovery establishes
  missing completion markers; the asynchronous projector advances checkpoints;
  request-driven recovery never writes them.

### Testing

- Feature scenarios are traced to `recovery: scn:` tests in
  [VIT recovery tests](../../../../pkg/sys/it/impl_recovery_test.go), with mocked
  command-processor coverage only for empty logs and lifecycle control that VIT
  cannot construct deterministically.
- The empty-partition case in "Missing checkpoints are computed for every
  declared partition" is covered by
  [initial-recovery tests](../../../../pkg/checkpoints/initial_recovery_test.go).
  VIT provisioning creates both PLog and WLog events; those tests use an
  unprovisioned mock partition rather than deleting a provisioned partition's PLog.
- Initial-recovery deployable-ID/error-map outcomes, ordering, failure isolation,
  and cancellation are covered by
  [initial-recovery tests](../../../../pkg/checkpoints/initial_recovery_test.go). Bootstrap logs each returned
  error once and passes the deployable IDs directly to partition deployment;
  this is covered by [bootstrap tests](../../../../pkg/sys/it/impl_bootstrap_test.go).
- Selective deployment and the standard not-found error are covered by
  [app-partitions tests](../../../../pkg/appparts/impl_test.go) and
  [command tests](../../../../pkg/processors/command/impl_test.go). VIT scenarios
  cover ready servers with one or all user-app partitions excluded, configured
  projectors/jobs staying stopped, healthy commands, repeated `500` responses,
  and successful initialization on a later bootstrap.
- `TestBootstrapCheckpointFailuresPreserveDeclaredPartitionCount` verifies the
  deployment request, local registration, and persisted descriptor on first and
  repeated `Bootstrap` calls with one or all non-cluster partitions excluded.
  Repeated calls use fresh local partition managers against the running VIT;
  real VVM restarts and public routing are covered by the VIT recovery scenarios.
- Routing regression tests create a record before restart in a real WSID that
  maps to partition `1` with count `3` and partition `0` with count `2`. This
  represents logical workspace `10` without API v1's pseudo-workspace translation.
  After restart, the command fails for partition `1`, attempts no PLog/WLog
  writes in any partition, and leaves the record active. A separate descriptor
  mismatch test verifies `409 Conflict` and an unchanged persisted descriptor.
- Four-slot saturation and later retry are exercised through the command
  service. Zero-capacity behavior remains generic-manager technical coverage;
  it is not exposed as a production configuration. The admission test follows
  the four-slot feature's scenario identity and setup steps.
- Partition-reset coverage holds four old-generation workspace attempts open,
  recovers replacement partition state, verifies that a fifth attempt is still
  rejected, and then verifies that completion of a detached attempt returns one
  shared slot to the replacement generation.
- Generic deduplication, limits, retry, reset, stale completion, context, and
  own-worker shutdown behavior is exercised by
  [recovery-manager tests](../../../../pkg/processors/command/recovery_manager_test.go).
  `TestServiceShutdownWaitsForWorkspaceRecovery` holds a canceled worker before
  exit and verifies the service waits, both for a current partition and for one
  detached by reset.
