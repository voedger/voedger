# Technical design: logging

## Overview

The logging subsystem provides structured, context-aware logging with automatic attribute propagation through the request processing pipeline. It uses Go's standard `log/slog` package as the underlying engine while maintaining custom log levels and adding context-based attribute management.

## Concepts

### Testing

Tests that ensure that certain attributes and messages are logged in basic scenarios should be implemented

### Logging Attribute

A key-value pair that provides additional context to log entries. Attributes are stored in `context.Context` and propagate with the context through the request processing pipeline using a linked-list chain structure.

**Implementation details:**

- Attributes are stored in a linked-list chain (`logAttrs` struct) attached to context
- Later calls shadow earlier ones for the same key (newest-first lookup)
- O(1) attribute addition via `logger.WithContextAttrs()`
- Attributes are extracted and appended to log entries automatically by `*Ctx` logging functions

**Standard attributes:**

- **feat** (string): Feature name within the application
  - Constant: `logger.LogAttr_Feat`
  - Example: "magicmenu"
  - Set by: Application-specific handlers
  - Purpose: Track feature-level activity

- **reqid** (string): Unique request identifier
  - Constant: `logger.LogAttr_ReqID`
  - Purpose: Trace single request through all processing stages
  - Set by: Router using global atomic counter
  - Format: "{Server start time (MMDDHHmm)}-{atomicCounter}"
  - Example: "03141504-42"

- **vapp** (string): Voedger application qualified name
  - Purpose: Identify which application is processing the request
  - Set by: Processing initiator
    - Router at request entry point: `sys/registry`, `untill/fiscalcloud`
    - Voedger on bootstrapping: `sys/voedger`

- **wsid** (int): Workspace ID
  - Constant: `logger.LogAttr_WSID`
  - Example: 1001
  - Set by: Router from validated request data, actualizers when processing events
  - Purpose: Filter logs by workspace for multi-tenant debugging

- **extension** (string): Extension or function being executed
  - Constant: `logger.LogAttr_Extension`
  - Example: `c.sys.UploadBLOBHelper`, `q.sys.Collection`, `sys._Docs`, `sys._CP`
    - `sys._Docs`: API v2, working with documents
  - Purpose: Identify specific command/query/function in logs
  - Set by: Processing initiator
    - Router: based on request resource/QName
    - Actualizer: based on event QName

- **stage** (string): Processing stage within the component that emits the log entry
  - Constant: `logger.LogAttr_Stage`
  - Purpose: Categorize log entries by processing phase, enabling filtering and inter-stage latency measurement across components
  - Set by: caller, provided as the `stage` parameter of `logger.*Ctx()` funcs

## General scenarios

- App enriches request context with logging attributes (vapp, reqid, wsid, extension) using `logger.WithContextAttrs()`
- App calls context-aware logging functions with `context`, `stage`, and message `args` as parameters
  - Context attributes (vapp, reqid, wsid, extension, etc.) are automatically extracted and added to the log entry
  - `stage` value is appended to the log as `stage` attribute
  - Message args are formatted via `fmt.Sprint()` and used as the log message
- Affected parts of both APIv1 and APIv2 must have the same logging, except when an attribute is not applicable in one of the API versions — in that case the attribute is set to the constant string `<not applicable in APIv1>`
- If there is already some logging in the affected source code files that is not described in this document then this logging must be dropped

### Built-in extension panic diagnostics

- The [built-in extension engine](../../../../pkg/iextengine/builtin/impl.go) recovers an extension panic before its stack unwinds and returns one error containing `extension <full QName> panic: <panic value>`, followed by the current goroutine stack
- The extension engine does not emit a separate log record; app partitions propagate the enriched error to the existing processor-specific error handlers
- Each existing processor error handler logs the enriched error in one call, keeping the extension identity, panic value, stack, stage, and context attributes in the same correlated log record
- WASM guest-runtime panic extraction and formatting remain unchanged

---

## Per-component logging

### Server core events

### HTTP

HTTP root context is derived from VVM context:

- `vapp="sys/voedger"`
- `extension` = server name: `sys._HTTPServer`, `sys._AdminHTTPServer`, `sys._HTTPSServer`, or `sys._ACMEServer`
  Used for logging server start/stop operations and for all incoming HTTP requests:

- Router params validation failure: level `Error`, stage `routing.validation`, msg `<error message>`
- Start accepting connections success: level `Info`, stage `endpoint.listen.start`, msg `<addr>:<port>`
- Start accepting connections failure: level `Error`, stage `endpoint.listen.error`, msg `<error message>`
- Server stops accepting connections: level `Info`, stage `endpoint.shutdown`, msg (empty)
- Error on http server shutdown: level `Error`, stage `endpoint.shutdown.error`, msg `<error message>`
- Server exits unexpectedly: level `Error`, stage `endpoint.unexpectedstop`, msg `<err>`
- http server internal error log: level `Error`, stage `endpoint.http.error`, msg `<trimmed line emitted to http.Server.ErrorLog>`
  - `http.Server.ErrorLog` is bridged to voedger logger via `logger.NewStdErrorLogBridge(s.rootLogCtx, "endpoint.http.error", logger.WithFilter(skipAnnoyingErrors...))`
  - Attributes are inherited from the http server root log context (`vapp="sys/voedger"`, `extension=<server name>`)
  - Lines matching any substring in `skipAnnoyingErrors` (currently `"TLS handshake error"`) are suppressed by `logger.WithFilter`

#### Application deployment

`btstrp.Bootstrap()` is called. Uses `vapp="sys/voedger"`, `extension="sys._Bootstrap"` attribs.

- Bootstrap starts: level `Info`, stage `bootstrap`, msg `started`
- Cluster app workspace initialized: level `Info`, stage `bootstrap`, msg `cluster app workspace initialized`
- For each built-in app: level `Info`, stage `bootstrap.appdeploy.builtin`, msg `<appQName>`
- For each sidecar app: level `Info`, stage `bootstrap.appdeploy.sidecar`, msg `<appQName>`
- For each built-in app partition: level `Info`, stage `bootstrap.apppartdeploy.builtin`, msg `<appQName>/<partID>`
- For each sidecar app partition: level `Info`, stage `bootstrap.apppartdeploy.sidecar`, msg `<appQName>/<partID>`
- Bootstrap completes: level `Info`, stage `bootstrap`, msg `completed`
- On app deployment failure: panics with `failed to deploy app <appName>: <error>` (no logging)

