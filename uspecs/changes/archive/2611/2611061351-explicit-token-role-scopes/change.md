---
change_id: 2611061146-explicit-token-role-scopes
type: refactor
issue_url: https://untill.atlassian.net/browse/AIR-5210
domains: [prod]
scope: [auth]
---

# Change request: Explicit token role scopes

Refs:

- [AIR-5210: Reseller access, voedger: make token role scopes explicit](./issue-AIR-5210.md)

## Why

We need to specify a role in a token for an exact workspace, so let's add a role scope and use [RoleType.WSID](https://github.com/voedger/voedger/blob/5c45e847270a93a7c5c45f7f4026c9d9415aeb6d/pkg/itokens-payloads/types.go#L32). It's expected to mean "any workspace whose WorkspaceDescriptor.OwnerWSID == RoleType.WSID", but there is confusion: `RoleType.WSID` currently has two meanings, selected implicitly by `PrincipalPayload.IsAPIToken`:

- For an API token, `WSID = 42` grants the role in Workspace 42 only
- For a regular User or Device token, `WSID = 42` grants the role in each Workspace whose immediate owner is Workspace 42

The role entry itself does not say which meaning applies. Understanding or transforming a grant therefore requires knowing the token kind; changing `IsAPIToken` can reinterpret the same `{WSID, QName}` as a different set of target Workspaces. This makes authentication, API-token conversion, and enrichment easy to misunderstand.

This PR records that distinction explicitly in each role grant and preserves it through validation, refresh, and deduplication, while retaining existing authorization and enrichment behavior.

## What

- Token role grants explicitly distinguish an exact target Workspace from Workspaces directly owned by the referenced Workspace
- Authentication matches grants according to their explicit scope, while legacy tokens retain the scope inferred from their original `IsAPIToken` value
- Token issuance and enrichment populate explicit scopes, and refresh preserves them
- Enrichment deduplicates grants by their complete scope, workspace identifier, and role name
- No behavior change in authorization decisions, token endpoint authorization, or the effective permissions produced by existing enrichment, including parent-to-child propagation
- Regression coverage verifies both scopes, legacy-token compatibility, refresh, enrichment, and grant deduplication

## How

Decisions:

- Use an integer `RoleScope` with `RoleScopeWorkspacesOwnedBy = 0` to emit the role in each Workspace whose owner is the specified WSID and `RoleScopeWorkspace = 1` for an exact target Workspace, making the owned-by scope the default for Go role literals
- Keep the existing Principal Token payload type and claim names, adding only the role scope claim so the JWT audience and existing identity claims remain compatible
- Own scope normalization and validation in PrincipalPayload.ValidateTokenClaims, invoked by ValidateToken after successful verification and decoding, so direct and application-bound validation, refresh, and enrichment share the same rules without adding authentication-specific behavior to the generic JWT signer
- During token validation, distinguish a missing scope claim from an explicit zero: infer a missing legacy scope from the original token kind, preserve explicit zero as `RoleScopeWorkspacesOwnedBy`, and reject unknown explicit integer values
- Use standard JSON serialization for issuance without scope validation; unsupported scopes in issued tokens are rejected by ValidateToken
- Serialize the scope claim even when it is zero so a newly issued default scope remains distinguishable from a missing scope in legacy API tokens; API-token issuance explicitly selects `RoleScopeWorkspace`
- Keep the existing API-token and user/device principal-composition stages, including their role sources, descriptor requirements, and early returns, and use explicit scope for grant matching within those stages
- Continue enrichment from effective role principals, retaining their emitted workspace identifiers and assigning new grants the scope corresponding to the existing token kind, so this representation change preserves current delegation behavior
- Preserve API-token conversion's existing carried roles and explicitly convert their scopes to exact-workspace grants, constructing the output without modifying the source payload's role entries
- During mixed-version operation, emit only the existing token-kind and scope combinations so older validators can ignore the additive scope claim and older refresh paths can omit it without changing access decisions
- Verify compatibility through real signed-token round trips and request-level token lifecycle coverage using the existing payload tests and VIT infrastructure, with assertions on effective access as well as token contents

Role representation:

```go
type RoleScope int

const (
	RoleScopeWorkspacesOwnedBy RoleScope = iota
	RoleScopeWorkspace
)

type RoleType struct {
	Scope RoleScope
	WSID  istructs.WSID
	QName appdef.QName
}
```

Assumptions:

- External consumers treat Principal Tokens as opaque bearer tokens or tolerate additive fields in role entries

Out of scope:

- Removing workspace identifiers from effective principals or changing the subject-identity contract
- Separating source grants from effective principals or changing enrichment delegation policy
- Additional scope kinds or new combinations of token kind and scope

