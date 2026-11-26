# Reseller access, voedger: API v2 CDoc reads reject quoted sys.ID for authorized non-owner roles

- URL: https://untill.atlassian.net/browse/AIR-5367
- ID: AIR-5367
- State: in-progress
- Author: Denis Gribanov
- Labels: none
- Parent: [AIR-1808: Enable Reseller Diagnostic Access to Clients Backoffice (first stage)](https://untill.atlassian.net/browse/AIR-1808)

## Why

`GET /cdocs/untill.untill_users?keys=%22sys.ID%22,name` fails with `field «"sys.ID"» in CDoc «untill.untill_users»`. Axios correctly encodes the quotes as `%22`. After URL decoding, the CDoc handler passes `"sys.ID"` directly to the field permission check, which looks for a schema field whose name includes the quotes. The actual field name is `sys.ID`. Removing the quotes makes the response projection interpret the dot as a nested path and omit the ID.

The existing test at `impl_qpv2_test.go:2453` passes because it uses `ws.Owner.Token`. The legacy ACL allows the workspace owner to read the test CDoc, so the handler suppresses the field lookup error. The request then reaches response projection, which correctly interprets `"sys.ID"` as one field and returns it. A non-owner role with `SELECT` access can hit the lookup error before projection runs.

## What

* Resolve quoted top-level keys to their schema field names before checking CDoc field permissions. For `"sys.ID"`, check permission on `sys.ID`
* Preserve the original key expression for response projection so the dot in `"sys.ID"` remains part of the field name
* Add an integration test using a non-owner role with `SELECT` access. Request `?keys=%22sys.ID%22,name` and verify HTTP `200` with exactly `sys.ID` and `name`, including the expected document ID
* Verify that a role lacking permission for a requested field still receives HTTP `403`
