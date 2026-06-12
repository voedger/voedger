---
change_id: 2606121429-document-enrich-principal-token
type: docs
domains: [prod]
---

# Change request: Document EnrichPrincipalToken in auth specifications

## Why

The `q.sys.EnrichPrincipalToken` query is implemented in `pkg/sys/authnz` and used in production to snapshot the request's runtime-composed role principals into a re-issued `[Principal Token]`, yet it is absent from the auth domain specifications. It spans two subsystems — `[[Token management]]` (mints a new token via `IAppTokens.IssueToken`) and `[[Authorization]]` (the `[Token-carried]` `Roles` source acquires a second population path, beyond sign-in) — and both views are incomplete without it.

## What

Document `EnrichPrincipalToken` across the two auth subsystems it spans:

- In the tokens architecture, add it as a fourth lifecycle scenario alongside Issue / Refresh / Validate, with the same depth of treatment as `RefreshPrincipalToken`
- In the authorization architecture, extend the `[Token-carried]` role-source origin paragraph to record that `EnrichPrincipalToken` is a runtime snapshot path into `PrincipalPayload.Roles` (in addition to sign-in)
- Describe inputs, the payload mutation rule (aggregate `Principal{Kind: Role}` from `processors.IProcessorWorkpiece.GetPrincipals()` into `PrincipalPayload.Roles`, deduplicated by `{WSID, QName}`), the TTL applied, and the basic-auth + `WorkspaceOwner` authorization context under which the query runs

## How

Decisions:

- Place the primary description in `uspecs/specs/prod/auth/arch-tokens.md`: it already documents `RefreshPrincipalToken` (same package, same `IAppTokens.IssueToken` primitive), so by parallel structure the new query belongs as a fourth scenario and a new entry under `Token endpoints`
- Add a cross-cutting note in `uspecs/specs/prod/auth/arch-authz.md` to the `[Token-carried]` role-source paragraph, since this query is the only runtime authz-driven write path into `PrincipalPayload.Roles`; this keeps the four-sources taxonomy honest without duplicating the token-lifecycle description
- Do **not** edit `arch-authn.md` or `authn--td.md`: `EnrichPrincipalToken` is not part of sign-in or any authentication flow they own
- Derive the documented behaviour from `pkg/sys/authnz/impl_enrichprincipaltoken.go` and its registration in `pkg/sys/authnz/provide.go`; describe the payload via the existing `PrincipalPayload` / `RoleType` concepts already named in `arch-tokens.md` and `arch-authz.md` rather than restating field-level details
- Apply `DefaultPrincipalTokenExpiration` consistently with the other token operations, citing `pkg/sys/authnz/consts.go`

Out of scope:

- Any change to runtime behaviour of `EnrichPrincipalToken`, its authorization rules, or its payload shape
- Renaming, relocating, or restructuring existing auth specs beyond the additions above
- Documenting the VIT test helper `VIT.EnrichPrincipalToken`

References:

- [tokens architecture spec](../../../../../uspecs/specs/prod/auth/arch-tokens.md)
- [authorization architecture spec](../../../../../uspecs/specs/prod/auth/arch-authz.md)
- [EnrichPrincipalToken implementation](../../../../../pkg/sys/authnz/impl_enrichprincipaltoken.go)
- [authnz query registration](../../../../../pkg/sys/authnz/provide.go)
- [PrincipalPayload type](../../../../../pkg/itokens-payloads/types.go)
- [DefaultPrincipalTokenExpiration constant](../../../../../pkg/sys/authnz/consts.go)

## Technical design

- [x] update: [prod/auth/arch-tokens.md](../../../../specs/prod/auth/arch-tokens.md)
  - add: `Enrich principal token` entry to `Scenarios overview` as a fourth lifecycle scenario alongside Issue / Refresh / Validate -- `@Client` (basic auth, `WorkspaceOwner`) calls `[q.sys.EnrichPrincipalToken]`, the request's runtime-composed role principals are folded into the decoded `PrincipalPayload.Roles`, and a fresh `[Principal Token]` is minted
  - update: `Layers` diagram to add `[q.sys.EnrichPrincipalToken]` under `Token endpoints`
  - add: `[q.sys.EnrichPrincipalToken]` Component under `Token endpoints` with the same depth as `[q.sys.RefreshPrincipalToken]` -- reads the bearer token via `storages.GetPrincipalTokenFromState`, decodes the payload via `payloads.GetPrincipalPayload`, aggregates every `Principal{Kind: Role}` from `processors.IProcessorWorkpiece.GetPrincipals()` into `PrincipalPayload.Roles` deduplicated by `RoleType{WSID, QName}`, and re-issues through `[IAppTokens].IssueToken` with `DefaultPrincipalTokenExpiration`; cite impl `pkg/sys/authnz/impl_enrichprincipaltoken.go#provideExecQryEnrichPrincipalToken` and decl `pkg/sys/sys.vsql#EnrichPrincipalToken`
  - add: `Enrich principal token` Scenario subsection capturing the request -> decode payload -> aggregate request roles into `Roles` -> `IssueToken` -> response flow
  - update: `Notes` to record that enrich applies `DefaultPrincipalTokenExpiration` consistently with the other token operations

- [x] update: [prod/auth/arch-authz.md](../../../../specs/prod/auth/arch-authz.md)
  - update: `[Token-carried]` role-source `Origin` paragraph to record `[q.sys.EnrichPrincipalToken]` as a runtime snapshot path into `PrincipalPayload.Roles` (in addition to the sign-in snapshot), folding the request's composed `Principal{Kind: Role}` set into the re-issued token; cross-reference `arch-tokens.md` for the token-lifecycle detail
