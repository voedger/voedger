# Decisions: Improved partition and workspace recovery

## Invariant: the partition checkpoint marks completion of initial recovery

Decision: During initial recovery, check only whether the partition checkpoint exists. If it exists, including a zero-valued checkpoint, skip initial recovery for the entire partition without checking its workspace checkpoints. If it is absent, scan the entire PLog to discover all workspaces and compute any missing checkpoints; existing workspace checkpoints do not establish that the entire partition has completed initial recovery.

Persist the partition checkpoint only after the checkpoints of all workspaces encountered in the PLog have been successfully stored. Consequently, a stored partition checkpoint with missing checkpoints for any encountered workspace is an impossible state under this write ordering. Checking workspace checkpoints when the partition checkpoint exists would require scanning the entire PLog to discover every workspace and would provide no additional completion guarantee.

- Pros: makes the partition checkpoint a sufficient completion marker; avoids a full PLog scan for an already initialized partition.
- Cons: relies on preserving the workspace-before-partition checkpoint write ordering.
- Confidence: user-provided

## Inconsistency: zero workspace checkpoint could not satisfy checkpoint validation

Decision: Treat an absent or zero-valued workspace checkpoint as a direct instruction to scan once from `FirstOffset`, without checkpoint validation, fallback, or invalid-checkpoint logging.

- Pros: makes the zero value a usable initialization sentinel; satisfies the requirement to persist a checkpoint for every encountered workspace; matches the existing zero-checkpoint scenarios and logging expectations.
- Cons: request-driven recovery intentionally treats a present zero workspace checkpoint and an absent workspace checkpoint the same way.
- Confidence: high

Alternatives:

1. Do not persist zero workspace checkpoints
   - Pros: avoids sentinel semantics.
   - Cons: violates the requirement to persist a workspace checkpoint for every encountered WSID.
   - Confidence: low
2. Define special validation rules for present-zero checkpoints
   - Pros: preserves a semantic distinction between present-zero and absent checkpoints during recovery.
   - Cons: adds a separate validation branch while producing the same full scan from `FirstOffset`.
   - Confidence: medium

## Inconsistency: zero partition checkpoint could not satisfy checkpoint validation

Decision: Treat an absent or zero-valued partition checkpoint as a direct instruction to scan once from `FirstOffset`, without checkpoint validation, fallback, or invalid-checkpoint logging.

- Pros: makes zero usable both as the completion marker for an initialized empty partition and as the normal full-scan sentinel; matches the existing zero-checkpoint scenarios and logging expectations.
- Cons: request-driven recovery intentionally treats a present zero partition checkpoint and an absent partition checkpoint the same way.
- Confidence: high

Alternatives:

1. Do not persist zero partition checkpoints
   - Pros: avoids sentinel semantics.
   - Cons: removes the completion marker for an initialized empty partition and causes initial recovery to repeat on every bootstrap.
   - Confidence: low
2. Define special validation rules for present-zero checkpoints
   - Pros: preserves a semantic distinction between present-zero and absent checkpoints during recovery.
   - Cons: adds a separate validation branch while producing the same full scan from `FirstOffset`.
   - Confidence: medium

## Ambiguity: scope of the sequence technical design

Decision: Keep `sequences.feature` focused on successful, externally observable sequence issuance, while allowing `sequences--td.md` to document the write boundaries, Sequence Trust Level policy, and failure-induced gaps needed to explain that issuance. Exclude all checkpoint and recovery behavior from both files.

- Pros: keeps the behavioral specification concise while preserving the technical invariants that would otherwise be lost with `arch-sequences.md`; maintains a clean boundary between sequence issuance and recovery.
- Cons: the technical design intentionally covers failure consequences that are not scenarios in its paired feature file.
- Confidence: high

Alternatives:

1. Restrict both sequence files to successful allocation only
   - Pros: gives "normal behavior" its narrowest interpretation.
   - Cons: leaves important advancement, persistence, and gap invariants undocumented after deleting `arch-sequences.md`.
   - Confidence: medium
2. Include user-visible allocation failures and gaps in both sequence files
   - Pros: provides complete behavioral coverage of sequence issuance failures.
   - Cons: broadens `sequences.feature` beyond the requested normal behavior.
   - Confidence: low