References:

- [Current principal payload and role representation](../../../../../pkg/itokens-payloads/types.go)
- [Signed payload round-trip test infrastructure](../../../../../pkg/itokens-payloads/impl_test.go)
- [JWT audience derivation and payload serialization](../../../../../pkg/itokensjwt/impl.go)
- [Principal composition stages and token-role matching](../../../../../pkg/iauthnzimpl/impl.go)
- [API-token conversion and carried-role behavior](../../../../../pkg/iauthnzimpl/utils.go)
- [Effective-principal snapshot used by enrichment](../../../../../pkg/sys/authnz/impl_enrichprincipaltoken.go)
- [Direct token validation and reissue used by refresh](../../../../../pkg/sys/authnz/impl_refreshprincipaltoken.go)
- [Request-level child Workspace authorization and enrichment coverage](../../../../../pkg/sys/it/impl_childworkspace_test.go)

## Domain design

- [x] update: [prod/auth/context.md](../../../../specs/prod/auth/context.md)
  - update: the RoleGrant Value Object definition and workspace reference to distinguish an exact target Workspace from Workspaces directly owned by the referenced Workspace
  - add: an integer scope discriminator with `RoleScopeWorkspacesOwnedBy = 0` as the default and `RoleScopeWorkspace = 1` and define RoleGrant equality by scope, workspace reference, and qualified role name
  - clarify: PrincipalToken workspace roles carry their own applicability scope, while the API-token flag continues to select restricted principal composition and the existing token-kind and scope combinations remain unchanged

## Technical design

- [x] update: [prod/auth/arch-tokens.md](../../../../specs/prod/auth/arch-tokens.md)
  - update: the principal payload contract to include integer `RoleScope` in `RoleType`, with `RoleScopeWorkspacesOwnedBy = 0` as the Go default and `RoleScopeWorkspace = 1`, referencing the auth context model for scope meanings and grant equality
  - add: shared principal-payload normalization and validation through the ValidateToken payload hook for direct and application-bound token paths, distinguishing missing legacy scope claims from explicit zero, rejecting unknown scope values, and serializing zero explicitly through standard JSON encoding without changing existing JWT audience or identity claims
  - add: API-token issuance and conversion semantics that explicitly select `RoleScopeWorkspace` for new and carried roles without modifying the source payload's role entries
  - update: enrichment descriptions and sequence to retain effective-principal snapshots, choose new grant scopes according to the existing token kind, and deduplicate by scope, WSID, and QName while preserving current parent-to-child propagation
  - update: refresh and mixed-version compatibility descriptions to preserve explicit scopes and infer missing legacy scopes from the original token kind, retaining the existing application, lifetime, and identity behavior

- [x] update: [prod/auth/arch-authz.md](../../../../specs/prod/auth/arch-authz.md)
  - update: the Token-carried role source and principal-composition sequence to match `RoleScopeWorkspace` against the request WSID and `RoleScopeWorkspacesOwnedBy` against the request Workspace descriptor's immediate OwnerWSID, retaining the existing emitted principal WSIDs
  - clarify: the existing API-token and user/device composition stages retain their role sources, descriptor requirements, and early returns, with only existing token-kind and scope combinations emitted in this change
  - reference: the token architecture for scope decoding, legacy compatibility, and validation rules rather than duplicating the payload validation contract

## Construction

### Tests

- [x] update: [pkg/itokens-payloads/impl_test.go](../../../../../pkg/itokens-payloads/impl_test.go)
  - add: role-scope JSON and signed-token round trips through both direct tokens and the application-bound facade, proving that an omitted Go field serializes as explicit numeric zero and both supported scopes survive decoding
  - add: genuine legacy signed-token fixtures with the existing audience and no Scope claims, covering regular-token owned-by defaults and API-token exact-Workspace defaults without generating those fixtures through the new payload validation hook
  - add: token validation coverage that distinguishes an absent Scope from explicit zero, resolves missing scopes per role, preserves explicit scopes, and rejects null, non-integer, negative, and unknown scope values through both direct and application-bound validation
  - add: ValidateToken rejection for unsupported Go scope values in issued tokens and assertions that encoding does not mutate the caller's payload or role entries
  - verify: supported scope meanings survive a legacy role representation that drops Scope before token reissue, reusing the legacy fixture preparation to cover mixed-version compatibility
  - verify: existing identity claims, JWT audience, application binding, role ordering, nil and empty role collections, and unrelated payload types retain their existing serialization behavior, reusing current round-trip assertions

