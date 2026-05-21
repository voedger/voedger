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

Once `select *` authorizes every field, system fields (`sys.ID`, `sys.QName`, `sys.IsActive`, ...) become part of the check. The current ACL engine treats a partial-field grant (e.g. `GRANT SELECT(Fld1)`) as an exhaustive allow-list, so the very same query that worked before — `select sys.ID from t` under any non-empty SELECT grant — would start returning `403`. System fields are universally needed (IDs for joins, `sys.IsActive` for filtering, `sys.QName` for polymorphic reads) and have never been intended to be hidden by a field-list grant. They must follow the table-level grant implicitly, with an explicit `REVOKE SELECT(<sys-field>)` being the only way to remove them.

## What

Authorize every real field of the source type whenever the VSQL projection is `*`, so that any field with `SELECT` denied causes the query to be rejected:

- `select *` from a record (`IDoc`/`IRecord`) with any denied field is rejected with `403 Forbidden`
- `select *` from a view with any denied key or value field is rejected with `403 Forbidden`
- `select *` succeeds when every field of the source type is granted to the caller
- Explicit projections and `WHERE`-clause ACL behavior are unchanged

Make system fields follow the table-level SELECT grant implicitly. Given `TABLE t INHERITS sys.CDoc (Fld1 varchar, Fld2 int32)`:

- `GRANT SELECT(Fld1) ON TABLE t` — `Fld1` allowed, `Fld2` denied, every system field allowed
- `GRANT SELECT ON TABLE t` — every field including every system field allowed
- `GRANT SELECT ON TABLE t; REVOKE SELECT(Fld1) ON TABLE t` — `Fld2` and every system field allowed, `Fld1` denied
- `GRANT SELECT ON TABLE t; REVOKE SELECT(sys.ID) ON TABLE t` — every field allowed except `sys.ID`

The invariant: a system field is allowed iff the role has any matching `Allow` SELECT rule on the type and no matching `Deny` rule names that system field; the rule's user-field list does not constrain system fields.

## Functional design

- [x] update: [apps/vsql-acl.feature](../../../../specs/prod/apps/vsql-acl.feature)
  - add: scenario for `select * from <table>.<id>` with a denied data field returning `403 Forbidden`
  - add: scenario for `select * from <view> where <key>=...` with a denied value field returning `403 Forbidden`
  - add: scenario for `select * from <view> where <key>=...` with a denied key field returning `403 Forbidden`
  - add: scenario for `select * from <table>.<id>` succeeding when every field of the record is granted
  - add: rule "System fields SELECT follows the table grant implicitly" with scenarios mirroring the four cases above (partial grant allows sys fields, full grant allows sys fields, full grant + user-field revoke keeps sys fields, explicit `REVOKE SELECT(sys.ID)` denies that field and any `select *` that implicitly selects it)

## Construction

- [x] update: [sys/it/impl_sqlquery_test.go](../../../../../pkg/sys/it/impl_sqlquery_test.go)
  - [x] add: subtest in `TestAuthnz` verifying `select * from app1pkg.TestCDocWithDeniedFields.123` returns `403 Forbidden`
  - [x] add: subtest in `TestAuthnz` verifying `select * from app1pkg.DailyIdx where Year=...` under `LimitedAccessRole` (a value field is not granted) returns `403 Forbidden`
  - [x] add: subtest verifying `select * from app1pkg.payments.<id>` under `sys.WorkspaceOwner` (every field granted) succeeds
  - [x] add: subtest covering `REVOKE SELECT(sys.ID)` on a table with an otherwise full SELECT grant — `select sys.ID` and `select *` return `403`, `select <granted-user-field>` succeeds

- [x] update: [sys/sqlquery/impl.go](../../../../../pkg/sys/sqlquery/impl.go)
  - [x] update: ACL check block so that when `f.acceptAll` is `true`, `aclFields` is seeded with every field of `sourceTableType` via `appdef.IWithFields.Fields()` (both user and system); for views this also covers partition and clustering key fields so denial of any key field on `select *` yields `403`
  - [x] remove: the `collectACLSystemFields` helper and its call site — once the ACL engine treats system fields as implicitly allowed by any matching `Allow` SELECT rule (see below), seeding the request with every system field of the type is sufficient and the per-rule sys-field scan is dead code
  - keep: the explicit-projection path unchanged (`f.fields` continues to drive `aclFields` when `f.acceptAll` is `false`)
  - keep: the existing `collectWhereFields` union with `WHERE`-clause fields

- [x] update: [appdef/acl/impl.go](../../../../../pkg/appdef/acl/impl.go) — `checkOperationOnTypeForRoles`
  - [x] update: on a matching `Allow` SELECT rule whose filter `HasFields()`, also add every system field of the resource to `allowedFields` (in addition to the rule's user fields)
  - keep: full-grant `Allow` (no field list) behavior — already adds every field including system ones
  - keep: `Deny` semantics — an explicit `REVOKE SELECT(sys.X)` continues to `delete(allowedFields, "sys.X")` and is the only way to remove a system field
  - [x] add: unit tests in `pkg/appdef/acl/...` covering the four cases from the "What" section (sys field allowed under partial grant, allowed under full grant, allowed after user-field revoke, denied after explicit sys-field revoke)

- [x] Review
