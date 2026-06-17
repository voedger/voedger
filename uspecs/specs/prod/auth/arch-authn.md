# Context subsystem architecture: prod/auth/authn

Authentication subsystem architecture covering login creation (including login-alias set/update/clear), sign-in by login or by active alias, password lifecycle, the verifier sub-flow that issues and consumes verified-value tokens, and the profile workspace readiness gate seen by sign-in. Context-level overview and shared concepts: [arch.md](./arch.md). Token issue, refresh, validation, and the principal payload contract used at the end of sign-in: [arch-tokens.md](./arch-tokens.md).

## External actors

Roles:

- `@Client`
  - Caller signing in, creating logins, changing or resetting passwords, and running the verifier flow.

- `@System`
  - Caller with a System Principal Token that manages login aliases and global roles on behalf of trusted backend systems.

## Scenarios overview

- **`Create login`**
  - `@Client` calls `[c.registry.CreateEmailLogin]` (user by verified email) with a `[Verified-value token]` and credentials, or `[c.registry.CreateLogin]` (device) with credentials; a `[(registry.Login)]` is persisted, `[(view.LoginIdx)]` is updated, and the profile workspace creation is started by the registry projector.

- **`Sign in by login or by active alias`**
  - `@Client` calls `[q.registry.IssuePrincipalToken]` with a sign-in identifier; `[(registry.Login)]` is resolved either directly or via `[(registry.LoginAlias)]`, the password hash is checked, the `Profile workspace readiness gate` is enforced, and a [Principal Token](./arch-tokens.md) is issued.

- **`Run verifier sub-flow`**
  - `@Client` calls `[q.sys.InitiateEmailVerification]` to receive a verification code by email, then `[q.sys.IssueVerifiedValueToken]` to redeem the code for a `[Verified-value token]` that downstream commands (login creation, password reset) consume as proof of value ownership.

- **`Change or reset password`**
  - `@Client` calls `[c.registry.ChangePassword]` with the current password or `[c.registry.ResetPassword]` with a `[Verified-value token]`; the `[(registry.Login)]` password hash is updated in place.

- **`Manage login alias`**
  - `@System` calls `[c.registry.InitiateSetLoginAlias]`; the registry projector emits `[c.registry.PutLoginAliasIndex]` and `[c.registry.DeactivateLoginAliasIndex]` to publish or retire `[(registry.LoginAlias)]` records and updates the alias snapshot on `[(registry.Login)]`.

## Components

### Layers

```text
External actors
    |
    +-- @Client
    +-- @System
    |
    v
Sign-in and lifecycle queries/commands
    |
    +-- [q.registry.IssuePrincipalToken]
    +-- [c.registry.CreateLogin]
    +-- [c.registry.CreateEmailLogin]
    +-- [c.registry.ChangePassword]
    +-- [c.registry.ResetPassword]
    +-- [c.registry.InitiateSetLoginAlias]
    +-- [c.registry.PutLoginAliasIndex]
    +-- [c.registry.DeactivateLoginAliasIndex]
    +-- [c.registry.UpdateGlobalRoles]
    |
    v
Verifier sub-flow
    |
    +-- [q.sys.InitiateEmailVerification]
    +-- [q.sys.IssueVerifiedValueToken]
    +-- [Verified-value token]
    |
    v
Registry records and indexes
    |
    +-- [(registry.Login)]
    +-- [(registry.LoginAlias)]
    +-- [(view.LoginIdx)]
```

### Sign-in and lifecycle queries/commands

