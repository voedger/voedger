# Implementation plan: Recover correct case in parsed SQL query fields and tables

## Construction

### Case recovery helpers

- [x] update: [impl.go](../../pkg/sys/sqlquery/impl.go)
  - add: `recoverTableName` — passes `plog`/`wlog` through unchanged; tries exact match via `Type()`, then case-insensitive search through `Types()`; returns original name if no match (no error)
  - add: `recoverFieldName` — tries exact match via `Field()`, then case-insensitive search through `Fields()`; returns original name if no match (no error)

### Apply case recovery

- [x] update: [impl.go](../../pkg/sys/sqlquery/impl.go)
  - update: replace the `blob` hack with `recoverTableName` to recover the correct table QName from the app schema
  - update: use recovered `source` QName for `readViewRecords` call instead of re-creating from raw parsed table name
  - update: recover field names in `f.fields` after table is resolved, using `recoverFieldName` against the resolved type's fields
  - update: changed `"do not know how to read"` error to HTTP 400

- [x] update: [impl_records.go](../../pkg/sys/sqlquery/impl_records.go)
  - update: field validation loop to use `recoverFieldName` for case-insensitive matching with corrected field names stored back into `f.fields`

- [x] update: [impl_viewrecords.go](../../pkg/sys/sqlquery/impl_viewrecords.go)
  - update: field validation to use `recoverFieldName(view, field)` with corrected names stored back into `f.fields`; error message simplified to `"field '%s' does not exist in %s"`
  - update: key part lookup to use `recoverFieldName(view.Key(), k.name)` with corrected name stored back into `kk[i].name`

### Tests

- [x] update: [impl_sqlquery_test.go](../../pkg/sys/it/impl_sqlquery_test.go)
  - remove: plog `abracadabra` subtest (field is now silently recovered or passed through)
  - remove: view records error subtests for `abracadabra` field and `partKey` key — no longer error scenarios
  - add: `"Should recover lowercased table and field names"` — queries `sys.collectionview` (lowercase) with lowercase field names and expects results
  - update: `git.hub` test — `Expect500` → `Expect400`, error message without `TypeKind_null` suffix
