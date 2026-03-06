# How: VSQL BLOB reading via blobinfo() and blobtext()

## Approach

- Extend the SELECT clause parser in `impl.go` (sqlquery) to recognise `*sqlparser.FuncExpr` in addition to `*sqlparser.StarExpr` / `*sqlparser.AliasedExpr`; intercept function names `blobinfo` and `blobtext` and extract the field name and optional `startFrom` argument
- Enforce constraints early: blob functions are only valid when a DML entity ID is provided or the doc is a singleton; reject queries with a WHERE clause when blob functions are present. This check should be done in `dml` package
- Extend `initResponse` in `impl_read.go` (blobber) to emit an `X-BLOB-Size` header (`bw.blobState.Size`) alongside the existing `Content-Type` and `X-BLOB-Name` headers
- Add `blobprocessor.IRequestHandler` as a parameter to `provideExecQrySQLQuery` in `impl.go` and `Provide` in `provide.go`; thread it through `sqlquery.Provide` → `sysprovide/provide.go` → VVM wire; `bus.IRequestSender` is NOT injected externally — instantiate a local one per request with `bus.NewIRequestSender`
- For both functions, call `IRequestHandler.HandleRead_V2(appQName, wsid, header, ctx, okResponseIniter, errorResponder, ownerRecord, fieldName, ownerID, localRequestSender)`:
  - `ownerRecord` is the doc QName from the query source, `fieldName` is the argument to the blob function, `ownerID` is the DML entity ID — `HandleRead_V2` handles owner-field BLOB ID lookup and `q.sys.DownloadBLOBAuthnz` internally
  - `okResponseIniter` captures `Content-Type`, `X-BLOB-Name`, and `X-BLOB-Size` headers from the blobber pipeline
- For `blobinfo()`: supply a discard `io.Writer`; read `name`, `mimetype`, and `size` from the captured headers; set `status` = `"completed"` on success
- For `blobtext()`: supply a limited `io.Writer` that skips the first `startFrom` bytes and captures at most 10 000 bytes; inspect the captured `Content-Type` header — return plain text if MIME is `text/*` or `application/json`, base64-encoded bytes otherwise
- Return blob function results as extra keys in the per-row JSON result alongside regular fields in `impl_records.go`

References:

- [pkg/sys/sqlquery/impl.go](../../pkg/sys/sqlquery/impl.go)
- [pkg/sys/sqlquery/impl_records.go](../../pkg/sys/sqlquery/impl_records.go)
- [pkg/sys/sqlquery/provide.go](../../pkg/sys/sqlquery/provide.go)
- [pkg/sys/sysprovide/provide.go](../../pkg/sys/sysprovide/provide.go)
- [pkg/processors/blobber/interface.go](../../pkg/processors/blobber/interface.go)
- [pkg/processors/blobber/impl_read.go](../../pkg/processors/blobber/impl_read.go)
- [pkg/bus/provide.go](../../pkg/bus/provide.go)