- `[q.registry.IssuePrincipalToken]`
  - Resolves the sign-in identifier against `[(registry.Login)]` directly, then against `[(registry.LoginAlias)]` if the direct lookup misses; verifies the password hash, enforces the `Profile workspace readiness gate`, builds `PrincipalPayload` (setting `Login` to the active-alias snapshot with canonical fallback and `CanonicalLogin` to the canonical login, both captured at issue time) and delegates token issuance to [Token management](./arch-tokens.md). Returns the same `errLoginOrPasswordIsIncorrect` for missing login, deactivated login, missing alias, and wrong password to prevent enumeration.
  - impl: [pkg/registry/impl_issueprincipaltoken.go#provideIssuePrincipalTokenExec](../../../../pkg/registry/impl_issueprincipaltoken.go)

- `[c.registry.CreateLogin]`, `[c.registry.CreateEmailLogin]`
  - Validate the request, consume a `[Verified-value token]` when required, write `[(registry.Login)]`, and trigger profile workspace creation through the registry projector. A deactivated `[(registry.Login)]` with the same login name does not block creation; a fresh `[(registry.Login)]` and profile workspace are produced.
  - impl: [pkg/registry/impl_createlogin.go](../../../../pkg/registry/impl_createlogin.go)

- `[c.registry.ChangePassword]`, `[c.registry.ResetPassword]`
  - Update the password hash on `[(registry.Login)]`. Reset consumes a `[Verified-value token]` issued by the verifier sub-flow in place of the current password.
  - impl: [pkg/registry/impl_changepassword.go](../../../../pkg/registry/impl_changepassword.go), [pkg/registry/impl_resetpassword.go](../../../../pkg/registry/impl_resetpassword.go)

- `[c.registry.InitiateSetLoginAlias]`, `[c.registry.PutLoginAliasIndex]`, `[c.registry.DeactivateLoginAliasIndex]`
  - Authority-gated by `@System` only. `Initiate` updates the alias snapshot on `[(registry.Login)]`; the registry projector emits `Put`/`Deactivate` to maintain `[(registry.LoginAlias)]` so that the next sign-in either resolves to the new alias or fails to resolve the retired one. Alias and primary login share the same uniqueness namespace.
  - impl: [pkg/registry/impl_setloginalias.go](../../../../pkg/registry/impl_setloginalias.go)

- `[c.registry.UpdateGlobalRoles]`
  - Authority-gated by `@System`. Writes the comma-separated `GlobalRoles` field on `[(registry.Login)]` so that the next `[q.registry.IssuePrincipalToken]` snapshots them into `PrincipalPayload.GlobalRoles`. Consumed on every request by [Authorization](./arch-authz.md).
  - impl: [pkg/registry/impl_updateglobalroles.go](../../../../pkg/registry/impl_updateglobalroles.go)

### Verifier sub-flow

- `[q.sys.InitiateEmailVerification]`, `[q.sys.IssueVerifiedValueToken]`
  - Two-step exchange that turns proof of email control into a short-lived `[Verified-value token]` bound to a specific field value. Email delivery is performed asynchronously by `applySendEmailVerificationCode`.
  - impl: [pkg/sys/verifier/provide.go](../../../../pkg/sys/verifier/provide.go), [pkg/sys/verifier/impl.go](../../../../pkg/sys/verifier/impl.go)

- `[Verified-value token]`
  - Bearer token whose claims prove a verification flow has succeeded for one named value. Consumed at most once by the downstream command that requires the proof.
  - decl: [pkg/sys/verifier/consts.go](../../../../pkg/sys/verifier/consts.go)

### Registry records and indexes

- `[(registry.Login)]`
  - Shared concept; see [arch.md#shared-concepts](./arch.md#shared-concepts).

- `[(registry.LoginAlias)]`
  - Per-alias CDoc used during sign-in to resolve an alias identifier to its `[(registry.Login)]`. Inactive entries (cleared or replaced aliases) are skipped by the resolver, making the retired identifier unreachable on the next sign-in.
  - decl: [pkg/registry/appws.vsql#LoginAlias](../../../../pkg/registry/appws.vsql)

- `[(view.LoginIdx)]`
  - Sync-projector-maintained index used to enforce uniqueness on login creation and to locate `[(registry.Login)]` by name within the registry app workspace.
  - decl: [pkg/registry/appws.vsql](../../../../pkg/registry/appws.vsql)
  - impl: [pkg/registry/utils.go](../../../../pkg/registry/utils.go)

## Scenarios

### Sign in by login or by active alias

```text
@Client POST q.registry.IssuePrincipalToken (login | alias, password, appName, ttl)
  -> [q.registry.IssuePrincipalToken]
       -> [(view.LoginIdx)] / [(registry.Login)] direct lookup
       -> on miss: [(registry.LoginAlias)] -> [(registry.Login)]
       -> checkPasswordHash
       -> Profile workspace readiness gate: if ProfileWSID == 0 or WSError != "" return error result
       -> build PrincipalPayload(Login=alias snapshot or canonical login, CanonicalLogin, SubjectKind, ProfileWSID, GlobalRoles)
       -> [Token management].IssueToken (see arch-tokens.md)
  -> @Client: principalToken, profileWSID
```

The login resolution phase (before the readiness gate) returns the same error result for deactivated logins, missing logins, and inactive aliases; the principal token is never issued for a deactivated `[(registry.Login)]`, and a recreated login of the same name resolves to its fresh `[(registry.Login)]` and fresh profile.

### Create login with verifier sub-flow

```text
@Client q.sys.InitiateEmailVerification(email)
  -> [q.sys.InitiateEmailVerification] -> async send code by email
@Client q.sys.IssueVerifiedValueToken(email, code)
  -> [q.sys.IssueVerifiedValueToken] -> [Verified-value token]
@Client c.registry.CreateEmailLogin(verifiedEmailToken, password, displayName)
  -> [c.registry.CreateEmailLogin]
       -> consume [Verified-value token]
       -> [(view.LoginIdx)] uniqueness check
       -> persist [(registry.Login)] (Active=true, password hash)
       -> registry projector triggers profile workspace creation
```

### Manage login alias

```text
@System c.registry.InitiateSetLoginAlias(login, newAlias?)
  -> [c.registry.InitiateSetLoginAlias]
       -> update alias snapshot on [(registry.Login)]
  -> registry projector
       -> [c.registry.DeactivateLoginAliasIndex] on previous alias (if any)
       -> [c.registry.PutLoginAliasIndex] on new alias (if any)
```

## Notes

The verifier sub-flow lives in `pkg/sys/verifier` rather than `pkg/registry`; the authentication subsystem consumes its `[Verified-value token]` output but does not own the email-delivery side. Roles and ACL evaluation of the `@System` caller for alias and global-role commands are owned by [arch-authz.md](./arch-authz.md).
