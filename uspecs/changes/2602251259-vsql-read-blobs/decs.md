# Decisions: VSQL BLOB reading via blobinfo() and blobtext()

## 1. BLOB access path: blobprocessor.IRequestHandler vs federation.ReadBLOB

Use `blobprocessor.IRequestHandler.HandleRead_V2` (confidence: high).

Rationale: `change.md` explicitly mandates using `blobprocessor.IRequestHandler`. `HandleRead_V2(appQName, wsid, header, ctx, okResponseIniter, errorResponder, ownerRecord, fieldName, ownerID, requestSender)` handles BLOB ID resolution from the owner record field and `q.sys.DownloadBLOBAuthnz` internally. `blobprocessor.IRequestHandler` is wired in the VVM (`wire_gen.go`: `blobprocessor.NewIRequestHandler`) and reachable by adding it as a parameter to `sqlquery.Provide` → `sysprovide/provide.go` → VVM wire. A `bus.IRequestSender` is instantiated locally per-request inside the query function body (`bus.NewIRequestSender`) — it is not injected as an external parameter.

Alternatives:

- `federation.IFederation.ReadBLOB` (confidence: low)
  - `federation` uses HTTP round-trips through the external URL, adding latency; bypasses the direct procbus path used by all other in-process BLOB reads

## 2. Size field in blobinfo() result

Add an `X-BLOB-Size` header to the blobber pipeline's `initResponse` step and read it from the `okResponseIniter` headers captured in the sqlquery handler (confidence: high).

Rationale: `blobprocessor.HandleRead_V2` delivers blob metadata only through the headers passed to `okResponseIniter` (`Content-Type`, `X-BLOB-Name`). Size is not currently exposed there; it is available inside the blobber pipeline as `bw.blobState.Size`. Extending `initResponse` in `impl_read.go` to emit `X-BLOB-Size` is a minimal, self-contained change that makes size available without downloading BLOB data. `blobinfo()` then calls `HandleRead_V2` with a discard `io.Writer` and reads name, content-type, and size from the headers.

Alternatives:

- Download BLOB data and count bytes for size (confidence: low)
  - Wastes bandwidth for large blobs; incompatible with the intent of a metadata-only function
- Return size as 0 / omit it from blobinfo() (confidence: low)
  - Loses important metadata; contradicts `change.md` requirement

## 3. SELECT clause with mixed fields and blob functions

Parse `*sqlparser.FuncExpr` in the SELECT walk loop alongside `*sqlparser.AliasedExpr` (confidence: high).

Rationale: The existing SELECT parser in `impl.go` already iterates `s.SelectExprs`. Adding a `*sqlparser.FuncExpr` case lets the parser collect blob function descriptors (name + arguments) alongside regular field names. Both regular-field results and blob function results are merged into the single per-row JSON object returned by the callback. A separate query operation for blob functions is unnecessary complexity.

Alternatives:

- Make blob functions a separate top-level statement type in `dml` (confidence: low)
  - Breaks backward compatibility with the `select` syntax shown in `change.md`; over-engineered for two functions

## 4. WHERE clause rejection location

Reject blob-function queries with a WHERE clause in `impl.go` (sqlquery), after parsing, not in `dml` (confidence: high).

Rationale: The `dml` package parses query strings into an `Op` struct and handles workspace/entity routing; it does not see SQL AST details like whether specific SELECT expressions are function calls. The sqlquery `impl.go` already has the parsed AST and the list of detected blob functions, making it the natural place to enforce the constraint `"WHERE clause not allowed with blobinfo/blobtext"`.

Alternatives:

- Reject in `dml` package (confidence: low)
  - `dml` would need to understand blob function semantics; mixes concerns

## 5. Auth token propagation to federation.ReadBLOB

Pass the caller's token via `httpu.WithAuthorizeBy(token)` to `federation.ReadBLOB` (confidence: high).

Rationale: Inside `provideExecQrySQLQuery`, the auth token is available from the query state: `args.State.(interface{ Token() string }).Token()` — the same mechanism used by federation calls elsewhere in the function (e.g., cross-workspace queries). Passing the token in `ReadBLOB` ensures the blobber's `q.sys.DownloadBLOBAuthnz` check is satisfied under the caller's identity without any extra token minting.

Alternatives:

- Use a system token (confidence: low)
  - Bypasses per-user authorization; security risk