## Ambiguity: whether initial recovery restores runtime sequence state

Decision: Initial recovery only computes and persists missing checkpoints. It does not restore or publish in-memory partition or workspace state. Normal request-driven partition recovery and lazy workspace recovery remain unchanged for deployed partitions; a failed initial-recovery partition remains undeployed and cannot enter request-driven recovery during that server run.

- Pros: matches the requested checkpoint-computation goal; keeps bootstrap bounded to checkpoint initialization; preserves one runtime recovery path and the existing recovery-admission lifecycle.
- Cons: the first request after bootstrap still incurs recovery, and partition recovery scans again from the newly created checkpoint.
- Confidence: high

Alternatives:

1. Eagerly restore every partition and workspace during initial recovery
   - Pros: requests can be admitted immediately after startup.
   - Cons: substantially increases bootstrap work and removes lazy workspace recovery.
   - Confidence: low
2. Restore partition runtime state during initial recovery but leave workspaces lazy
   - Pros: avoids the first partition-recovery response while retaining bounded workspace recovery.
   - Cons: couples checkpoint initialization to command-processor publication and creates two partition-recovery paths.
   - Confidence: medium

## Vagueness: partitions covered by initial recovery

Decision: Treat every partition ID in the application's declared contiguous range `0..NumPartitions-1` as known. Initial recovery checks and, when necessary, initializes every one of those partitions, including partitions with empty PLogs or no allocated log storage.

- Pros: gives every deployable partition an explicit completion marker; aligns initialization with bootstrap's partition deployment set; keeps `IInitialRecovery` count-based.
- Cons: performs checkpoint checks and may write zero-valued markers for partitions that have never received an event.
- Confidence: high

Alternatives:

1. Initialize only partitions for which PLog storage already exists
   - Pros: avoids work for unused partitions.
   - Cons: leaves empty partitions without completion markers and requires a separate stored-PLog enumeration facility.
   - Confidence: low
2. Let bootstrap pass an explicit partition-ID list
   - Pros: supports non-contiguous or selectively deployed partitions.
   - Cons: expands the interface even though current applications deploy a contiguous declared range.
   - Confidence: medium

## Uncertainty: partition scheduling and failure policy during initial recovery

Decision: Process missing partitions with CPU-bounded concurrency and return `([]istructs.PartitionID, map[istructs.PartitionID]error, error)`: successful/already-completed IDs, failed partition IDs with operation-specific errors, and a separate fatal setup/context error. Bootstrap passes the IDs directly to `DeployAppPartitions` and logs each map entry once with application and partition identity; the initializer does not duplicate the failure logs. A failed reconstruction leaves its completion marker absent; any checkpoint-read/scan/write error prevents deployment of that partition. Let other attempts finish and continue to server readiness even when all attempts for a non-cluster application fail. The cluster application is the required exception described below. Start no asynchronous projectors or scheduled jobs for excluded partitions. Every command to an undeployed partition receives `500` with `partition not found`, without a runtime retry; a missing application instead returns `400` with `application not found`. Application and partition identity belong in structured log fields, not in these request errors. Missing completion markers are retried on a later bootstrap. Keep `IAppPartitions` and its existing errors unchanged, classify missing applications through its existing registration lookup, and do not retain the returned error map in request handling.

- Pros: reduces bootstrap latency; isolates failed partitions from commands and background workers; supplies deployment IDs directly and gives bootstrap each error needed for logging; reuses the existing unknown-partition behavior without retained error state.
- Cons: failed partitions remain unavailable until a later successful bootstrap; readiness no longer implies that all partitions are deployed; operators use initial-recovery logs to diagnose why a partition was not deployed.
- Confidence: user-provided

Alternatives:

1. Process partitions sequentially and fail bootstrap on the first error
   - Pros: deterministic ordering and simple failure semantics.
   - Cons: one partition prevents every healthy partition and the public service from starting.
   - Confidence: high
2. Process partitions with bounded concurrency and cancel remaining attempts on the first error
   - Pros: improves startup speed while still treating incomplete initialization as bootstrap-fatal.
   - Cons: discards the opportunity to initialize healthy partitions after one failure.
   - Confidence: medium
