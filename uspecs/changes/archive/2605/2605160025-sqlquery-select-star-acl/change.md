---
registered_at: 2026-05-15T17:34:49Z
change_id: 2605151734-sqlquery-select-star-acl
baseline: 59247f436a281fd98e8d1cde3265b2fda7063bc9
issue_url: https://untill.atlassian.net/browse/AIR-3943
archived_at: 2026-05-16T00:25:06Z
---

# Change request: Enforce SELECT ACL on all fields when VSQL projection is `*`

## Why

After AIR-3289 closed the `WHERE`-clause leak, the projection side of the same problem remains: a VSQL `select *` query still returns fields the caller's role is not granted `SELECT` on, because the ACL check is invoked with an empty field set and trivially passes. This is a data-confidentiality regression of the same class as AIR-3289 and must be closed for the field-level `SELECT` ACL to be a real boundary.

## What

Authorize every real field of the source type whenever the VSQL projection is `*`, so that any field with `SELECT` denied causes the query to be rejected:

- `select *` from a record (`IDoc`/`IRecord`) with any denied field is rejected with `403 Forbidden`
- `select *` from a view with any denied key or value field is rejected with `403 Forbidden`
- `select *` succeeds when every field of the source type is granted to the caller
- Explicit projections and `WHERE`-clause ACL behavior are unchanged

## Functional design

- [x] update: [apps/vsql-acl.feature](../../../../specs/prod/apps/vsql-acl.feature)
  - add: scenario for `select * from <table>.<id>` with a denied data field returning `403 Forbidden`
  - add: scenario for `select * from <view> where <key>=...` with a denied value field returning `403 Forbidden`
  - add: scenario for `select * from <view> where <key>=...` with a denied key field returning `403 Forbidden`
  - add: scenario for `select * from <table>.<id>` succeeding when every field of the record is granted

## Construction

- [x] update: [sys/it/impl_sqlquery_test.go](../../../../../pkg/sys/it/impl_sqlquery_test.go)
  - add: subtest in `TestAuthnz` verifying `select * from app1pkg.TestCDocWithDeniedFields.123` returns `403 Forbidden`
  - add: subtest in `TestAuthnz` verifying `select * from app1pkg.DailyIdx where Year=...` under `LimitedAccessRole` (a value field is not granted) returns `403 Forbidden`
  - add: subtest verifying `select * from app1pkg.payments.<id>` under `sys.WorkspaceOwner` (every field granted) succeeds

- [x] update: [sys/sqlquery/impl.go](../../../../../pkg/sys/sqlquery/impl.go)
  - update: ACL check block so that when `f.acceptAll` is `true`, `aclFields` is seeded with every user field of `sourceTableType` (via `appdef.IWithFields.UserFields()`, which on `IView` already spans both key and value); for views this also covers partition and clustering key fields so denial of any key field on `select *` yields `403`
  - keep: the explicit-projection path unchanged (`f.fields` continues to drive `aclFields` when `f.acceptAll` is `false`)
  - keep: the existing `collectWhereFields` union with `WHERE`-clause fields

- [x] Review
