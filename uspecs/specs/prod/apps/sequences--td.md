# Feature technical design: Sequence allocation

Technical design for the successful allocation behavior specified in
[Sequence allocation](./sequences.feature). Checkpoint production, startup
initialization, and request-driven recovery are outside this feature and are
specified in [Partition and workspace recovery](./recovery--td.md).

## Key concepts

- **Sequence state**: the current in-memory or persisted value from which one
  kind of offset or identifier is issued.
- **Next value**: the value assigned to the next successful operation. PLog and
  WLog state stores the next offset; the ordinary record-ID generator stores
  the next user record ID; `sys.NextBaseWSID` stores the next base WSID.
- **Partition scope**: one application partition owns one PLog-offset sequence.
- **Workspace scope**: one application workspace owns one WLog-offset sequence
  and one ordinary record-ID sequence shared by all ordinary record kinds.
- **Application scope**: one application owns one base-WSID sequence, persisted
  at application WSID 0.
- **Ordinary record ID**: a generated ID beginning at `FirstUserRecordID`
  (200001) and consumed from the workspace's record-ID generator.
- **Singleton ID**: an application-defined ID in the reserved singleton range;
  resolving it does not consume an ordinary record ID.
- **Successful-write advancement**: a PLog or WLog next offset advances only
  after the corresponding log write succeeds. Ordinary record IDs are assigned
  while the PLog event is built. A base WSID is persisted before the command's
  PLog write.

## External actors

Systems:

- `*Client`
  - Submits commands and receives generated identifiers through the application
    request pipeline.

## Components

### Layers

```text
External actors
    |
    +-- *Client
    |
    v
Command sequencing
    |
    +-- [Command processor]
    +-- [Offsets bookkeeping]
    +-- [Record ID generator]
    +-- [WSID generator]
    |
    v
Application persistence
    |
    +-- [Application events]
    +-- [Application records]
    +-- [App partitions engine]
    +-- [(sys.NextBaseWSID)]
```

### Command sequencing

- `[Command processor]`
  - Selects the partition and workspace state, copies their next offsets into
    the raw event, invokes the command, persists and applies the result, and
    returns the generated-ID mapping.
  - impl: [pkg/processors/command/provide.go](../../../../pkg/processors/command/provide.go)
  - impl: [pkg/processors/command/impl.go](../../../../pkg/processors/command/impl.go)

- `[Offsets bookkeeping]`
  - `appPartition.nextPLogOffset` is scoped by application and partition;
    `workspace.NextWLogOffset` is scoped by application and workspace. Both
    start at `FirstOffset` (1). Selection does not advance either value.
  - impl: [pkg/processors/command/types.go](../../../../pkg/processors/command/types.go)
  - impl: [pkg/processors/command/impl.go](../../../../pkg/processors/command/impl.go)

- `[Record ID generator]`
  - One generator per workspace serves every ordinary record kind. `NextID`
    returns the current next value and increments it; `UpdateOnSync(id)` moves
    the next value to `id + 1` only when necessary. Singleton resolution
    bypasses this generator.
  - impl: [pkg/istructsmem/idgenerator.go](../../../../pkg/istructsmem/idgenerator.go)
  - impl: [pkg/istructsmem/internal/singletons/impl.go](../../../../pkg/istructsmem/internal/singletons/impl.go)

- `[WSID generator]`
  - Serializes process-local access to the application's persisted next-base
    counter, uses `FirstBaseUserWSID` (131071) when absent, persists base + 1,
    and returns `(ClusterID << 47) + base`.
  - impl: [pkg/sys/workspace/utils.go](../../../../pkg/sys/workspace/utils.go)
  - impl: [pkg/sys/workspace/impl.go](../../../../pkg/sys/workspace/impl.go)

### Application persistence

- `[Application events]`
  - Regenerates raw IDs and references while building the PLog event, persists
    PLog and WLog events, and enforces the configured Sequence Trust Level for
    ordinary event writes.
  - decl: [pkg/istructs/interface.go](../../../../pkg/istructs/interface.go)
  - impl: [pkg/istructsmem/impl.go](../../../../pkg/istructsmem/impl.go)
  - impl: [pkg/istructsmem/event-types.go](../../../../pkg/istructsmem/event-types.go)

- `[Application records]`
  - Applies event CUDs and enforces the record-write side of the configured
    Sequence Trust Level.
  - impl: [pkg/istructsmem/impl.go](../../../../pkg/istructsmem/impl.go)

- `[App partitions engine]`
  - Invokes the command extension and synchronous projectors on the borrowed
    application partition.
  - decl: [pkg/appparts/interface.go](../../../../pkg/appparts/interface.go)
  - impl: [pkg/appparts/impl.go](../../../../pkg/appparts/impl.go)

- `[(sys.NextBaseWSID)]`
  - Persistent application-level next-base counter stored at `NullWSID` (0),
    independently of command partition and workspace state.
  - decl: [pkg/appdef/sys/sys.go](../../../../pkg/appdef/sys/sys.go)
  - impl: [pkg/sys/workspace/utils.go](../../../../pkg/sys/workspace/utils.go)

## Scenarios

### Log offsets

#### PLog offsets advance within an application partition

```text
*Client
  -> [Command processor]: successful command in workspace Orders, partition 1
      -> [Offsets bookkeeping]: select partition nextPLogOffset = 41
      -> [App partitions engine]: invoke command
      -> [Application events]: persist PLog event 41
      -> [Offsets bookkeeping]: set partition nextPLogOffset = 42
  -> [Command processor]: successful command in workspace Billing, partition 1
      -> [Offsets bookkeeping]: select partition nextPLogOffset = 42
      -> [Application events]: persist PLog event 42
      -> [Offsets bookkeeping]: set partition nextPLogOffset = 43
```