3. Attempt every partition but return an aggregated error and fail bootstrap afterward
   - Pros: maximizes checkpoint progress while reporting all failures.
   - Cons: still prevents service startup even though healthy partitions completed successfully.
   - Confidence: low
4. Deploy failed partitions and rely on runtime recovery retries
   - Pros: permits recovery without restarting the server.
   - Cons: starts background workers on partitions whose checkpoint initialization failed and conflicts with the requested deployment gate.
   - Confidence: superseded by user-provided selective-deployment policy

## Uncertainty: source of the initial-recovery concurrency bound

Decision: Bound partition initial-recovery workers by `runtime.GOMAXPROCS(0)`, sampled once at the start of each application's initialization. Do not add a VVM configuration field.

- Pros: follows the Go runtime's effective available-processor limit; adapts to the process environment without additional configuration; keeps initial-recovery concurrency independent of workspace recovery.
- Cons: CPU availability, rather than storage throughput, determines concurrency; operators cannot tune initial recovery independently without changing `GOMAXPROCS`.
- Confidence: user-provided

Alternatives:

1. Add `VVMConfig.NumInitialRecoveryWorkers` with a default of four
   - Pros: independently tunable for deployment and storage characteristics.
   - Cons: adds configuration validation and wiring for a bootstrap-only operation.
   - Confidence: high
2. Use a fixed package constant of four
   - Pros: simple and predictable.
   - Cons: cannot adapt to different process limits or deployment sizes.
   - Confidence: medium
3. Reuse `NumWSRecoverers`
   - Pros: avoids a new setting or runtime lookup.
   - Cons: couples application-level bootstrap work to an unrelated per-partition workspace-recovery limit.
   - Confidence: low

## Ambiguity: initial-recovery errors that may stop bootstrap

Decision: Only a pre-scheduling application setup failure or bootstrap context cancellation is fatal to an initial-recovery invocation. Every error inside a partition attempt is isolated to that partition, even if the same dependency error affects all partitions. For non-cluster applications, the server still starts and reports ready with affected partitions undeployed, and their command requests return `500` with `partition not found`. Bootstrap separately requires successful initialization and deployment of the cluster application's sole partition and fails if it is unavailable. Initial-recovery causes are returned to bootstrap in the partition-error map for logging, not passed into request handling. Existing bootstrap validation failures such as incompatible deployment descriptors remain fatal.

- Pros: keeps the service reachable and healthy partitions usable; callers receive the existing unknown-partition error while initial-recovery logs contain the failure details.
- Cons: readiness no longer means every partition has a usable checkpoint; an outage can produce repeated per-partition logs and request errors after startup.
- Confidence: user-provided

Alternatives:

1. Treat only context cancellation as fatal
   - Pros: maximizes startup availability under setup failures as well.
   - Cons: bootstrap may lack the application structures or dependencies required to schedule meaningful partition attempts.
   - Confidence: low
2. Treat shared-storage failures as fatal even when encountered by partition workers
   - Pros: stops startup early when a systemic outage is likely.
   - Cons: requires unreliable local-versus-global error classification and prevents healthy partitions from serving requests after startup.
   - Confidence: medium

## Dependency: bootstrap currently sends deployment commands through the cluster partition

Decision: The cluster application has one partition and bootstrap requires it. Return an appropriate error if its workspace initialization, checkpoint initialization, or partition deployment fails, including an empty successful-partition list. Stop before registering other applications or calling `c.cluster.DeployApp`. This supersedes the earlier unavailable-cluster fallback that performed read-only validation and deferred provisioning. Partition failures in other applications remain isolated and do not prevent readiness.

- Pros: makes the cluster dependency explicit; avoids command retries against an unavailable cluster partition; preserves the normal descriptor-validation and provisioning path.
- Cons: cluster-partition failure prevents the server from reaching readiness even when other applications could recover.
- Confidence: user-provided

## Safety: declared partition count is independent of checkpoint initialization

