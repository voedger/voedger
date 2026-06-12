# Context subsystem architecture: prod/apps/processing

Apps processing subsystem architecture for the command, query (v1 and v2), sync and async actualizer, scheduler, BLOB, and N10n pipelines that authenticate the caller, authorize against the workspace ACL, invoke extensions on application partitions, and return a response. Context-level overview: [arch.md](./arch.md).

## External actors

Systems:

- `*Client`
  - External caller invoking commands, queries, BLOB transfers, and N10n subscriptions over Voedger HTTP APIs.

## Scenarios overview

- **`Handle command`**
  - Command pipeline authenticates the caller, authorizes the command and CUDs, executes the extension, writes the PLog event and CUDs, runs sync actualizers, sends the response, and notifies async actualizers.

- **`Handle query`**
  - Query pipeline (v1 or v2) authenticates the caller, authorizes the query, invokes the query extension, and streams result rows to the responder.

- **`Run async actualizer`**
  - Async actualizer watches the PLog notification channel, reads new events, and invokes each triggered projector extension on its partition.

- **`Run scheduler job`**
  - Scheduler waits for the next cron tick, borrows the partition, and invokes the job extension.

- **`Handle BLOB transfer`**
  - BLOB processor authenticates the caller, authorizes the operation, registers or looks up BLOB metadata via a paired command on the cluster app, streams BLOB bytes to or from storage, and returns the result to the responder.

- **`Deliver N10n notification`**
  - The command pipeline updates the N10n broker with the new PLog offset; subscribers (async actualizers and SSE clients) receive the update on their channels.

## Components

### Layers

```text
External actors
    |
    +-- *Client
    |
    v
Entry points
    |
    +-- [Command processor]
    +-- [Query v1 processor]
    +-- [Query v2 processor]
    +-- [BLOB processor]
    +-- [Async actualizer]
    +-- [Scheduler]
    |
    v
In-pipeline operators
    |
    +-- [Sync actualizer]
    +-- [Authenticator call]
    +-- [ACL check]
    +-- [Legacy ACL fallback]
    |
    v
Shared infrastructure
    |
    +-- [App partitions engine]
    +-- [Bus responder]
    +-- [N10n broker]
    |
    v
State
    |
    +-- [(PLog)]
    +-- [(WLog)]
    +-- [(Records)]
    +-- [(Views)]
    +-- [(BLOB storage)]
```

### Entry points

- `[Command processor]`
  - Sync pipeline that borrows the partition, applies rate limits, authenticates, authorizes the command and parsed CUDs, executes the extension, runs the sync actualizer branch, validates and writes the PLog event and CUDs, and emits an N10n update on success.
  - Path to file: [pkg/processors/command/provide.go](../../../../pkg/processors/command/provide.go)

- `[Query v1 processor]`
  - Sync pipeline behind the API v1 query route that borrows the partition, authorizes the query, invokes the query extension, and streams rows to the responder via a rows processor.
  - Path to file: [pkg/processors/query/impl.go](../../../../pkg/processors/query/impl.go)

- `[Query v2 processor]`
  - API v2 sync pipeline that dispatches by API path (`Queries`, `Views`, `Docs`, …) to the matching handler before invoking the extension and streaming rows.
  - Path to file: [pkg/processors/query2/impl.go](../../../../pkg/processors/query2/impl.go)

- `[BLOB processor]`
  - Pipeline that registers or reads BLOB metadata through a paired command on the cluster app and streams BLOB bytes between client and BLOB storage via a `bus.IRequestSender`.
  - Path to file: [pkg/processors/blobber/impl.go](../../../../pkg/processors/blobber/impl.go)

- `[Async actualizer]`
  - Per-projector goroutine triggered by `[N10n broker]` notifications: reads new `[(PLog)]` events and invokes the projector extension via `IAppPartition.Invoke`.
  - Path to file: [pkg/processors/actualizers/async.go](../../../../pkg/processors/actualizers/async.go)

- `[Scheduler]`
  - Per-job goroutine triggered by a cron tick: waits on the schedule, then invokes the job extension after `WaitForBorrow` on the partition.
  - Path to file: [pkg/processors/schedulers/impl_scheduler.go](../../../../pkg/processors/schedulers/impl_scheduler.go)

### In-pipeline operators

- `[Sync actualizer]`
  - Per-partition operator invoked from `[Command processor]` via `IAppPartition.DoSyncActualizer`; forks across registered sync projectors and applies their intents inside the command transaction.
  - Path to file: [pkg/processors/actualizers/impl.go](../../../../pkg/processors/actualizers/impl.go)

- `[Authenticator call]`
  - Calls `iauthnz.IAuthenticator.Authenticate` with the request token, returning principals stored on the workpiece; identity and token policy are owned by the `auth` context (see [../auth/arch.md](../auth/arch.md)).
  - Path to file: [pkg/iauthnz/authn-interface.go](../../../../pkg/iauthnz/authn-interface.go)

