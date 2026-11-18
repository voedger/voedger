# Context subsystem architecture: prod/auth/tokens

Token management subsystem architecture covering principal token issue, refresh, enrich, and validation, and the principal payload contract shared by authentication and authorization. Context-level overview and shared concepts: [arch.md](./arch.md). Producers of `PrincipalPayload`: [arch-authn.md](./arch-authn.md). Consumer that composes principals from a validated token: [arch-authz.md](./arch-authz.md).

## External actors

Roles:

- `@Client`
  - Caller obtaining or refreshing a principal token through the registry and sys APIs.

- `@System`
  - Trusted caller issuing application-bound Principal Tokens with scoped grants or converting payloads to API tokens for backend integrations

## Scenarios overview

- **`Issue principal token`**
  - At the end of sign-in (see [arch-authn.md](./arch-authn.md#sign-in-by-login-or-by-active-alias)) the authentication subsystem builds `PrincipalPayload` and calls `[ITokens].IssueToken(appQName, ttl, &payload)` to mint a `[Principal Token]`. The `Login` field carries the canonical primary login (the immutable internal identity); the new `Alias` field carries the active alias snapshot at issue time, or is empty when no alias is set. Both are captured at issue time and are immune to any subsequent change to `[(registry.Login)]` or `[(registry.LoginAlias)]`.

- **`Issue API token`**
  - `@System` uses `[API-token issuer]` to convert the supplied principal payload into an API token with exact-Workspace role grants

- **`Issue regular token with an exact-Workspace grant`**
  - `@System` uses `[IAppTokens]` to preserve a regular token's identity and existing grants while adding an exact-Workspace grant, as detailed in the issuance flow below

- **`Refresh principal token`**
  - `@Client` calls `[q.sys.RefreshPrincipalToken]` with the current `[Principal Token]`; the payload is decoded, the same identity (including the captured `Login` and `Alias`) is re-encoded with the same TTL/AppQName by `[ITokens].IssueToken`, and the new token is returned. Refresh never re-resolves the login or alias and never updates the captured values.

- **`Enrich principal token`**
  - `@Client` (basic auth, `WorkspaceOwner`) calls `[q.sys.EnrichPrincipalToken]` with the current `[Principal Token]`; the payload is decoded, every runtime-composed `Principal{Kind: Role}` for the request is folded into `PrincipalPayload.Roles` (deduplicated by `RoleType{Scope, WSID, QName}`), and a fresh `[Principal Token]` is minted by `[IAppTokens].IssueToken` with `DefaultPrincipalTokenExpiration`. Enrichment preserves existing grants and appends absent grants; the other identity fields are preserved

- **`Validate principal token`**
  - On every request the authorization subsystem calls `[IAppTokens].ValidateToken(token, &payload)` (via `[Auth boundary]`) to verify the signature, audience, and expiry and to decode `PrincipalPayload` for principal composition (see [arch-authz.md](./arch-authz.md)).

## Components

### Layers

```text
External actors
    |
    +-- @Client
    +-- @System
    |
    v
Token endpoints
    |
    +-- [q.sys.RefreshPrincipalToken]
    +-- [q.sys.EnrichPrincipalToken]
    |
    v
Token primitives
    |
    +-- [API-token issuer]
    +-- [ITokens]
    +-- [IAppTokens]
    |
    v
Payload contract
    |
    +-- [PrincipalPayload]
    +-- [Principal Token]
```

### Token endpoints

- `[q.sys.RefreshPrincipalToken]`
  - Reads the bearer token from request state via `storages.GetPrincipalTokenFromState`, decodes `PrincipalPayload` through `payloads.GetPayloadRegistry`, and re-issues a token for the same AppQName and duration. Token validation resolves legacy role scopes before reissue; explicit scopes and all other decoded payload fields are preserved. Alias changes performed in the registry between issue and refresh have no effect on the refreshed token
  - impl: [pkg/sys/authnz/impl_refreshprincipaltoken.go#provideRefreshPrincipalTokenExec](../../../../pkg/sys/authnz/impl_refreshprincipaltoken.go)

- `[q.sys.EnrichPrincipalToken]`
  - Reads the bearer token from request state via `storages.GetPrincipalTokenFromState`, decodes `PrincipalPayload` through `payloads.GetPrincipalPayload`, and snapshots every `Principal{Kind: Role}` from `args.Workpiece.(processors.IProcessorWorkpiece).GetPrincipals()`. Each new grant retains the principal's WSID and QName and receives `RoleScopeWorkspace` for an API token or `RoleScopeWorkspacesOwnedBy` for a regular token. Grants are appended only when the complete `{Scope, WSID, QName}` is absent, and the token is reissued with `DefaultPrincipalTokenExpiration`
  - Runs under the existing `WorkspaceOwner` authorization. The input remains the effective-principal list; existing token grants retain their scopes, and newly captured regular-token roles retain current parent-to-child propagation. Repeated enrichment can therefore add grants for another ownership level; source-grant bookkeeping and delegation-policy changes belong to a separate change
  - impl: [pkg/sys/authnz/impl_enrichprincipaltoken.go#provideExecQryEnrichPrincipalToken](../../../../pkg/sys/authnz/impl_enrichprincipaltoken.go)
  - decl: [pkg/sys/sys.vsql#EnrichPrincipalToken](../../../../pkg/sys/sys.vsql)

### Token primitives

- `[API-token issuer]`
  - Retains the supplied payload's carried roles, converts their scopes to `RoleScopeWorkspace`, and appends the requested roles with the supplied target WSID and the same scope before selecting API-token principal composition. Builds the output role collection without modifying the source payload's entries
  - Preserves the existing rejection of a null target WSID or a requested system role
  - impl: [pkg/iauthnzimpl/utils.go#IssueAPIToken](../../../../pkg/iauthnzimpl/utils.go)

- `[ITokens]`
  - VVM-level token signer and validator that turns any payload-by-reference into a JWT and back. Audience is derived from the payload type. Returns `ErrTokenExpired`, `ErrInvalidToken`, `ErrInvalidAudience` from `ValidateToken`. Also exposes `CryptoHash256` used by the verifier sub-flow.
  - decl: [pkg/itokens/interface.go#ITokens](../../../../pkg/itokens/interface.go)
  - impl: [pkg/itokensjwt/impl.go#ValidateToken](../../../../pkg/itokensjwt/impl.go)

- `[IAppTokens]`
  - Per-app facade over `[ITokens]` produced by `payloads.implIAppTokensFactory`. Bind a token to a specific `AppQName` so that `ValidateToken` rejects cross-app reuse. Used by `[Auth boundary]` on every authn enforcement.
  - impl: [pkg/itokens-payloads/types.go#implIAppTokens](../../../../pkg/itokens-payloads/types.go)

### Payload contract

- `[PrincipalPayload]`
  - Identity payload carried by `[Principal Token]`:
    - `Login` - canonical primary login resolved at issue time: the immutable internal identity used for credentials, routing, authorization, quotas, and metrics; never re-read on refresh
    - `Alias` - active alias snapshot at issue time, empty when no alias is set; never re-read on refresh
    - `SubjectKind` - `User` or `Device`
    - `ProfileWSID` - profile workspace of the subject (`NullWSID` for system-only tokens)
    - `Roles []RoleType` - grants represented by `{Scope RoleScope, WSID istructs.WSID, QName appdef.QName}`; the [auth model](./context.md#rolegrant) defines grant equality and [RoleScope](./context.md#rolescope) defines applicability
    - `GlobalRoles []QName` - global roles to be applied in any workspace; emitted by `[[Authorization]]` on every request when `IsAPIToken=false`
    - `IsAPIToken` - selects restricted principal composition as defined in [Authorization](./arch-authz.md#composition)
  - decl: [pkg/itokens-payloads/types.go#PrincipalPayload](../../../../pkg/itokens-payloads/types.go)
  - impl: [pkg/itokens-payloads/impl_principalpayload.go#ValidateTokenClaims](../../../../pkg/itokens-payloads/impl_principalpayload.go)
  - The payload type and existing claim names remain unchanged so JWT audience derivation and identity claims stay compatible. `RoleScope` is an `int`, with `RoleScopeWorkspacesOwnedBy = 0` first in the enumeration and `RoleScopeWorkspace = 1`; a Go role literal may omit `Scope` to select the owned-by default
  - `PrincipalPayload` uses standard JSON encoding and decoding. After successful JWT verification, audience checking, and payload decoding, `ValidateToken` invokes the optional payload method `ValidateTokenClaims([]byte) error` with the original claim bytes, shared by direct `[ITokens]` calls and the `[IAppTokens]` facade. Scope interpretation stays in `PrincipalPayload`; the generic JWT signer owns signature, audience, expiry, and application metadata
  - Encoding includes the numeric `Scope` claim even for zero; the claim must not use `omitempty`. Encoding does not validate scopes or change the caller's role entries; a token issued with an unsupported scope fails `ValidateToken`
  - `PrincipalPayload.ValidateTokenClaims` inspects raw scope claims to distinguish missing values from explicit zero. An absent legacy claim resolves to `RoleScopeWorkspace` when the original `IsAPIToken` is true and to `RoleScopeWorkspacesOwnedBy` otherwise. An explicit zero retains the owned-by meaning; null, non-integer, and unknown integer values fail token validation

- `[Principal Token]`
  - Shared concept; see [arch.md#shared-concepts](./arch.md#shared-concepts).

## Scenarios

### Issue principal token (end of sign-in)

```text
[q.registry.IssuePrincipalToken] (see arch-authn.md)
  -> build PrincipalPayload{Login=canonical login, Alias=active alias snapshot, SubjectKind, ProfileWSID, GlobalRoles}
  -> [ITokens].IssueToken(appQName, ttl, &payload)
  -> @Client: principalToken, profileWSID
```

### Issue API token

```text
@System
  -> [API-token issuer]: duration, requested roles, target WSID, current principal payload
       -> reject a null target WSID or a requested system role
       -> copy carried role entries and set their Scope = RoleScopeWorkspace
       -> append requested roles with Scope = RoleScopeWorkspace and WSID = target WSID
       -> set IsAPIToken = true on the output payload
       -> [IAppTokens].IssueToken(duration, &payload)
            -> [PrincipalPayload]: validate and encode role scopes
            -> [ITokens]: sign for the application and requested lifetime
  -> @System: API token
```

### Issue regular token with an exact-Workspace grant

```text
@System
  -> [IAppTokens].ValidateToken(current regular Principal Token, &payload)
       -> verify application binding, signature, audience, expiry, and payload claims
  -> require payload.IsAPIToken == false
  -> retain Login, Alias, SubjectKind, ProfileWSID, GlobalRoles, and existing Roles
  -> append RoleType{Scope: RoleScopeWorkspace, WSID: target Workspace, QName: granted role}
  -> [IAppTokens].IssueToken(requested lifetime, &payload)
       -> [ITokens]: sign for the bound application and requested lifetime
  -> @System: regular Principal Token with IsAPIToken still false
```

This flow uses the trusted in-process token service; it adds no public issuance endpoint or Subject membership. Effective grant matching and the additional permissions retained by regular tokens are defined in [Authorization](./arch-authz.md#role-sources).

### Refresh principal token

```text
@Client POST q.sys.RefreshPrincipalToken (Authorization: Bearer <current token>)
  -> [q.sys.RefreshPrincipalToken]
       -> storages.GetPrincipalTokenFromState(state) -> current token
       -> payloads.GetPayloadRegistry([ITokens], token, &payload) -> decode and normalize scopes + appQName + ttl
       -> [ITokens].IssueToken(appQName, ttl, &payload) -> preserve scopes and encode zero explicitly
  -> @Client: principalToken (new), Login and Alias unchanged from original issue
```

The `Login` and `Alias` fields are taken verbatim from the decoded payload and are not re-resolved against `[(registry.Login)]` or `[(registry.LoginAlias)]`. A login whose alias was set, replaced, or cleared after the original issue continues to refresh with the snapshotted `Alias` until the next sign-in. To pick up a new alias the caller must sign in again rather than refresh.

### Enrich principal token

```text
@Client POST q.sys.EnrichPrincipalToken (Authorization: Bearer <current token>, WorkspaceOwner)
  -> [q.sys.EnrichPrincipalToken]
       -> storages.GetPrincipalTokenFromState(state) -> current token
       -> payloads.GetPrincipalPayload(appTokens, token) -> decode payload and normalize scopes
       -> for each Principal{Kind: Role} in IProcessorWorkpiece.GetPrincipals():
            Scope = RoleScopeWorkspace if IsAPIToken, otherwise RoleScopeWorkspacesOwnedBy
            append RoleType{Scope, WSID: Principal.WSID, QName: Principal.QName} if absent
       -> [IAppTokens].IssueToken(DefaultPrincipalTokenExpiration, &payload)
  -> @Client: EnrichedToken
```

### Validate principal token

```text
[Auth boundary] (see arch.md)
  -> [IAppTokens].ValidateToken(token, &payload)
       -> verify signature, audience (AppQName), expiry
       -> decode PrincipalPayload and normalize and validate role scopes
  -> return to [[Authorization]] for principal composition (see arch-authz.md)
```

## Notes

`DefaultPrincipalTokenExpiration = 1h` is defined in `pkg/sys/authnz/consts.go`; `[q.registry.IssuePrincipalToken]` rejects TTLs above `maxTokenTTLHours`. The TTL is preserved across refresh: each refresh extends the bearer-token wall-clock lifetime by the same duration that the original issue requested. Enrich does not preserve the input token's remaining lifetime; it mints with `DefaultPrincipalTokenExpiration`, consistent with a freshly issued token.

The `IsAPIToken` branch of principal composition is documented in [arch-authz.md](./arch-authz.md); the token subsystem owns only the flag's presence on `PrincipalPayload`, not the composition rules driven by it.

## Cross-cutting concerns

### Versioning and compatibility

- Every trusted issuer of regular tokens with `RoleScopeWorkspace` grants requires all nodes that validate those tokens to support this combination and all refresh and enrichment nodes to preserve explicit scopes before issuance begins
- Every API-token conversion and enrichment path retains its existing scope selection policy; trusted regular-token issuance can additionally carry exact-Workspace grants as defined by the [auth model](./context.md#principaltoken)
- Every principal-token validation path uses the shared payload validation hook, including authentication, refresh, and enrichment

Legacy tokens without a Scope claim retain the token-kind defaults specified in the payload contract. Omitting Scope remains compatible only with API-token exact-Workspace grants and regular-token owned-by grants. Dropping Scope from a regular-token exact-Workspace grant changes its meaning to owned-by; scope-preserving serialization alone is also insufficient when a validator still ignores exact-Workspace grants on regular tokens. Deploy support across the relevant validation, refresh, and enrichment nodes before issuing this combination. Incompatible nodes must not process these tokens while they remain valid, including during rollback.