Decision: Use `AppDeploymentDescriptor.NumParts` as the total partition count supplied to `c.cluster.DeployApp`, local application registration, `AppPartsCount`, and workspace routing. Checkpoint initialization supplies only the IDs eligible for runtime partition deployment. Exclude failed IDs from `DeployAppPartitions`, without removing, renumbering, or reducing the application's declared partitions. Apply this rule when one, several, or every partition of a non-cluster application fails.

`c.cluster.DeployApp` validates the supplied descriptor against the persisted `cluster.App` descriptor and preserves the existing `409 Conflict` response for changed partition or app-workspace counts. It neither reads checkpoint initialization outcomes nor counts deployed runtime partitions. On first deployment it persists the descriptor's full count; on later compatible deployments it retains that count even when fewer runtime partitions start. A genuinely changed configured count remains incompatible; silently taking the old count must not hide that mismatch.

Routing uses `WSID % NumPartitions`. For example, with `NumPartitions=3`, WSID `10` belongs to partition `1`. If checkpoint initialization fails for partition `1`, only IDs `[0, 2]` start, while the count stays `3`. Using the successful count `2` would route WSID `10` to partition `0`, where new writes could conflict with or overwrite data associated with a different partition history. The correct response is `500` with `partition not found` for the original partition, with no rerouting or log writes.

- Pros: preserves stable workspace placement and existing data; keeps descriptor compatibility independent of availability; requires no runtime deployment-state input in the cluster command.
- Cons: workspaces assigned to excluded partitions remain unavailable until a later successful bootstrap.
- Confidence: user-provided

Alternatives:

1. Supply the number of successfully initialized or deployed partitions
   - Pros: reflects current availability.
   - Cons: changes modulo-based routing and can direct writes into a different partition history; zero successful partitions also gives an invalid routing divisor.
   - Confidence: rejected for data safety
2. Remove failed IDs and renumber the remaining partitions
   - Pros: exposes a contiguous available runtime set.
   - Cons: changes stable partition identities and workspace placement without migrating stored data.
   - Confidence: rejected for data safety
3. Remove deployment-descriptor compatibility validation
   - Pros: accepts a changed configured count automatically.
   - Cons: permits destructive routing changes even without checkpoint failures.
   - Confidence: rejected for data safety

## Safety: workspace recovery follows rate limiting and request validation

Decision: Keep partition recovery before rate limiting and the other command validations, but defer workspace recovery until after rate limiting, command lookup, execute authorization, request-body unmarshalling, and unexpected-field validation. An authenticated principal without execute permission therefore receives `403 Forbidden`, and a rate-limited request receives `429 Too Many Requests`, without starting or occupying a workspace-recovery slot. This ordering does not bypass execute authorization: command execution and workspace recovery are reached only after the request passes that authorization check.

The PLog scan is safe before these gates because it is performed once per recovered partition instance and publishes state shared by every workspace in that partition. A partition contains many WSIDs, each of which can independently require a WLog scan. Allowing rejected requests to start those per-workspace scans would let requests distributed across unrecovered WSIDs occupy all four workspace-recovery slots, repeatedly force WLog I/O, and delay authorized users. That multiplicative, WSID-addressable work is a basis for denial-of-service-like recovery-slot exhaustion, whereas the single partition-level PLog scan is bounded and shared.

- Pros: rejects over-limit, unauthorized, and malformed requests before WLog I/O; prevents rejected requests across many WSIDs from exhausting workspace-recovery capacity; preserves the existing partition recovery prerequisite for partition-backed rate limiting and authorization.
- Cons: a request that is eventually rejected may still trigger the partition's one shared PLog recovery; clients can receive `429`, `403`, or request-validation errors before learning that the workspace itself still needs recovery.
- Confidence: user-provided

Alternatives:

1. Run workspace recovery immediately after partition recovery
   - Pros: exposes workspace recovery state before the remaining command checks.
   - Cons: lets rejected requests trigger WLog scans and consume all workspace-recovery slots across different WSIDs.
   - Confidence: rejected for denial-of-service resistance
2. Move partition recovery behind rate limiting and authorization as well
   - Pros: no recovery scan starts for a rejected request.
   - Cons: rate limiting and authorization depend on recovered partition state, and the single PLog scan is bounded and shared rather than repeated per WSID.
   - Confidence: rejected because the checks require the recovered partition