- `[ACL check]`
  - Per-pipeline operator that calls `IAppPartition.IsOperationAllowed` for the request operation kind (`Execute` for commands and queries, per-CUD kind for command CUDs); maps a deny to `403 Forbidden`.
  - Path to file: [pkg/processors/command/impl.go](../../../../pkg/processors/command/impl.go)

- `[Legacy ACL fallback]`
  - Falls back to `oldacl.IsOperationAllowed` when the new ACL denies, kept as a temporary bridge for Air; logged as `newACL not ok, but oldACL ok` when only the legacy path allows.
  - Path to package: [pkg/processors/oldacl](../../../../pkg/processors/oldacl)

### Shared infrastructure

- `[App partitions engine]`
  - Borrow/invoke surface used by every entry point: `Borrow` / `WaitForBorrow` (`IAppPartitions.Borrow` / `WaitForBorrow`, pool sized to `EnginePoolSize[ProcessorKind]`), `Invoke` (`IAppPartition.Invoke`, resolves the extension and checks `ProcessorKind` compatibility), and `DoSyncActualizer` (`IAppPartition.DoSyncActualizer`, called by `[Command processor]` after writing the PLog event). Cross-subsystem definition: see [arch.md](./arch.md).
  - Path to file: [pkg/appparts/interface.go](../../../../pkg/appparts/interface.go)

- `[Bus responder]`
  - `bus.IRequestSender` / `IResponder` carry the response (single payload for commands, streamed rows for queries) back to the caller through a per-request channel.
  - Path to file: [pkg/bus/interface.go](../../../../pkg/bus/interface.go)

- `[N10n broker]`
  - In-memory broker that fans out projection-key updates to subscriber channels: `[Command processor]` publishes the new PLog offset, `[Async actualizer]` and SSE subscribers consume it.
  - Path to file: [pkg/in10nmem/impl.go](../../../../pkg/in10nmem/impl.go)

### State

- `[(PLog)]`
  - Partition-scoped event log appended by `[Command processor]` and read by `[Async actualizer]`.
  - Path to file: [pkg/istructs/interface.go](../../../../pkg/istructs/interface.go)

- `[(WLog)]`
  - Workspace-scoped event log appended by `[Command processor]`.
  - Path to file: [pkg/istructs/interface.go](../../../../pkg/istructs/interface.go)

- `[(Records)]`
  - Record storage updated by `[Command processor]` CUDs and read by `[Query v1 processor]` / `[Query v2 processor]`.
  - Path to file: [pkg/istructs/interface.go](../../../../pkg/istructs/interface.go)

- `[(Views)]`
  - Projection view storage updated by `[Sync actualizer]` and `[Async actualizer]` intents.
  - Path to file: [pkg/istructs/interface.go](../../../../pkg/istructs/interface.go)

- `[(BLOB storage)]`
  - BLOB byte storage targeted by `[BLOB processor]`.
  - Path to package: [pkg/iblobstorage](../../../../pkg/iblobstorage)

## Scenarios

### Handle command

```text
*Client
  -> [Command processor]
  -> [App partitions engine].Borrow(Command)
  -> [Authenticator call]
  -> [ACL check] (+ [Legacy ACL fallback])
  -> [App partitions engine].Invoke(command)
  -> append [(PLog)], store [(Records)] CUDs, append [(WLog)]
  -> [App partitions engine].DoSyncActualizer -> [Sync actualizer] -> apply [(Views)] intents
  -> response via [Bus responder]
  -> [N10n broker].Update(PLog offset)
```

### Handle query

```text
*Client
  -> [Query v1 processor] | [Query v2 processor]
  -> [App partitions engine].Borrow(Query)
  -> [Authenticator call]
  -> [ACL check] (+ [Legacy ACL fallback])
  -> [App partitions engine].Invoke(query/view/doc handler)
  -> stream rows via [Bus responder]
```

### Run async actualizer

```text
[N10n broker]: Update(PLog offset)
  -> [Async actualizer] wakes on its channel
  -> reads new [(PLog)] events
  -> [App partitions engine].Borrow(Actualizer)
  -> [App partitions engine].Invoke(projector) per triggered event
  -> apply intents to [(Views)] (and emit further N10n updates)
```

### Run scheduler job

```text
[Scheduler] cron tick
  -> [App partitions engine].WaitForBorrow(Scheduler)
  -> [App partitions engine].Invoke(job)
  -> apply intents to [(Views)] / [(Records)]
```

### Handle BLOB transfer

```text
*Client
  -> [BLOB processor]
  -> [Authenticator call]
  -> [ACL check] (+ [Legacy ACL fallback])
  -> paired command on sys.cluster (register/lookup BLOB metadata)
       -> [Bus responder] dispatches internal request
       -> [Command processor] handles the paired command
  -> stream bytes to/from [(BLOB storage)]
  -> response via [Bus responder]
```

### Deliver N10n notification

```text
[Command processor] commits a command successfully
  -> [N10n broker].Update(ProjectionKey{App, PLogUpdates, WS=partitionID}, offset)
  -> [Async actualizer] channel wakes (see Run async actualizer)
  -> SSE subscribers receive the update through [N10n broker].WatchChannel
```
