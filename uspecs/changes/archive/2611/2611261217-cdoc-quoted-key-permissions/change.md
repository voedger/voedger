---
change_id: 2611261159-cdoc-quoted-key-permissions
type: fix
issue_url: https://untill.atlassian.net/browse/AIR-5367
domains: [prod]
scope: [apps]
---

# Change request: Field authorization for quoted keys in API v2 CDoc reads

Refs:

- [AIR-5367: Reseller access, voedger: API v2 CDoc reads reject quoted sys.ID for authorized non-owner roles](./issue-AIR-5367.md)

## Why

A reseller with `SELECT` access needs the document ID alongside the user's name, but CDoc authorization treats the quotes around `"sys.ID"` as part of the schema field name. Removing the quotes makes response projection interpret the dot as a path separator and omit the ID. The existing integration test uses a workspace-owner token and passes because the legacy ACL allows the read, letting the handler suppress the lookup error and proceed to projection.

## What

Symptom: A non-owner with permission to read `sys.ID` and `name` receives HTTP 400 when requesting those fields from `untill.untill_users`.

```text
GET /cdocs/untill.untill_users?keys=%22sys.ID%22,name
    |
    v
The router decodes the keys parameter to "sys.ID",name
    |
    v
cdocsAuthorizeResult passes "sys.ID" to the ACL lookup  <-- fault: the field name still contains quotes
    |
    v
The schema lookup fails and the legacy ACL does not grant the non-owner access
    |
    v
HTTP 400: field «"sys.ID"» in CDoc «untill.untill_users»
```

Corrected behavior: After resolving quoted top-level keys for authorization and preserving them for projection, the CDoc handler returns HTTP 200 with exactly `sys.ID` and `name` for the authorized non-owner, or HTTP 403 if the caller lacks permission for either field.

## How

Decisions:

- Reuse the projection path parser in the CDoc handler. When a key parses as one segment, pass that segment to both the legacy and VSQL ACL checks. For a multi-segment path, pass the original expression to both checks
- Verify field-level denial with a non-owner role that can read selected fields of the same CDoc. A successful read of its permitted fields establishes that the role passes the table-level check

Assumptions:

- None

References:

- [Path parsing and table-level request authorization](../../../../../pkg/processors/query2/impl.go)
- [Use of parsed paths in field projection](../../../../../pkg/processors/query2/types.go)
- [Legacy and VSQL authorization for CDoc results](../../../../../pkg/processors/query2/impl_cdocs_handler.go)
- [Integration coverage for field-specific grants](../../../../../pkg/sys/it/impl_qpv2_test.go)

## Construction

- [x] update: [sys/it/impl_qpv2_test.go](../../../../../pkg/sys/it/impl_qpv2_test.go)
  - Extend `TestQueryProcessor2_CDocs` with a non-owner `app1pkg.ApiRole` token. Request `?keys=%22sys.ID%22,name` on `app1pkg.category` and verify HTTP 200 with exactly the seeded document's ID in `sys.ID` and `"Awesome food"` in `name`
  - With a non-owner `app1pkg.LimitedAccessRole` token, verify that `?keys=name` returns HTTP 200 and the seeded name, then request `?keys=%22sys.ID%22,name` with the same token and verify HTTP 403
- [x] update: [vit/schemaTestApp1.vsql](../../../../../pkg/vit/schemaTestApp1.vsql)
  - Grant `SELECT(name)` on `category` to the existing `LimitedAccessRole`, then explicitly revoke `SELECT(sys.ID)` for that role because a partial field grant also grants access to system fields
- [x] update: [query2/impl_cdocs_handler.go](../../../../../pkg/processors/query2/impl_cdocs_handler.go)
  - In `cdocsAuthorizeResult`, build a separate list of permission-check fields from explicit keys using `splitPath`. Use the parsed field name for a single segment and the original expression for a multi-segment path
  - Pass that list to both ACL checks and keep the original `Constraints.Keys` available to the response projection
