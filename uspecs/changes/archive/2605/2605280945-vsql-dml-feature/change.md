---
registered_at: 2026-05-22T19:05:58Z
change_id: 2605221905-vsql-dml-feature
baseline: 4344f6ebd11c8a3cdf7509027b56717ce5982fed
issue_url: https://untill.atlassian.net/browse/AIR-4027
---

# Change request: Describe VSQL DML in a dedicated Gherkin feature

## Why

The existing `vsql-acl.feature` documents only the access-control aspects of VSQL data manipulation and omits the broader DML surface (select and update against tables, views, and singletons; cross-workspace and cross-application access; behaviour with and without a specific record ID; HTTP response status codes). Coverage gaps in the spec make it unclear which behaviours are intentional contracts versus implementation details, and make new contributors rely on test code to learn the feature.

## What

Reorganize the VSQL DML documentation into a single, dedicated functional specification.

- Drop `vsql-acl.feature`
- Author `vsql-dml.feature` covering, for both `SELECT` and `UPDATE`:
  - Targets: table, view, singleton
  - Cross-workspace and cross-application access
  - With and without a specific record ID
  - ACL grant and deny effects (subsuming what `vsql-acl.feature` covered)
  - HTTP response status codes for success and failure paths

## Functional design

- [x] remove: [apps/vsql-acl.feature](../../../../specs/prod/apps/vsql-acl.feature)
  - superseded by `vsql-dml.feature` which covers the same ACL aspects within a broader DML scope
- [x] create: [apps/vsql-dml.feature](../../../../specs/prod/apps/vsql-dml.feature)
  - Feature specification for VSQL DML operations covering `SELECT` and `UPDATE` against table, view, and singleton; cross-workspace and cross-application targets; with and without a specific record ID; ACL effects; HTTP response status codes
  - Derive scenarios from existing integration tests: [it/impl_sqlquery_test.go](../../../../../../pkg/sys/it/impl_sqlquery_test.go) and [it/impl_vsqlupdate_test.go](../../../../../../pkg/sys/it/impl_vsqlupdate_test.go)
