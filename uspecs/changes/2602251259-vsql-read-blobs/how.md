# How: VSQL BLOB reading via blobinfo() and blobtext()

## Approach

- Extend the SELECT clause parser in `impl.go` (sqlquery) to recognise `*sqlparser.FuncExpr` in addition to `*sqlparser.StarExpr` / `*sqlparser.AliasedExpr`; intercept function names `blobinfo` and `blobtext` and extract the field name and optional `startFrom` argument
- Enforce constraints in `sqlquery/impl.go`: blob functions are only valid when a DML entity ID is provided or the doc is a singleton; reject queries with a WHERE clause when blob functions are present
- Extend `initResponse` in `impl_read.go` (blobber) to emit `Content-Length` from `bw.blobState.Size` alongside the existing `Content-Type` and `X-BLOB-Name` headers
- Expose blob size to federation BLOB readers through a shared `readBLOB` helper that parses `Content-Length` into `iblobstorage.BLOBReader.BLOBSize`
- Thread `blobprocessor.IRequestHandlerPtr` and `bus.IRequestSenderPtr` through `sqlquery.Provide` → `sysprovide.ProvideStateless` → VVM wiring
- Group bootstrap-settled interface pointers in `btstrp.SettledInterfacePtrs` so VVM wiring can pass blob/router storages, blob handler, and request sender through one bootstrap-owned struct
- For both functions, group requests by field name and execute one `IRequestHandler.HandleRead_V2(appQName, wsid, header, ctx, okResponseIniter, errorResponder, ownerRecord, fieldName, ownerID, requestSender)` call per field so `blobinfo(field)` and `blobtext(field)` can share the same blobber call
  - `ownerRecord` is the doc QName from the query source, `fieldName` is the argument to the blob function, `ownerID` is the DML entity ID — `HandleRead_V2` handles owner-field BLOB ID lookup and `q.sys.DownloadBLOBAuthnz` internally
  - pass the bootstrap-settled `*requestSenderPtr` into `HandleRead_V2`
  - `okResponseIniter` captures `Content-Type`, `X-BLOB-Name`, and `Content-Length` headers from the blobber pipeline
- For `blobinfo()`: supply a discard `io.Writer`; read `name`, `mimetype`, and `size` from the captured headers
- For `blobtext()`: supply a limited `io.Writer` that skips the first `startFrom` bytes and captures at most 10 000 bytes; inspect the captured `Content-Type` header — return plain text if MIME is `text/*` or `application/json`, base64-encoded bytes otherwise
- Return blob function results as extra keys in the per-row JSON result alongside regular fields, using dedicated blob-function helpers in `impl_blobfuncs.go`

References:

- [pkg/sys/sqlquery/impl.go](../../pkg/sys/sqlquery/impl.go)
- [pkg/sys/sqlquery/impl_blobfuncs.go](../../pkg/sys/sqlquery/impl_blobfuncs.go)
- [pkg/sys/sqlquery/impl_records.go](../../pkg/sys/sqlquery/impl_records.go)
- [pkg/sys/sqlquery/provide.go](../../pkg/sys/sqlquery/provide.go)
- [pkg/sys/sysprovide/provide.go](../../pkg/sys/sysprovide/provide.go)
- [pkg/processors/blobber/interface.go](../../pkg/processors/blobber/interface.go)
- [pkg/processors/blobber/impl_read.go](../../pkg/processors/blobber/impl_read.go)
- [pkg/btstrp/types.go](../../pkg/btstrp/types.go)
- [pkg/btstrp/impl.go](../../pkg/btstrp/impl.go)
- [pkg/vvm/provide.go](../../pkg/vvm/provide.go)
- [pkg/coreutils/federation/impl.go](../../pkg/coreutils/federation/impl.go)
