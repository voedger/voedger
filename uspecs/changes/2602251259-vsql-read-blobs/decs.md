# Decisions: VSQL BLOB reading via blobinfo() and blobtext()

## 1. BLOB access path: blobprocessor.IRequestHandler vs federation.ReadBLOB

Use `blobprocessor.IRequestHandler.HandleRead_V2` (confidence: high).

Rationale: `change.md` explicitly mandates using `blobprocessor.IRequestHandler`. `HandleRead_V2(appQName, wsid, header, ctx, okResponseIniter, errorResponder, ownerRecord, fieldName, ownerID, requestSender)` handles BLOB ID resolution from the owner record field and `q.sys.DownloadBLOBAuthnz` internally. The final implementation wires `blobprocessor.IRequestHandlerPtr` and `bus.IRequestSenderPtr` through `sqlquery.Provide` → `sysprovide.ProvideStateless` → VVM bootstrap wiring, then passes the bootstrap-settled sender into `HandleRead_V2`.

Alternatives:

- `federation.IFederation.ReadBLOB` (confidence: low)
  - `federation` uses HTTP round-trips through the external URL, adding latency; bypasses the direct procbus path used by all other in-process BLOB reads

## 2. Size field in blobinfo() result

Use the standard `Content-Length` header in the blobber pipeline's `initResponse` step and read it from the `okResponseIniter` headers captured in the sqlquery handler (confidence: high).

Rationale: `blobprocessor.HandleRead_V2` delivers blob metadata through the headers passed to `okResponseIniter`. Size is available inside the blobber pipeline as `bw.blobState.Size`, so emitting `Content-Length` in `initResponse` exposes it without downloading BLOB data. This also aligns sqlquery metadata capture with federation BLOB reads, where `readBLOB` parses the same header into `iblobstorage.BLOBReader.BLOBSize`.

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

## 5. Wiring bootstrap-settled interfaces into sqlquery

Group bootstrap-settled interface pointers in `btstrp.SettledInterfacePtrs` and pass that struct through VVM wiring (confidence: high).

Rationale: the final implementation settles four interface pointers during bootstrap: blobber app storage, router app storage, blob handler, and request sender. Grouping them in `SettledInterfacePtrs` keeps bootstrap ownership local to `btstrp`, reduces loose parameters across `btstrp.Bootstrap`, `provideBootstrapOperator`, `provideStatelessResources`, and the generated wire file, and lets sqlquery consume the settled handler/sender pointers without introducing another VVM-local container.

Alternatives:

- Pass four separate pointer parameters through bootstrap and VVM wiring (confidence: medium)
  - Works but spreads one concept across multiple signatures and test helpers
- Introduce a VVM-local grouping type instead of a bootstrap-owned one (confidence: low)
  - Hides a bootstrap concept outside the package that actually settles those pointers
