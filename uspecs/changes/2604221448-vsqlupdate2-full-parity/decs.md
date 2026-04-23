# Decisions: Make q.cluster.VSqlUpdate2 cover all c.cluster.VSqlUpdate features

## Retention of c.cluster.VSqlUpdate

Keep `c.cluster.VSqlUpdate` unchanged (confidence: high).

Rationale: the legacy command must remain callable for backward compatibility with frontends and integrations that have not yet migrated to `q.cluster.VSqlUpdate2`. No code, schema, or wiring changes are applied to the existing command extension, its handler, or its registration.

Alternatives:

- Remove the extension entirely (confidence: low)
  - Breaks every unmigrated caller with a 404 on the resource
- Replace the handler with a thin pass-through that delegates to `c.cluster.LogVSqlUpdate` + `c.sys.CUD` (confidence: low)
  - Changes legacy behavior in ways unrelated to this change and risks response-shape regressions

## Router shim retention

Keep the AIR-3656 / AIR-3661 router shim (`dispatchVSqlUpdateShim_V1` / `_V2`) active: the router continues to intercept `c.cluster.VSqlUpdate` calls, reroute them to `q.cluster.VSqlUpdate2`, and reshape the query response back into the legacy command envelope (confidence: high).

Rationale: the shim is the sole mechanism that prevents a self-deadlock when `c.cluster.VSqlUpdate` synchronously calls `c.sys.CUD` on the same command processor (issue #3845). Legacy callers that still target `c.cluster.VSqlUpdate` must keep working transparently, and the shim is the only place where the rerouting can happen without breaking the command envelope contract. The query handler and the shim share the `q.cluster.VSqlUpdate2` path; direct query callers consume the native row, legacy command callers see the reshaped command envelope.

Alternatives:

- Retire the shim and require all callers to migrate to `q.cluster.VSqlUpdate2` directly (confidence: low)
  - Breaks unmigrated callers and removes the deadlock guard without a replacement
- Keep the shim but move the reshape into the query handler (confidence: low)
  - The reshape is inherently a command-envelope concern (adding `CurrentWLogOffset` at the top level, preserving `NewIDs` / `CmdResult` shape); doing it in the query handler would require the handler to know whether it runs under a command or query caller, coupling it to transport details

## Response transformation location

Router-side inside the shim: the shim captures the query response body and rewrites it into the legacy command envelope shape (`CurrentWLogOffset` for API v1, `currentWLogOffset` for API v2). The query handler itself emits the native `VSqlUpdate2Result` row (confidence: high).

Rationale: the reshape is a transport-level concern (command envelope vs. query row), and the shim already captures the downstream response for this purpose. Keeping the reshape in the shim leaves the query handler free of caller-kind branching. Direct query callers consume `VSqlUpdate2Result` rows as defined by the schema; legacy command callers consume the envelope the shim produces.

Alternatives:

- Query-handler-side reshape (confidence: low)
  - Forces the handler to branch on the caller kind, which is a transport concern not visible in the query processor contract

## NewID plumbing for InsertTable

`q.cluster.VSqlUpdate2` calls `c.sys.CUD` via federation, reads `NewID` from the command response, and returns it to the caller in a `NewID` result field on `VSqlUpdate2Result` (confidence: high).

Rationale: the query processor has no `istructs.IIntents` to allocate ids locally; taking the id from the `c.sys.CUD` response reuses the id allocation already performed by the command processor and keeps the query stateless.

Alternatives:

- Keep rejecting `InsertTable` in the query path (confidence: low)
  - Prevents full parity with `c.cluster.VSqlUpdate` and forces callers to retain two code paths
- Introduce a cluster-app-scoped id allocator exposed to queries (confidence: low)
  - Large scope change with unclear benefit over reusing `c.sys.CUD`

## NewID result for non-insert DML

Emit `NewID = istructs.NullRecordID` (0) on the `VSqlUpdate2Result` row for every DML kind other than `insert table` (confidence: high).

Rationale: a present-but-null id keeps the result row shape stable for the record-id contract, lets clients pattern-match on a single field, and matches the convention already used in the codebase for absent record ids (the legacy `VSqlUpdateResult.NewID` uses the same nullable-`ref` shape).

Alternatives:

- Omit the `NewID` field when not applicable (confidence: low)
  - Forces clients to branch on field presence for a value that already has a canonical null encoding
- Emit a distinct sentinel (e.g., -1) (confidence: low)
  - Diverges from the established `NullRecordID` convention

## Shim log levels and fall-through

Shim operational logs (the `rerouting ...` announcement and the offset reporting line) are emitted at `Verbose` level, not `Info`. APIv2 preflight failures (body unmarshal error, missing / malformed `args`, args marshal error, initial `SendRequest` error) do not emit a shim-specific error log: `dispatchVSqlUpdateShim_V2` returns `false` and the caller in `impl_apiv2.go` falls through to the standard command processor path, which handles the error uniformly (confidence: high).

Rationale: `Verbose` keeps the shim quiet in production while remaining accessible for debugging via log capture (integration tests use `logger.StartCapture(t, logger.LogLevelVerbose)`). Fall-through delegates error reporting to the processor that would have owned the request without the shim, eliminating a parallel shim-error contract (`routing.vsqlupdate.error` with shim-specific messages) and guaranteeing the client sees the same error envelope regardless of whether the shim was tried. The remaining `routing.vsqlupdate.error` entry (shim reply failure) stays because, at that point, the reply has already been captured and the shim owns the flush.

Alternatives:

- Keep `Info` level for operational logs (confidence: low)
  - Pollutes production logs with per-request shim traces that carry no operational signal outside of debugging
- Keep the shim emitting its own APIv2 preflight error envelopes (confidence: low)
  - Duplicates the processor's error contract and forces callers to handle two shapes (shim vs. processor) for the same class of failures

## CUDWLogOffset result for non-CUD DML

Drop `NOT NULL` from `CUDWLogOffset` on `VSqlUpdate2Result` and emit `NullOffset` (0) on DML kinds that do not call `c.sys.CUD` (`update corrupted`, `unlogged update`, `unlogged insert`); callers that do not want to see the zero simply leave `CUDWLogOffset` out of the `elements[].fields` (apiv1) / `keys` (apiv2) selector on the request (confidence: high).

Rationale: the query processor pipeline has no supported mechanism for a handler to omit a schema-registered field on a per-row basis - `WithAllFields()` forces every declared field into the row and kind-specific result types would fragment the single-row contract. At the same time, the legacy command shape that unmigrated callers see via the router shim never exposes `CUDWLogOffset`, so the asymmetry with `NewID` only matters for direct `q.cluster.VSqlUpdate2` callers, who already choose which fields to pull through the selector. Dropping `NOT NULL` on the schema field keeps the type correct (the value is genuinely absent on non-CUD paths) and leaves room for the framework to support true omission later without a schema migration.

Alternatives:

- Frame-level field omission via `WithAllFields()` changes (confidence: low)
  - Invasive framework change rejected during implementation in favor of the field selector
- Split `VSqlUpdate2Result` into per-kind result types (confidence: low)
  - Over-engineered for a single optional field and breaks the single-row contract callers already consume
- Emit a distinct sentinel (e.g., -1) (confidence: low)
  - Diverges from the `NullOffset` convention and still requires caller-side branching
