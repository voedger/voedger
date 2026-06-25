---
change_id: 2606181452-remove-unused-alias-recordid
type: refactor
issue_url: https://untill.atlassian.net/browse/AIR-4260
domains: [prod]
scope: [auth]
---

# Change request: Remove unused RecordID return from getActiveLoginAlias

Refs:

- [AIR-4260: RecordID result of getActiveLoginAlias is never used](./issue-AIR-4260.md)

## Why

The `getActiveLoginAlias` helper in the registry login-alias code returns a `RecordID` that every caller discards. The dead return value adds noise to the signature and invites confusion about whether the id is meant to be consumed.

## What

Simplify the registry login-alias lookup helper without changing any externally observable behavior:

- Drop the unused `RecordID` from the return signatures of the internal `getActiveLoginAlias` and `getLoginAlias` helpers, leaving the `IStateValue` and `error` they actually provide
- Adjust the call sites in `pkg/registry` so they no longer discard a `RecordID` placeholder
- Preserve all current sign-in, alias deactivation, and identifier-availability behavior: alias resolution, ownership checks, and record updates continue to operate through the returned `IStateValue` exactly as before

## Construction

- [x] update: [registry/impl_setloginalias.go](../../../../../pkg/registry/impl_setloginalias.go)
  - remove: `RecordID` from the return signature of `getActiveLoginAlias`, leaving `(istructs.IStateValue, error)`; drop the now-redundant `NullRecordID` bookkeeping in the inactive-alias branch
  - remove: `RecordID` from the return signature of `getLoginAlias`, leaving `(istructs.IStateValue, error)`; keep `recordID` as a local since it is still needed to build the record key
  - update: call site in `execCmdPutLoginAliasIndex` to drop the discarded `RecordID` placeholder
  - update: call site in `execCmdDeactivateLoginAliasIndex` to drop the discarded `RecordID` placeholder
  - update: call site in `assertIdentifierAvailable` to drop the discarded `RecordID` placeholder
  - update: call site in `resolveAliasSignInLogin` to drop the discarded `RecordID` placeholder