**Initial recovery:**

Initial recovery derives its context from bootstrap and replaces `extension`
with `sys._Recovery`. Every entry uses `vapp=sys/voedger` and the
component-local `app=<appQName>` attribute. Every per-partition entry also uses
`partid=<partitionID>`; a failed workspace-checkpoint write additionally uses
`wsid=<workspaceID>`.

- Application initialization starts after application structures are available
  and before partition attempts are scheduled: level `Info`, stage
  `bootstrap.initial_recovery.start`, msg
  `partitions <numPartitions>, workers <workerLimit>`
- When the saved projector position is missing, `0`, or `FirstOffset` (`1`),
  initial recovery logs once immediately after a successful position write and
  before writing the partition completion marker. Level `Info`, stage
  `bootstrap.initial_recovery`, msg `ap.sys.ApplyRecoveryCheckpoint offset set to %d`,
  where `%d` is the newly stored PLog offset
- A missing partition is initialized successfully: level `Info`, stage
  `bootstrap.initial_recovery.partition.complete`, msg
  `lastPLogOffset <offset>, workspaceCheckpoints <count>`
- Bootstrap alone emits one entry per partition-error map entry returned by
  `IInitialRecovery.InitializeCheckpoints`: level `Error`, stage
  `bootstrap.initial_recovery.error`, msg `<operation>: <error message>`, where
  `<operation>` is one of `get partition checkpoint`, `read PLog`,
  `put workspace checkpoint`, `read projector position`, `write projector position`,
  or `put partition checkpoint`
- Application initialization completes after every scheduled attempt has
  finished: level `Info`, stage `bootstrap.initial_recovery.complete`, msg
  `initialized <count>, skipped <count>, failed <count>`

The bootstrap failure entry uses `app=<appQName>` for the current application
and `partid=<partitionID>` from the map key. The initializer returns
operation-wrapped errors without logging them; failed attempts emit no
partition-completion entry. The message preserves both the failed operation and
its underlying cause, which remain in bootstrap logs rather than being retained
in app partitions or included in command errors. Such an error is isolated to
the partition: a failed reconstruction writes no partition completion marker,
the failed partition is excluded from deployment, and sibling attempts continue.
The error map is returned for logging rather than as a fatal initialization
error. For non-cluster applications these entries do not prevent the
application-completion log or server readiness. Bootstrap separately requires
the cluster application's sole partition: if it has no eligible ID or fails to
deploy, bootstrap returns an error and the server does not become ready.
Bootstrap-context cancellation takes precedence over classifying an
operation error as partition-scoped; cancellation and application setup failure
before partition scheduling are returned to bootstrap, and the
application-completion entry is not emitted.

#### Leadership acquisition

Uses `vapp="sys/voedger"`, `extension="sys._Leadership"`, `key` attribs.

- When elections are finalized: level `Warning`, stage `leadership.acquire.finalized`, msg `elections cleaned up; cannot acquire leadership`
- On each attempt when another node holds leadership: level `Info`, stage `leadership.acquire.other`, msg `leadership already acquired by someone else`
- On storage error: level `Error`, stage `leadership.acquire.error`, msg `InsertIfNotExist failed: <err>`
- On acquire success: level `Info`, stage `leadership.acquire.success`, msg `success`

#### Leadership maintenance

Uses `vapp="sys/voedger"`, `extension="sys._Leadership"`, `key` attribs.

- First 10 renewal ticks: level `Verbose`, stage `leadership.maintain.10`, msg `renewing leadership`
- Every 200 ticks after initial 10: level `Verbose`, stage `leadership.maintain.200`, msg `still leader for <duration>`
- On transient storage error (retried every second within the interval): level `Error`, stage `leadership.maintain.stgerror`, msg `compareAndSwap error: <err>`
- On leadership stolen: level `Error`, stage `leadership.maintain.stolen`, msg `compareAndSwap !ok => release`
- On all retries exhausted within interval: level `Error`, stage `leadership.maintain.release`, msg `retry deadline reached, releasing. Last error: <err>`
- On error after `processKillThreshold` (TTL/4), before `os.Exit(1)`: level `Error`, stage `leadership.maintain.terminating`, msg `the process is still alive after the time allotted for graceful shutdown -> terminating...`

#### Leadership release

Uses `vapp="sys/voedger"`, `extension="sys._Leadership"`, `key` attribs.

- When `ReleaseLeadership` is called but this node is not the leader: level `Warning`, stage `leadership.release.notleader`, msg `we're not the leader`
- On successful release: level `Info`, stage `leadership.released`, msg (empty)

---

### Router

**Request handling:**

- Validates request data (app, workspace, resource)
- Generates unique request ID: `fmt.Sprintf("%s-%d", globalServerStartTime, reqID.Add(1))`
  - `globalServerStartTime` format: "MMDDHHmm" (e.g., "03061504" for March 6, 15:04)
  - `reqID` is atomic counter incremented per request
- Creates request context with attributes:
  - `vapp`: Application QName from validated data
  - `reqid`: Generated request ID
  - `wsid`: Workspace ID from validated data
  - `extension`: Resource name (API v1) or QName/API path (API v2)
  - `origin`: HTTP Origin header value
  - `headers`: all request headers formatted as a single string for production debugging of real IP propagation
- Request received: level `Verbose`, stage `routing.accepted`, msg (empty)
- Per-workspace query limit reached: level `Warning`, stage `routing.qp.limit`, msg `droppedInLast10Seconds=<count>`
  - Dropped queries are aggregated per [wsid, extension] key
  - On each rejection: bump counter for the key, store request context from the last dropped query
  - On every query request: check `lastLoggedAt` timestamp under mutex; if 10 seconds have elapsed, swap the rejections map, log one warning per key with non-zero count, update `lastLoggedAt`
  - On server shutdown: `flushAll()` logs and purges all pending entries regardless of elapsed time
- First response from bus (immediately after `SendRequest` returns): level `Verbose`, stage `routing.latency1`, msg `<latency_ms>`
- Error sending request to VVM: level `Error`, stage `routing.send2vvm.error`, msg `<error message>` or `forwarding <source resource> to <target resource> failed: <error message>` for the VSqlUpdate shim
- Error sending response to client: level `Error`, stage `routing.response.error`, msg `<error message>`