- [x] update: [pkg/iauthnzimpl/impl_test.go](../../../../../pkg/iauthnzimpl/impl_test.go)
  - extend: existing authentication cases with explicit scopes and assertions that exact grants match only the request Workspace and owned-by grants match Workspaces with the referenced immediate owner, including siblings and an unmatched ownership level
  - verify: the API-token branch retains its existing principals without reading Subjects or Workspace descriptors, and regular-token descriptor requirements and System short-circuits remain unchanged
  - add: API-token conversion coverage with carried roles, proving that carried and requested roles have RoleScopeWorkspace and that the source payload and shared backing role entries remain unchanged
  - retain: existing null-WSID and system-role rejection coverage and current principal WSID assertions

- [x] update: [sys/it/impl_childworkspace_test.go](../../../../../pkg/sys/it/impl_childworkspace_test.go)
  - extend: TestForeignAuthorization using its existing VIT setup to inspect emitted scope claims while retaining the current authorization assertions before and after enrichment
  - add: repeated enrichment in the same Workspace with assertions on complete grant tuples and stable grant counts, followed by enrichment at the next ownership level to prove existing propagation behavior
  - add: refresh of an enriched token through the real endpoint, asserting unchanged grant scopes, application, duration, and identity claims and continued child-Workspace access
  - extend: the existing API-token subtest to assert RoleScopeWorkspace and successful access only in the selected Workspace

### Payload contract

- [x] update: [pkg/itokens-payloads/types.go](../../../../../pkg/itokens-payloads/types.go)
  - add: RoleScope as int with RoleScopeWorkspacesOwnedBy first at zero and RoleScopeWorkspace at one
  - add: Scope as the first RoleType field, keeping WSID and QName and serializing Scope even when zero
  - remove: the misleading statement that every role WSID must be an OwnerWSID

- [x] create: [pkg/itokens-payloads/impl_principalpayload.go](../../../../../pkg/itokens-payloads/impl_principalpayload.go)
  - implement: PrincipalPayload.ValidateTokenClaims as the shared scope boundary for the existing generic and application-bound token validation paths, using standard JSON serialization without a payload alias or custom JSON methods
  - include: scope validation for the two defined values during ValidateToken, leaving issuance and explicit numeric-zero encoding to standard JSON serialization
  - include: raw scope claim inspection that preserves Scope claim presence, infers absent legacy scopes from the original IsAPIToken value, retains explicit zero, and rejects malformed or unsupported explicit values
  - preserve: existing field types and JSON behavior for identity claims and role collections, leaving JWT signature, audience, expiry, and application checks in their current owners

- [x] update: [pkg/itokensjwt/impl.go](../../../../../pkg/itokensjwt/impl.go)
  - add: an optional ValidateTokenClaims payload hook after successful JWT verification, audience checking, and payload decoding, forwarding original claim bytes and propagating validation errors without interpreting role scopes

### Authentication and token issuance

- [x] update: [pkg/iauthnzimpl/impl.go](../../../../../pkg/iauthnzimpl/impl.go)
  - update: the existing API-token role loop to match RoleScopeWorkspace against RequestWSID and the regular-token role loop to match RoleScopeWorkspacesOwnedBy against the immediate OwnerWSID
  - preserve: current role sources, principal WSIDs, deduplication, descriptor requirements, and early returns without adding role processing to other composition stages

- [x] update: [pkg/iauthnzimpl/utils.go](../../../../../pkg/iauthnzimpl/utils.go)
  - update: IssueAPIToken to copy carried role entries, assign RoleScopeWorkspace to carried and requested roles, and retain the existing IsAPIToken transition and validation behavior
  - ensure: conversion does not modify source role entries even when the supplied role slice has spare capacity

- [x] update: [authnz/impl_enrichprincipaltoken.go](../../../../../pkg/sys/authnz/impl_enrichprincipaltoken.go)
  - assign: RoleScopeWorkspacesOwnedBy to newly captured regular-token roles and RoleScopeWorkspace to newly captured API-token roles while retaining each effective principal's WSID and QName
  - preserve: existing normalized grants and use complete RoleType equality for deduplication, retaining current endpoint authorization, token lifetime, and propagation behavior

- [x] verify: scoped implementation and regression checks
  - run: `go test ./pkg/itokens-payloads ./pkg/iauthnzimpl ./pkg/sys/authnz`
  - run: `go test ./pkg/sys/it -run '^TestForeignAuthorization$' -count=1`
  - run: `golangci-lint run ./pkg/itokens-payloads/... ./pkg/iauthnzimpl/... ./pkg/sys/authnz/... ./pkg/sys/it/...`
  - run: `git diff --check`
  - result: scoped unit tests, TestForeignAuthorization, and diff checks pass; full scoped lint reports an existing add-constant diagnostic in unchanged pkg/sys/it/test_utils.go:131; lint restricted to changes since HEAD reports zero issues