#### WLog offsets advance independently in each workspace

```text
*Client
  -> [Command processor]: successful command in workspace Orders
      -> [Offsets bookkeeping]: select Orders NextWLogOffset = 8
      -> [Application events]: persist Orders WLog event 8
      -> [Offsets bookkeeping]: set Orders NextWLogOffset = 9
  -> [Command processor]: successful command in workspace Billing
      -> [Offsets bookkeeping]: select Billing NextWLogOffset = 13
      -> [Application events]: persist Billing WLog event 13
      -> [Offsets bookkeeping]: set Billing NextWLogOffset = 14
```

### Record IDs

#### A command receives consecutive ordinary record IDs

```text
*Client
  -> [Command processor]: create three ordinary records in workspace Orders
      -> [Application events]: build PLog event
          -> [Record ID generator]: NextID three times -> 200001..200003
      -> [Application events]: persist event and generated-ID mapping
      -> [Application records]: apply three records
  <- [Command processor]: record IDs 200001..200003 in creation order
  -> [Command processor]: create another ordinary record in Orders
      -> [Record ID generator]: NextID -> 200004
  <- [Command processor]: record ID 200004
```

#### Ordinary record-ID sequences are independent between workspaces

```text
*Client
  -> [Command processor]: create an ordinary record in workspace Orders
      -> [Record ID generator]: Orders NextID -> 200001
  -> [Command processor]: create an ordinary record in workspace Billing
      -> [Record ID generator]: Billing NextID -> 200001
```

#### A singleton uses its assigned ID without consuming an ordinary record ID

```text
*Client
  -> [Command processor]: create singleton Configuration in workspace Orders
      -> [Application events]: resolve application-defined singleton ID 65536
      - [Record ID generator] is not called
  <- [Command processor]: singleton stored with ID 65536
  -> [Command processor]: create an ordinary record in Orders
      -> [Record ID generator]: NextID -> 200001
  <- [Command processor]: ordinary record ID 200001
```

### Workspace IDs

#### Workspace-ID sequences are scoped to an application

```text
*Client
  -> [Command processor]: create Orders in application Sales
      -> [App partitions engine]: invoke c.sys.CreateWorkspaceID
          -> [WSID generator]: read Sales [(sys.NextBaseWSID)] or 131071
          -> [(sys.NextBaseWSID)]: persist 131072
  <- [Command processor]: Orders base WSID 131071
  -> [Command processor]: create Billing in application Sales
      -> [WSID generator]: persist Sales next base 131073; return 131072
  -> [Command processor]: create Helpdesk in application Support
      -> [WSID generator]: persist Support next base 131072; return 131071
```

The `sys.WorkspaceID` document created by the command also receives an ordinary
record ID in the command's current workspace. Its PLog and WLog offsets belong
to that current workspace and partition, not to the newly allocated workspace.

## Cross-cutting concerns

### Persistence boundaries and gaps

| Value              | Scope                   | Initial next value           | Advancement boundary                       |
| ------------------ | ----------------------- | ---------------------------- | ------------------------------------------ |
| PLog offset        | Application + partition | `FirstOffset` (1)            | Increment after successful PLog write      |
| WLog offset        | Application + workspace | `FirstOffset` (1)            | Increment after successful WLog write      |
| Ordinary record ID | Application + workspace | `FirstUserRecordID` (200001) | Allocate while building the PLog event     |
| Base WSID          | Application             | `FirstBaseUserWSID` (131071) | Persist base + 1 before returning the WSID |

`VVMConfig.SequencesTrustLevel` selects collision handling for event and record
writes; it does not change allocation scope or select the WSID-counter policy.

| Trust level | PLog/WLog writes    | New table records   | Existing table records |
| ----------- | ------------------- | ------------------- | ---------------------- |
| 0           | `InsertIfNotExists` | `InsertIfNotExists` | `Put`                  |
| 1           | `InsertIfNotExists` | `PutBatch`          | `PutBatch`             |
| 2           | `Put`               | `PutBatch`          | `PutBatch`             |

A conditional-write collision returns `ErrSequencesViolation`; an invalid trust
level panics. `sys.Corrupted` events use `Put` at every level. These edges are
covered by `TestSequencesTrustLevel` in
[write-policy tests](../../../../pkg/istructsmem/impl_seqtrust_test.go).

Failure boundaries intentionally permit gaps between values observed by
successful client calls. Once a PLog event is persisted, its offset and ordinary
record IDs remain consumed even if later record application, synchronous
projection, WLog persistence, or response handling fails. A failed PLog write
does not persist those allocations; partition state is discarded before another
command can reuse it. The WSID counter is committed before the containing PLog
event, so even a PLog failure can leave that base WSID unused. PLog and WLog
counters themselves advance only after their respective writes succeed.
Technical evidence is in
[command processor tests](../../../../pkg/processors/command/impl_test.go),
[event tests](../../../../pkg/istructsmem/event-types_test.go), and
[ID-generator tests](../../../../pkg/istructsmem/idgenerator_test.go).

### Concurrency and consistency

- Every `[Offsets bookkeeping]` value is mutated only within its owning command
  partition/workspace state.
- Every `[WSID generator]` allocation is protected by the process-wide
  `nextWSIDGlobalLock`; coordination across VVM processes is not provided.
- Every `[Application events]` and `[Application records]` ordinary write uses
  the active Sequence Trust Level shown above.

### Testing

- Every scenario under `## Scenarios` is traced to a `sequences: scn:` VIT test
  in [allocation tests](../../../../pkg/sys/it/impl_sequences_test.go).
- Every failure-induced gap or write-collision edge remains technical coverage,
  not a scenario in `sequences.feature`; evidence is the linked command, event,
  ID-generator, and Sequence Trust Level tests.
