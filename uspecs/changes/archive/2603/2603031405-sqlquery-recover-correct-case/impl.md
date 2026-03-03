# Implementation plan: Recover correct case in parsed SQL query fields and tables

## Construction

### Case recovery helpers

- [x] update: [impl.go](../../pkg/sys/sqlquery/impl.go)
  - add: `recoverTableName` — passes `plog`/`wlog` through unchanged; tries exact match via `Type()`, then case-insensitive search through `Types()`; returns original name if no match (no error)
  - add: `recoverFieldName` — tries exact match via `Field()`, then case-insensitive search through `Fields()`; returns original name if no match (no error)

### Apply case recovery

- [x] update: [impl.go](../../pkg/sys/sqlquery/impl.go)
  - update: replace the `blob` hack with `recoverTableName`; compute `sourceTableName` and `sourceTableType` upfront before filter building
  - update: recover field names during SELECT expression parsing using `sourceTableType`; use `sourceTableName` for all downstream calls
  - update: changed `"do not know how to read"` error to HTTP 400

- [x] update: [impl_records.go](../../pkg/sys/sqlquery/impl_records.go)
  - update: field validation loop to use `recoverFieldName` for case-insensitive matching with corrected field names stored back into `f.fields`

- [x] update: [impl_viewrecords.go](../../pkg/sys/sqlquery/impl_viewrecords.go)
  - update: build `allowedFields` from view key/value fields applying `recoverFieldName` to normalize; original error message and structure retained
  - update: key name recovery at parse time in `keyParts` closure (`name: recoverFieldName(view, name)`) and again in key iteration loop via `recoverFieldName(view.Key(), k.name)`

### Tests

- [x] update: [impl_sqlquery_test.go](../../pkg/sys/it/impl_sqlquery_test.go)
  - remove: plog `abracadabra` subtest (field is silently passed through for plog/wlog)
  - replace: `partKey` key def error test → `"Should recover lowercased table and field names"` — queries `sys.collectionview` (lowercase) with lowercase field names and expects results
  - update: `git.hub` test — `Expect500` → `Expect400`, error message without `TypeKind_null` suffix