**VSqlUpdate shim** (both API v1 and API v2, emitted by `dispatchVSqlUpdateShim_V1/V2`):

The router reroutes legacy `c.cluster.VSqlUpdate` command requests to the `q.cluster.VSqlUpdate2` query to avoid a command processor deadlock. Shim-specific log entries share the `routing.vsqlupdate*` subsystem so `stage=routing.vsqlupdate*` filters the whole shim feed.

- Reroute announcement (emitted once the shim is committed to forwarding the request - in `_V1` right before request rewriting, in `_V2` after the body / args preflight succeeds): level `Verbose`, stage `routing.vsqlupdate`, msg `rerouting c.cluster.VSqlUpdate to q.cluster.VSqlUpdate2`
- Offset reporting (emitted on success after the query response has been captured): level `Verbose`, stage `routing.vsqlupdate`, msg `LogWLogOffset=<log> (to be sent to the client as CurrentWLogOffset), CUDWLogOffset=<cud>`
- Shim reply failure (captured status is not 200 OK or downstream `respErr` is non-nil, logged right before the reply is flushed to the client): level `Error`, stage `routing.vsqlupdate.error`, msg `c.cluster.VSqlUpdate shim reply failed: status=<status> respErr=<err> body=<captured body>`

APIv2 preflight failures (malformed JSON body, missing `args` object, args marshal error, downstream `SendRequest` error) do not emit a log line - the shim returns a fall-through signal so the request is replayed against the standard command processor which owns the error reporting.

---

### VVM Request Handler

Receives requests from the Router and submits them to processors via procbus. The context with attributes is received from Router.

Logging on submit failure is controlled by `VVMConfig.BusyProcessorLogMode`:

- `BusyProcessorLogMode_Error` (default): log error on submit failure
- `BusyProcessorLogMode_Silent`: skip logging, only reply 503

Log entries (when `BusyProcessorLogMode_Error`):

- Failed to submit to query processors: level `Error`, stage `vvm.submit`, msg `no query processors <v1|v2> available`
- Failed to submit to command processors: level `Error`, stage `vvm.submit`, msg `no command processors available, partition <partitionID>`

---

### Command Processor

**Request processing:**

The context with attributes is received from Router. Command error logging
preserves that request context and sets `vapp=<appQName>` and
`partid=<partitionID>` from the command message, including when borrowing the
application partition fails.

