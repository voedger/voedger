# Context subsystem architecture: prod/auth/tokens

Token management subsystem architecture covering principal token issue, refresh, and validation, and the principal payload contract shared by authentication and authorization. Context-level overview and shared concepts: [arch.md](./arch.md). Producers of `PrincipalPayload`: [arch-authn.md](./arch-authn.md). Consumer that composes principals from a validated token: [arch-authz.md](./arch-authz.md).

## External actors

Roles:

- `@Client`
  - Caller obtaining or refreshing a principal token through the registry and sys APIs.

- `@System`
  - Caller issuing tokens with API-token semantics (`IsAPIToken=true`) for backend integrations and trusted internal callers.

## Scenarios overview

- **`Issue principal token`**
  - At the end of sign-in (see [arch-authn.md](./arch-authn.md#sign-in-by-login-or-by-active-alias)) the authentication subsystem builds `PrincipalPayload` and calls `[ITokens].IssueToken(appQName, ttl, &payload)` to mint a `[Principal Token]`. The `Alias` field captures the active alias snapshot at issue time and is immune to any subsequent change to `[(registry.Login)]` or `[(registry.LoginAlias)]`.

- **`Refresh principal token`**
  - `@Client` calls `[q.sys.RefreshPrincipalToken]` with the current `[Principal Token]`; the payload is decoded, the same identity (including the captured `Alias`) is re-encoded with the same TTL/AppQName by `[ITokens].IssueToken`, and the new token is returned. Refresh never re-resolves the login or alias and never updates the alias snapshot.

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
    |
    v
Token primitives
    |
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
  - Reads the bearer token from request state via `storages.GetPrincipalTokenFromState`, decodes `PrincipalPayload` through `payloads.GetPayloadRegistry`, and re-issues a token for the same AppQName, duration, and payload. The decoded `Alias` is preserved verbatim, so the snapshot taken at the original issue time survives the refresh; alias changes performed in the registry between issue and refresh have no effect on the refreshed token.
  - impl: [pkg/sys/authnz/impl_refreshprincipaltoken.go#provideRefreshPrincipalTokenExec](../../../../pkg/sys/authnz/impl_refreshprincipaltoken.go)

### Token primitives

- `[ITokens]`
  - VVM-level token signer and validator that turns any payload-by-reference into a JWT and back. Audience is derived from the payload type. Returns `ErrTokenExpired`, `ErrInvalidToken`, `ErrInvalidAudience` from `ValidateToken`. Also exposes `CryptoHash256` used by the verifier sub-flow.
  - decl: [pkg/itokens/interface.go#ITokens](../../../../pkg/itokens/interface.go)

- `[IAppTokens]`
  - Per-app facade over `[ITokens]` produced by `payloads.implIAppTokensFactory`. Bind a token to a specific `AppQName` so that `ValidateToken` rejects cross-app reuse. Used by `[Auth boundary]` on every authn enforcement.
  - impl: [pkg/itokens-payloads/types.go#implIAppTokens](../../../../pkg/itokens-payloads/types.go)

### Payload contract

- `[PrincipalPayload]`
  - Identity payload carried by `[Principal Token]`:
    - `Login` - canonical login resolved at issue time
    - `Alias` - snapshot of the active alias at issue time; never re-read on refresh
    - `SubjectKind` - `User` or `Device`
    - `ProfileWSID` - profile workspace of the subject (`NullWSID` for system-only tokens)
    - `Roles []{WSID, QName}` - workspace-scoped roles emitted on every request; for `IsAPIToken=true` these are the only persisted-role source (alongside the implicit `AuthenticatedUser`)
    - `GlobalRoles []QName` - global roles to be applied in any workspace; emitted by `[[Authorization]]` on every request when `IsAPIToken=false`
    - `IsAPIToken` - when true, principal composition emits only `AuthenticatedUser` and `Roles` filtered to `RequestWSID`, and omits the `Host` principal, `GlobalRoles`, the `WorkspaceOwner`/`ProfileOwner`/`WorkspaceDevice` derivations, the `System` short-circuit, and the `[(cdoc.sys.Subject)]` read
  - decl: [pkg/itokens-payloads/types.go#PrincipalPayload](../../../../pkg/itokens-payloads/types.go)

- `[Principal Token]`
  - Shared concept; see [arch.md#shared-concepts](./arch.md#shared-concepts).

## Scenarios

### Issue principal token (end of sign-in)

```text
[q.registry.IssuePrincipalToken] (see arch-authn.md)
  -> build PrincipalPayload{Login, Alias=loginForSignIn.alias (snapshot), SubjectKind, ProfileWSID, GlobalRoles}
  -> [ITokens].IssueToken(appQName, ttl, &payload)
  -> @Client: principalToken, profileWSID
```

### Refresh principal token

```text
@Client POST q.sys.RefreshPrincipalToken (Authorization: Bearer <current token>)
  -> [q.sys.RefreshPrincipalToken]
       -> storages.GetPrincipalTokenFromState(state) -> current token
       -> payloads.GetPayloadRegistry([ITokens], token, &payload) -> decode + appQName + ttl
       -> [ITokens].IssueToken(appQName, ttl, &payload)
  -> @Client: principalToken (new), Alias unchanged from original issue
```

The Alias field is taken verbatim from the decoded payload and is not re-resolved against `[(registry.Login)]` or `[(registry.LoginAlias)]`. A login whose alias was set, replaced, or cleared after the original issue continues to refresh with the snapshotted alias until the next sign-in. To pick up a new alias the caller must sign in again rather than refresh.

### Validate principal token

```text
[Auth boundary] (see arch.md)
  -> [IAppTokens].ValidateToken(token, &payload)
       -> verify signature, audience (AppQName), expiry
       -> decode PrincipalPayload
  -> return to [[Authorization]] for principal composition (see arch-authz.md)
```

## Notes

`DefaultPrincipalTokenExpiration = 1h` is defined in `pkg/sys/authnz/consts.go`; `[q.registry.IssuePrincipalToken]` rejects TTLs above `maxTokenTTLHours`. The TTL is preserved across refresh: each refresh extends the bearer-token wall-clock lifetime by the same duration that the original issue requested.

The `IsAPIToken` branch of principal composition is documented in [arch-authz.md](./arch-authz.md); the token subsystem owns only the flag's presence on `PrincipalPayload`, not the composition rules driven by it.
