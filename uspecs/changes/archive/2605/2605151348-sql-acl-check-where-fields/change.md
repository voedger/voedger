---
registered_at: 2026-05-15T13:25:19Z
change_id: 2605151325-sql-acl-check-where-fields
baseline: e14db6996bb6c70f80693907672d39e984332873
issue_url: https://untill.atlassian.net/browse/AIR-3289
archived_at: 2026-05-15T13:48:34Z
---

# Change request: Enforce SELECT ACL on fields in VSQL WHERE clause

## Why

VSQL SELECT queries currently authorize only the fields that appear in the result projection, ignoring fields referenced in the `WHERE` clause. With grants such as `GRANT SELECT(fld1)` and `REVOKE SELECT(fld2)`, a query like `SELECT fld1 FROM t WHERE fld2 = 123` succeeds and implicitly leaks the value of `fld2`, which is a security vulnerability.

## What

Authorize every field referenced in a VSQL SELECT statement, including fields used inside the `WHERE` clause:

- A SELECT request whose `WHERE` clause references any field for which `SELECT` is not allowed is rejected with `403 Forbidden`
- Authorization covers fields used in result projection and in the `WHERE` clause uniformly

## Functional design

- [x] create: [apps/vsql-acl.feature](../../../../specs/prod/apps/vsql-acl.feature)
  - Feature specification with scenarios for ACL enforcement on VSQL SELECT, covering fields in the result projection and in the `WHERE` clause, including the `403 Forbidden` response when any referenced field has `SELECT` denied

## Construction

- [x] update: [sys/it/impl_sqlquery_test.go](../../../../../pkg/sys/it/impl_sqlquery_test.go)
  - add: subtest verifying SELECT with allowed projection but denied field in `WHERE` returns `403 Forbidden`
  - add: subtest verifying SELECT with denied field in `WHERE` combined via `AND` and via `OR` returns `403 Forbidden`
  - add: subtest verifying SELECT with denied field in nested `WHERE` expression returns `403 Forbidden`
  - add: subtest verifying SELECT succeeds when every projected and `WHERE`-referenced field is allowed
  - add: subtests verifying SELECT with denied field on the left of `IN`, inside an `IN` value tuple and on the left of `NOT IN` returns `403 Forbidden`
  - add: subtest verifying SELECT with denied field qualified by the source table name (e.g. `where TestCDocWithDeniedFields.DeniedFld2 = 1`) returns `403 Forbidden`
  - dropped (out of scope for this entrypoint): view subtest — VSQL view reading only supports key-field filters in `WHERE`, and the existing test schema grants every key field of `app1pkg.DailyIdx` to `LimitedAccessRole`; the same security guarantee for views is covered by the unified projection+`WHERE` ACL union in `sys/sqlquery/impl.go`

- [x] update: [sys/sqlquery/impl.go](../../../../../pkg/sys/sqlquery/impl.go)
  - add: `collectWhereFields` helper that traverses a `sqlparser.Expr` via `sqlparser.Walk`, collecting every `*sqlparser.ColName` referenced in the `WHERE` clause, recovering the original case via `recoverFieldName` and ignoring identifiers that are not real fields of the source type (e.g. the `id` pseudo-column used by record-by-id lookup)
  - update: ACL check block to pass the union of projection fields and `WHERE` fields to `apppart.IsOperationAllowed`, so that any field denied by `SELECT` ACL causes a `403 Forbidden`
  - update: behavior so that when the projection is `*` (`f.acceptAll`), the ACL check still runs against the `WHERE` field set even if no explicit projection fields were collected
  - update: `collectWhereFields` strips any column qualifier (e.g. `tbl.fld`) before the schema/ACL lookup, since VSQL is single-table and schema field names are unqualified — preventing qualified column references from silently bypassing the ACL set
