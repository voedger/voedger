# Reseller access, voedger: make token role scopes explicit

- URL: https://untill.atlassian.net/browse/AIR-5210
- ID: AIR-5210
- State: in-progress
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov
- Linked issues: AIR-1808 (parent), AIR-5083 (blocks)

## Why

* `RoleType.WSID` identifies an exact target workspace for API tokens and the owner of target workspaces for regular tokens
* Its interpretation depends on `IsAPIToken`, making grants difficult to understand and easy to reinterpret incorrectly when creating or transforming tokens

## What

* Represent exact-workspace and owner-workspace scopes explicitly in token role grants
* Make authentication match grants according to their explicit scope
* Update token issuance and enrichment to populate scope explicitly
* Preserve scope through refresh and include it in grant deduplication
* Support legacy tokens by deriving missing scope from their original `IsAPIToken` value
* Preserve existing authorization and enrichment behavior during this representation change
* Test both scopes, legacy-token compatibility, refresh, and deduplication