- Logs the event details right after successful write to PLog: calls `processors.LogEventAndCUDs()` (see [LogEventAndCUDs](#logeventandcuds)) with stage `cp.plog_saved`; per-CUD callback always returns `shouldLog=true`; `msgAdds` is `,oldfields={...}` for CUDs that arrived with the HTTP request, empty for CUDs created by the command; `eventMessageAdds` is empty
- Right before sending the response to the bus:
  - Command handling error: level `Error`, stage `cp.error`, msg `<error message>, body: <compacted request body>`. Error-message variants:
    - Missing partition of a registered application (`appparts.ErrNotFound`): `partition not found`; response `500 Internal Server Error`
    - Missing application (`appparts.ErrNotFound`): `application not found`; response `400 Bad Request`
    - Partition recovery in progress: msg `partition <partitionID>: recovery is in progress`
    - Partition recovery admission limit reached: msg `partition <partitionID>: recovery concurrency limit is reached`
    - Partition recovery failed: msg `partition <partitionID>: recovery failed: <error message>`
    - Workspace recovery in progress: msg `workspace <wsid>: recovery is in progress`
    - Workspace recovery admission limit reached: msg `workspace <wsid>: recovery concurrency limit is reached`
    - Workspace recovery failed: msg `workspace <wsid>: recovery failed: <error message>`
  - Command executed successfully: level `Verbose`, stage `cp.success`, msg `<command result>`

The not-found entries use the normal request context received from Router, not
the `sys._Recovery` context. Their concise error text omits application and
partition identifiers because `vapp` and `partid` identify the request. It also
omits the initial-recovery cause. That cause is logged
only by bootstrap as described in [Application deployment](#application-deployment).
Repeated commands to an undeployed partition log the same standard not-found
error and do not emit partition/workspace recovery start, failure, admission, or
retry messages. The runtime recovery logs below apply only to deployed
partitions; see [recovery request handling](./recovery--td.md#recovery-on-command-request)
for the request lifecycle and status policy.

- Additional log on errors:
  - if error happens on any of:
    - sync actualizers run
    - apply records
    - put to WLog or PLog
  - then logs that partition restart is scheduled
    - stage `cp.partition_recovery`
    - level `Warning`
    - `vapp` attrib is replaced with `sys/voedger`
    - `extension` attrib is replaced with `sys._Recovery`
    - msg `partition will be restarted due of an error on <syncActualizers, applyRecords, etc>: <error message>`

**Partition recovery:**

- `vapp` attrib is replaced with `sys/voedger`
- `extension` attrib: `sys._Recovery`
- `partid` attrib: partition ID
- Partition recovery start: level `Info`, stage `cp.partition_recovery.start`, msg (empty)
- Invalid partition checkpoint: level `Error`, stage `cp.partition_recovery.checkpoint.invalid`, msg `stored PLog offset <checkpointOffset> does not identify an existing event; retrying from FirstOffset`
  - emitted when an inclusive read from a present nonzero checkpoint returns no event or its first event offset differs from the stored offset; absent and zero checkpoints start at `FirstOffset` without validation
  - uses the partition recovery context attributes and precedes the fallback read from `FirstOffset`
- Partition recovery complete: level `Info`, stage `cp.partition_recovery.complete`, msg `nextPLogOffset <offset>, workspaces {}, duration <milliseconds>ms`; workspace recovery starts only after the partition is published
- Partition recovery failure, including checkpoint lookup, scan, re-apply, or cancellation: level `Error`, stage `cp.partition_recovery.error`, msg `<error message>, duration <milliseconds>ms`
  - Emitted at attempt completion in addition to any operation-specific error log; cancellation is checked before reporting success
- ReadPLog failure during either the checkpoint scan or the `FirstOffset` fallback scan: level `Error`, stage `cp.partition_recovery.readplog.error`, msg `<error message>`
- Last event re-apply: `processors.LogEventAndCUDs()` called with stage `cp.partition_recovery.reapply` to log which event is being re-applied (with `woffset`, `poffset`, `evqname` attribs); the enriched context is stored in `cmdWorkpiece.logCtx` and used by sync projectors during re-apply
- `LogEventAndCUDs` failure during re-apply: level `Error`, stage `cp.partition_recovery.logeventandcuds.error`, msg `<error message>`
- StoreOp failure (re-apply last event): level `Error`, stage `cp.partition_recovery.storeop.error`, msg `<error message>`

**Workspace recovery:**

- Inherits the partition-recovery attributes `vapp=sys/voedger`, `extension=sys._Recovery`, and `partid=<partitionID>`
- `wsid` attrib: workspace ID
- Workspace recovery start: level `Info`, stage `cp.workspace_recovery.start`, msg (empty)
- Workspace recovery observes cancellation caused by partition reset: level `Info`, stage `cp.workspace_recovery.canceled`, msg `partition reset, duration <milliseconds>ms`
  - Emitted once when the worker exits; uses `vapp=sys/voedger`, `extension=sys._Recovery`, `app=<appQName>`, `partid=<partitionID>`, and `wsid=<workspaceID>`
  - Replaces the success or WLog-read-error entry for that canceled attempt; service shutdown and request cancellation are not reported as partition resets
- Invalid workspace checkpoint: level `Error`, stage `cp.workspace_recovery.checkpoint.invalid`, msg `stored WLog offset <checkpointOffset> does not identify an ID-bearing event; retrying from FirstOffset`
  - emitted when an inclusive read from a present nonzero checkpoint returns no event, its first event offset differs from the stored offset, or the checkpoint event has no newly allocated non-singleton record ID; absent and zero checkpoints start at `FirstOffset` without validation
  - uses the workspace recovery context attributes and precedes the fallback read from `FirstOffset`
- ReadWLog failure during either the checkpoint scan or the `FirstOffset` fallback scan: level `Error`, stage `cp.workspace_recovery.readwlog.error`, msg `<error message>, duration <milliseconds>ms`
- Workspace recovery failure before a WLog scan starts, including checkpoint lookup or cancellation: level `Error`, stage `cp.workspace_recovery.error`, msg `<error message>, duration <milliseconds>ms`
- Workspace recovery complete: level `Info`, stage `cp.workspace_recovery.complete`, msg `nextWLogOffset <offset>, lastRecordID <recordID>, duration <milliseconds>ms`

Partition and workspace recovery completion messages measure elapsed time from entry into the recovery function through its final outcome. The `recoveryDuration` helper appends `, duration <milliseconds>ms` to `msg`, using whole milliseconds, for example `123ms` or `0ms` for a sub-millisecond attempt. This is not a separate structured attribute and does not change bootstrap initial-recovery messages.

---

### Query Processor

**Request processing:**

The context with attributes is received from Router

- query execution error: level `Error`, stage `qp.error`, msg `<error message>`
- Logs success right after `execQuery` returns: level `Verbose`, stage `qp.success`, msg (empty)
- Failed to send error response: level `Error`, stage `qp.error`, msg `"failed to send error: <respondErr>"`

### Sync Projectors

Launched by command processor between `ApplyRecords` and `PutWLog` stages

**Context propagation:**

The enriched context returned by `processors.LogEventAndCUDs()` (with attribs `woffset`, `poffset`, `evqname`) is stored in the `cmdWorkpiece.logCtx` field and used for all sync projector logging:

- `logEventAndCUDs` (called after PLog write) saves the enriched context to `cmdWorkpiece.logCtx`
- During partition recovery, `LogEventAndCUDs()` is called for the re-applied event and its result is also stored in `cmdWorkpiece.logCtx`
- Command processor sync projector handler reads `cmd.logCtx` for `sp.success` and `sp.error` logs
- Each projector branch receives `cmd.logCtx` (via `processors.IProjectorWorkpiece.LogCtx()`) for its per-projector logs

**Command processor logs** (using `cmd.logCtx`):

- After all sync projectors succeed: level `Verbose`, stage `sp.success`, msg (empty)
- Sync projector error: level `Error`, stage `sp.error`, msg `<error message>`

**Each triggered sync projector** (using `LogCtx()`):

The event is already logged by the command processor (`cp.plog_saved`), so there is no separate `logEventAndCUDs` call per projector. The projector uses `processors.IProjectorWorkpiece.LogCtx()` to obtain the enriched context and extends it with `extension`=`<projector QName>`:

- Right before `IAppParts.Invoke()`: level `Verbose`, stage `sp.triggeredby`, msg `<triggered by qname>`, `extension`=`<projector QName>`
- After successful `Invoke()`: level `Verbose`, stage `sp.success`, `extension`=`<projector QName>`, msg (empty)
- On `Invoke()` failure: level `Error`, stage `sp.error`, `extension`=`<projector QName>`, msg `<error message>`

### Async Projectors

Attributes:

- `vapp`, `extension`, and `partid` attributes are set in `PartitionActualizers.startNews()` during app partition deployment and inherited by child log contexts. `extension` is `ap.<projector QName>` and `partid` is the partition ID
- Each event's context retains inherited attributes and adds `wsid` from the event workspace and `woffset`, `poffset`, and `evqname` through [event log context enrichment](#event-log-context)
- Enrichment runs before partition borrowing, workspace checks, and projector invocation, independently of log level and entire log run suppression
- Store the enriched context in the pipeline workpiece and pass it to projector invocation and subsequent logging; never add event argument payloads to it

**Entire log run policy:**

- At each actualizer initialization, including retries, read the persisted projector offset using `ActualizerOffset()`; `istructs.NullOffset` means the next read begins at `istructs.FirstOffset`
- For that initialization, suppress routine per-event verbose logging for events queued by the startup `readPlogToTheEnd()` phase only when the starting offset is `istructs.NullOffset`
- Keep `entireLogRunProgress` as the actualizer's only stored entire log run lifecycle state; derive `entireLogRun` once from the starting offset for each startup read and pass it through all batches to queued workpieces, while notification reads always pass false
- Before the first suppressed startup read, emit one Info entry at `ap.entire_log_run.start` with context tags `vapp`, `extension=ap.<projector QName>`, and `partid`; this stage identifies the quiet entire log run
- After startup reading reaches the end, the actualizer calls `waitForEntireLogRunCompletion()` to wait for the end-of-log marker's one-shot completion signal after successful flushing before watching live notifications. Its `entireLogRunProgress.complete()` emits one Info entry at `ap.entire_log_run.complete` with the same identity tags and nonnegative `duration_ms` measured since `start()`, including retry delays
- Only while an entire log run is pending, enqueue an end-of-log marker after the startup events and wait for its one-shot completion channel or cancellation. Pipeline ordering ensures the marker is handled after all earlier events. The projector closes the channel immediately if those events have already been flushed or the log is empty; otherwise it retains the channel until the next successful flush and then clears it. This forces no extra flush or checkpoint and requires no shared offset tracking
- Each retry of a pending entire log run uses a fresh marker and completion channel. Ordinary projectors and completed entire log runs allocate neither; the projector stores no entire log run logging progress
- Use the inherited actualizer context in `start()` and add only `duration_ms` in `complete()`; keep lifecycle contexts separate from event contexts
- Measure start time and elapsed duration with `BasicAsyncActualizerConfig.Time`, injected from `VVMConfig.Time`; `Prepare()` supplies real time only when the clock is omitted
- Use the existing `logger.InfoCtx` without message arguments for both lifecycle entries; the standard formatter emits `msg=""`, and neither entry includes `args` or `poffset`
- Emit each lifecycle entry once per projector/partition runtime, retaining the start time and completion status across retries; a runtime resuming from a saved nonzero offset emits neither entry
- An empty initial log still emits start and completion; a failed or canceled entire log run emits no completion until a subsequent attempt successfully finishes
- While waiting for entire log run completion, suppress parent cancellation errors for graceful shutdown; propagate pipeline failures for retry and preserve their diagnostic logging
- A projector resuming from a saved nonzero offset uses normal logging during startup catch-up; a retry that still starts from `FirstOffset` uses the entire log run policy again
- Capture `entireLogRun` in each workpiece before enqueueing it and keep it immutable thereafter; the pipeline must not infer the event's logging policy from the runtime's entire log run completion state
- Once startup reading reaches the log end, events queued by notification-driven reads use normal logging, even if there were no initial events; events already queued for an entire log run remain quiet until processed
- During a quiet entire log run, omit the event detail entry at `ap`, CUD detail entries at `ap.log_cud`, and the success entry at `ap.success`; skip their payload serialization as well
- This policy does not suppress error reporting or logging explicitly performed by projector extensions; preserve event selection, invocation, intent application, flushing, and offset semantics

**Event processing:**

- Determines if the projector triggered by the current event via `ProjectorEvent()`
- For a triggered projector defined in the event's workspace, when verbose logging is enabled and the workpiece is not suppressed, calls [LogEventAndCUDs](#logeventandcuds) with stage `ap` and argument logging disabled:
  - `perCUDCallback`: returns `shouldLog=true` for all CUDs when triggered by a function or ODoc/ORecord; for other triggers only CUDs matching the trigger QName; `msgAdds` is empty
  - `eventMessageAdds`: `triggeredby=<QName>`
- The event message is exactly `triggeredby=<QName>`, without an `args` prefix, empty argument placeholder, or leading separator; CUD filtering and field messages remain unchanged outside a quiet entire log run
- Logs success only when the workpiece is not suppressed: level `Verbose`, stage `ap.success`, msg (empty)

**Error handling:**

Done in the actualizer's retrier `OnError` handler:

- For a `pipeline.IErrorPipeline` with an event workpiece, logs level `Error`, stage `ap.error`, msg `<error message>`, using that workpiece's enriched context
- Event errors retain `vapp`, `extension`, `partid`, `wsid`, `woffset`, `poffset`, `evqname`, and other inherited tags even during a quiet entire log run and when verbose logging is disabled; argument payloads are neither serialized nor appended by the actualizer
- Preserve the workpiece's enriched context if detail logging fails; never overwrite it with a nil context returned on an error path
- Errors without an event workpiece use the actualizer's base context with `extension=sys._Actualizer`, level `Error`, stage `actualizer.error`, msg `<error message>`; event-specific tags are not fabricated when no event is available
- Retry, cancellation, and projector error metrics retain their existing behavior

**Verification:**

Extend the [actualizer execution tests](../../../../pkg/processors/actualizers/async_test.go) to exercise the real read, queue, invocation, and retry paths with captured logs. Confirm projector invocation or persisted projection results and saved-offset progress before asserting that per-event details are absent during an entire log run.

| Case                                                                                         | Required evidence                                                                                                                     |
|----------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------|
| New projector processes multiple events from FirstOffset                                       | Processing completes with no `ap`, `ap.log_cud`, or `ap.success` detail entries for processed events                                   |
| Entire log run transitions to notification-driven processing, including an empty initial log | New matching events emit normal trigger, filtered CUD, and success entries without `args`                                             |
| Pipeline processing lags behind startup reading                                              | Queued entire log run events remain quiet and completion waits for their successful flush                                                      |
| Existing projector resumes from a saved offset                                               | Matching catch-up events retain normal verbose details without `args`                                                                 |
| Event processing fails during an entire log run or live processing at Verbose and Info levels           | The same `ap.error` entry carries the expected error and all applicable event context tags, without argument payloads                 |
| Partition borrowing or workspace validation fails before projector invocation                | The `ap.error` entry still carries the failing event's context tags without arguments                                                 |
| Failed processing retries                                                                    | Error diagnostics remain available, processing eventually succeeds, and each initialization follows the persisted-offset policy above |

Retain function-trigger and record-trigger CUD filtering coverage outside a quiet entire log run. Verify that the [command-processing logging tests](../../../../pkg/processors/command/impl_test.go) and partition-recovery logging retain argument payloads and their existing stages.

Map the three [entire log run logging scenarios](logging.feature) to named Go subtests:

- A new projector logs entire log run start and completion
- Entire log run completion waits for pending processing
- An entire log run retry keeps one start and one completion

Assert that start and completion have the specified level and tags with an empty message and no `args`, appear once through live processing and retries at Verbose and Info levels, and are absent for saved-offset resumption. Verify that event-specific tags do not leak into lifecycle contexts and only completion includes duration. Hold the final entire log run event while the reader reaches the end to prove completion waits for processing and verify the saved offset when buffered completion is logged.

Empty-log and cancellation behavior remains covered by existing Go regression tests, without separate Gherkin scenarios.

Verify that a failed entire log run which has already saved an offset retries with normal event logging while retaining one start and one completion entry for the runtime.

### Blob Processor

Note: BLOB ID is a string — either a numeric record ID or a SUUID, depending on whether the BLOB is persistent or temporary.

**Router (both API v1 and API v2):**

API v1 (`blobHTTPRequestHandler_Write`, `blobHTTPRequestHandler_Read`) already calls `withLogAttribs()`. API v2 blob handlers must also be updated to call `withLogAttribs()`, so both paths set identical context attributes and emit `routing.accepted`.

- `extension`:
  - Write: `sys._Blob_Write`
  - Read: `sys._Blob_Read`

**Attributes set by Blob Processor (both read and write):**

Attribute key strings are defined as local constants within the blob processor package.

- `ownerqname` (string): QName of the record that owns the BLOB (e.g. `air.Bill`); set to `<not applicable in APIv1>` for APIv1 requests
- `ownerfield` (string): field name on the owner record that holds the BLOB reference; set to `<not applicable in APIv1>` for APIv1 requests
- `ownerid` (string): record ID of the owner record (read only; write: owner record does not exist yet at this point); set to `<not applicable in APIv1>` for APIv1 requests
- `blobid` (string): BLOB ID (numeric record ID or SUUID):
  - **Read**: added to context at the start of processing (BLOB ID is known from the request URL)
  - **Write**: added to context right after `blobStorage.WriteBLOB()` completes successfully

**Read:**

- Logs success: level `Verbose`, stage `bp.success`, msg (empty)

**Write:**

- After all validation succeeds (`validateQueryParams` completes): level `Verbose`, stage `bp.meta`, msg `name=<name>,contenttype=<type>`
- After `registerBLOB`: level `Verbose`, stage `bp.register.success`, msg (empty)
- After `blobStorage.WriteBLOB()` — adds `blobid` attrib to context, then: level `Verbose`, stage `bp.write.success`, msg (empty)
- After setting BLOB status to `completed`: level `Verbose`, stage `bp.setcompleted.success`, msg (empty)

**Error handling:**

Context may include `blobid`, `ownerqname`, `ownerfield`, `ownerid` attribs.

- Logs error: level `Error`, stage `bp.error`, msg `<error message>`
  - If the error is `SysError` with 400 Bad Request status code then url query and BLOBs-related headers are added to the msg comma-separated

### N10N Processor

**Router (both API v1 and API v2):**

API v1 n10n handlers must be updated to call `withLogAttribs()`, matching API v2 behavior. The base context sets `reqid`, `origin`, and `extension`. The `wsid` and `vapp` attribs are NOT set on the base context — they are set per projection key (see Per-projection logging below):

- Subscribe+Watch (`POST .../notifications`): `extension` = `sys._N10N_SubscribeAndWatch`
  - Requires adding `processors.APIPath_N10N_SubscribeAndWatch` and a new case in `apiPathToExtension()` returning `"sys._N10N_SubscribeAndWatch"`
- Subscribe-extra (`PUT .../subscriptions/{entity}`): `extension` = entity QName (e.g. `air.view_price`)
- Unsubscribe (`DELETE .../subscriptions/{entity}`): `extension` = entity QName (e.g. `air.view_price`)

**Component-local attribute keys** (defined as local constants in the n10n processor package, not in `logger/consts.go`):

- `channelid` (string): N10N channel UUID — added to context after channel creation
- `projection` (string): Projection QName — added per projection to per-projection context

**Per-projection logging:**

Each projection key carries `App`, `Projection`, and `WS`. For success, subscribe error, SSE, and watch-done logging, a **per-projection context** is created from the base context enriched with:

- `vapp` = projection key's `App`
- `wsid` = projection key's `WS`
- `projection` = projection key's `Projection` QName string

**Errors before projection keys are parsed:**

- Logs error: level `Error`, stage `n10n.error`, msg `<error message>,rawkeys=<raw projection keys from request>`
  - The base context is used (`vapp` and `wsid` are empty)
  - Applies to: missing/malformed JSON payload, unmarshal errors

**Errors on individual projection subscribe:**

- Logs error: level `Error`, stage `n10n.subscribe.error`, msg `<error message>`
  - Uses per-projection context with `vapp`, `wsid`, and `projection` attribs specific to the failed projection
  - `channelid` attrib is present if the channel was already created

**Errors on individual projection unsubscribe:**

- Logs error: level `Error`, stage `n10n.unsubscribe.error`, msg `<error message>`
  - Uses per-projection context with `vapp`, `wsid`, and `projection` attribs specific to the failed projection

**Subscribe+Watch flow (APIv1 `subscribeAndWatchHandler` and APIv2 `impl_subscribeandwatch.go`):**

- Adds `channelid` attrib to the base context after channel creation
- After all projections are subscribed successfully, logs **per each projection**: level `Verbose`, stage `n10n.subscribe&watch.success`, msg empty, using per-projection context (with `vapp`, `wsid`, `projection`, `channelid`)
- Logs each SSE message: level `Verbose`, stage `n10n.sse_send.success`, msg `<sse message>`, using per-projection context created from the SSE event's projection key (`vapp`, `wsid`, `projection`, `channelid`)
- Logs error on fail to send SSE message: level `Error`, stage `n10n.sse_send.error`, msg `<error>`, using per-projection context from the SSE event's projection key (`vapp`, `wsid`, `projection`, `channelid`)
- On `WatchChannel` goroutine finish: logs **per each subscribed projection**: level `Verbose`, stage `n10n.watch.done`, msg (empty), using per-projection context (`vapp`, `wsid`, `projection`, `channelid`) — logged in both APIv1 (`serveN10NChannel`) and APIv2 (`watchChannel` goroutine)

**Subscribe-extra flow (APIv1 `subscribeHandler` and APIv2 `impl_subscribeextra.go`):**

- For each projection key, logs: level `Verbose`, stage `n10n.subscribe.success`, msg empty, using per-projection context (`vapp`, `wsid`, `projection`)

**Unsubscribe flow (APIv1 `unSubscribeHandler` and APIv2 `impl_unsubscribe.go`):**

- For each projection key, logs: level `Verbose`, stage `n10n.unsubscribe.success`, msg empty, using per-projection context (`vapp`, `wsid`, `projection`)

**N10N Broker lifecycle (in10nmem):**

A dedicated log context is created inside `NewN10nBroker` with `vapp="sys/voedger"` and `extension="sys._N10NBroker"`. All broker lifecycle log calls use this context.

- Notifier goroutine started: level `Info`, stage `n10n.notifier.start`, msg (empty)
- Notifier goroutine stopped: level `Info`, stage `n10n.notifier.stop`, msg (empty)
- Heartbeat goroutine started: level `Info`, stage `n10n.heartbeat.start`, msg `Heartbeat30Duration: <duration>`
- Heartbeat goroutine stopped: level `Info`, stage `n10n.heartbeat.stop`, msg (empty)
- Channel expired during `WatchChannel`: level `Error`, stage `n10n.channel.expired`, msg `<subjectLogin>`
- Channel cleanup unsubscribe error: level `Error`, stage `n10n.cleanup.error`, attribs `channelid=<id>`, `projection=<projection QName>`, msg `<error>`

## Schedulers

**Attributes:**

- `vapp`, `extension`, and `wsid` attribs are set on app partition deployment on start new actualizers stage. `extension` is `job.<job QName>`, `wsid` is the workspace ID where the job runs

**Job execution:**

- Logs schedule: level `Verbose`, stage `job.schedule`, msg `now=<timeNow>,next=<nextRunTime>`
- Logs wake-up: level `Verbose`, stage `job.wake-up`, msg `<timeNow>`
- Logs successful invoke: level `Verbose`, stage `job.success`, msg (empty)
- Logs invocation error (`runJob` defer): level `Error`, stage `job.error`, msg `<error>`
- Logs retrier error (`Prepare` OnError):
  - If `appparts.ErrNotFound`: level `Error`, stage `job.error`, msg `appparts <error>, will try again`
  - Otherwise: level `Error`, stage `job.error`, msg `<error>`

---

## Key components

### Core logging infrastructure

**[logger package](../../../../pkg/goutils/logger)**

Provides structured logging with context-aware attribute propagation.

- **Files:**
  - `logger.go`: Core logging functions and level management
  - `loggerctx.go`: Context-aware logging functions
  - `consts.go`: Standard attribute constants and slog configuration
  - `types.go`: Internal types for context key and attribute chain
  - `impl.go`: Implementation details (level checking, caller tracking, formatting)

- **Key features:**
  - Hierarchical log levels (Error, Warning, Info, Verbose, Trace)
  - Atomic level checking for thread-safe filtering
  - Automatic caller tracking (function name and line number)
  - Context-based attribute propagation
  - slog integration for structured output

- **Used by:** All request processing components (router, command processor, query processor, actualizers)

### Context management

**[logger.WithContextAttrs](../../../../pkg/goutils/logger/loggerctx.go#L23)**

```go
func WithContextAttrs(ctx context.Context, attrs map[string]any) context.Context
```

Adds logging attributes to context for propagation through call chain.

- **Implementation:**
  - Stores attributes in linked-list chain (`logAttrs` struct)
  - O(1) attribute addition by prepending new node
  - Shadowing support: later calls override earlier ones for same key
  - Thread-safe: immutable chain structure

- **Usage pattern:**

  ```go
  ctx = logger.WithContextAttrs(ctx, map[string]any{
      logger.LogAttr_ReqID: "03061504-42",
      logger.LogAttr_WSID:  1001,
  })
  ```

- **Used by:**
  - Router: initial request context
  - Command processor: event and CUD attributes
  - Actualizers: workspace and event attributes

**[sLogAttrsFromCtx](../../../../pkg/goutils/logger/loggerctx.go#L89)**

Internal function that extracts attributes from context chain.

- Walks linked list from newest to oldest
- First-seen-wins per key (implements shadowing)
- Returns slice of `slog.Any` attributes

### Logging functions

**Context-aware functions ([loggerctx.go](../../../../pkg/goutils/logger/loggerctx.go#L31))**

```go
func VerboseCtx(ctx context.Context, stage string, args ...interface{})
func ErrorCtx(ctx context.Context, stage string, args ...interface{})
func InfoCtx(ctx context.Context, stage string, args ...interface{})
func WarningCtx(ctx context.Context, stage string, args ...interface{})
func TraceCtx(ctx context.Context, stage string, args ...interface{})
func LogCtx(ctx context.Context, skipStackFrames int, level TLogLevel, stage string, args ...interface{})
```

Automatically append context attributes and stage to log entries using slog.

- **Parameters:**
  - `ctx`: Context containing logging attributes (vapp, reqid, wsid, extension, etc.)
  - `stage`: Processing stage that emits the log entry; added to the log as the `stage` slog attribute with key `logger.LogAttr_Stage`. Convention: `<component>.<substage>` (e.g., `"routing"`, `"cp.received"`, `"cp.plog_saved"`)
  - `args`: Message components to be formatted via `fmt.Sprint()`
  - `skipStackFrames` (LogCtx only): Number of stack frames to skip for source location
  - `level` (LogCtx only): Log level for the entry

- **Implementation:**
  - Extracts attributes from context via `sLogAttrsFromCtx()`
  - Adds `stage` parameter as a log attribute with key `logger.LogAttr_Stage`
  - Adds source location (`src` attribute with function:line)
  - Formats message via `fmt.Sprint(args...)`
  - Routes to slogOut (stdout) or slogErr (stderr) based on level
  - Respects global log level via `isEnabled()` check

- **Used by:**
  - Router: request acceptance, error logging
  - Processors: error, success, event/CUD logging
  - Sync Actualizers: error, success logging
  - Async Actualizers: error, success, event/CUD logging

- **Usage example:**

  ```go
  logger.VerboseCtx(ctx, "routing", "request accepted")
  logger.ErrorCtx(ctx, "cp.error", "command failed:", err)
  logger.InfoCtx(ctx, "cp.partition_recovery.complete", "nextPLogOffset ", offset)
  ```

**Standard functions ([logger.go](../../../../pkg/goutils/logger/logger.go#L44))**

```go
func Error(args ...interface{})
func Warning(args ...interface{})
func Info(args ...interface{})
func Verbose(args ...interface{})
func Trace(args ...interface{})
func Log(skipStackFrames int, level TLogLevel, args ...interface{})
```

Non-context-aware logging functions (legacy).

- Used by query processor (opportunity for migration)
- Used by components that don't have request context

### Standard attributes

**[Attribute constants](../../../../pkg/goutils/logger/consts.go#L18)**

```go
const (
    LogAttr_VApp      = "vapp"      // Voedger application QName
    LogAttr_Feat      = "feat"      // Feature name
    LogAttr_ReqID     = "reqid"     // Request ID
    LogAttr_WSID      = "wsid"      // Workspace ID
    LogAttr_Extension = "extension" // Extension/function name
    LogAttr_Stage     = "stage"     // Processing stage name
)
```

Ensures consistent attribute naming across all components.

Component-specific attribute keys used only within a single package (e.g. `blobid`, `ownerqname`, `ownerfield`, `ownerid`, `channelid`, `projectionkey`) are defined as local string constants within their respective packages, not in `logger/consts.go`.

### Event log context

Share event context enrichment in the [processor logging utilities](../../../../pkg/processors/utils.go), separately from message emission:

- Derive `woffset` from `event.WLogOffset()`, `poffset` from the supplied PLog offset, and `evqname` from `event.QName()`
- Preserve inherited attributes using `logger.WithContextAttrs()`
- Do not check the log level, emit messages, read argument objects, or serialize payloads
- Async actualizers use this enrichment independently before processing can fail; the shared detail logger reuses it when emitting verbose details

### LogEventAndCUDs

**[processors.LogEventAndCUDs](../../../../pkg/processors/utils.go)**

Shared utility for logging PLog events and their CUDs with consistent context enrichment. Used by command processor and async actualizers.

- Does nothing and returns `logCtx` unchanged if `!logger.IsVerbose()`
- Uses [event log context enrichment](#event-log-context) for verbose detail entries; callers needing context when verbose logging is disabled enrich it independently
- Accepts an explicit argument-logging choice from the caller, independently of CUD filtering and event message additions
- With argument logging enabled, serializes arguments as before and emits stage `<stage>`, msg `args={...}`, appending `, <eventMessageAdds>` only when additions are nonempty
- With argument logging disabled, never reads or serializes `event.ArgumentObject()` and emits stage `<stage>`, msg `<eventMessageAdds>` without an `args` field or leading separator
- Command processing at `cp.plog_saved` and partition recovery at `cp.partition_recovery.reapply` enable argument logging; async actualizers disable it
- For each CUD:
  - Calls `perCUDLogCallback` to get `shouldLog`, `msgAdds`, `err`
  - If `err != nil`, returns the error
  - If `shouldLog` is false, skips the CUD
  - Enriches context with `rectype`, `recid`, `op`
  - Logs new fields as JSON at Verbose level: stage `<stage>.log_cud`, msg `newfields={...}{msgAdds}`
- On success at verbose level, returns the enriched context for subsequent logging; on failure, async actualizers retain their independently enriched workpiece context for error reporting

**Parameters:**

- `stage`: stage name for the event log entry; CUD entries use `<stage>.log_cud`
- `skipStackFrames`: call-stack frames to skip for the correct `src` attribute
- `perCUDLogCallback`: per-CUD filter; `shouldLog` — whether to log this CUD; `msgAdds` — text appended to the `newfields` message
- `eventMessageAdds`: extra text appended to the pre-CUD event log message (e.g. `triggeredby=<QName>` for actualizers, empty for command processor)
- Argument-logging choice: controls only argument access, serialization, and inclusion in the event message; does not control context enrichment or CUD messages
- `plogOffset`: used as the `poffset` context attribute
- `appDef`: used to marshal CUD field values to JSON
- `event`: source of `woffset`, event arguments, and CUDs

### slog integration

**[Handler configuration](../../../../pkg/goutils/logger/consts.go#L26)**

```go
ctxHandlerOpts = &slog.HandlerOptions{
    Level: slog.LevelDebug,
}
slogOut = slog.New(slog.NewTextHandler(os.Stdout, ctxHandlerOpts))
slogErr = slog.New(slog.NewTextHandler(os.Stderr, ctxHandlerOpts))
```

- slog level is DEBUG already. Actual log level controlling is done by logger's funcs
- Separate handlers for stdout and stderr

### Component integrations

**[Router logging](../../../../pkg/router/utils.go#L143)**

```go
func withLogAttribs(ctx context.Context, data validatedData,
    busRequest bus.Request, req *http.Request) context.Context {
    extension := busRequest.Resource
    if busRequest.IsAPIV2 {
        if busRequest.QName == appdef.NullQName {
            extension = apiPathToExtension(processors.APIPath(busRequest.APIPath))
        } else {
            extension = busRequest.QName.String()
        }
    }
    newReqID := fmt.Sprintf("%s-%d", globalServerStartTime, reqID.Add(1))
    return logger.WithContextAttrs(ctx, map[string]any{
        logger.LogAttr_ReqID:     newReqID,
        logger.LogAttr_WSID:      data.wsid,
        logger.LogAttr_VApp:      data.appQName,
        logger.LogAttr_Extension: extension,
        logAttrib_Origin:         req.Header.Get(httpu.Origin),
        logAttrib_Headers:        formatHeaders(req.Header),
    })
}
```

Creates initial request context with logging attributes.

## Key data models

### logAttrs (internal)

```go
type logAttrs struct {
    attrs  map[string]any
    parent *logAttrs
}
```

Linked-list node for storing logging attributes in context.

- Immutable chain structure for thread safety
- Parent pointer creates chain
- Newest attributes shadow older ones with same key

### ctxKey (internal)

```go
type ctxKey struct{}
```

Unexported context key type for storing `logAttrs` in context.

- Prevents key collisions with other context values
- Type-safe context value access

### TLogLevel

```go
type TLogLevel int32

const (
    LogLevelNone = TLogLevel(iota)
    LogLevelError
    LogLevelWarning
    LogLevelInfo
    LogLevelVerbose
    LogLevelTrace
)
```

Log level enumeration with atomic operations support.
