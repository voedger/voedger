# Context subsystem architecture: prod/auth/authn

Authentication subsystem architecture covering login creation (including login-alias set/update/clear), canonical Login enablement, sign-in by login or by active alias, password lifecycle, the verifier sub-flow that issues and consumes verified-value tokens, and the profile workspace readiness gate seen by sign-in. Context-level overview and shared concepts: [arch.md](./arch.md). Token issue, refresh, validation, and the principal payload contract used at the end of sign-in: [arch-tokens.md](./arch-tokens.md).

## External actors

Roles:

- `@Client`
  - Caller signing in, creating logins, changing or resetting passwords, and running the verifier flow.

- `@System`
  - Caller with a System Principal Token that manages canonical Login enablement, login aliases, and global roles on behalf of trusted backend systems.

- `@WorkspaceOwner`
  - Caller with the WorkspaceOwner role in a workspace; can read Login CDocs only when that workspace is the request's target registry workspace and cannot mutate canonical enablement or LoginAlias state.

## Scenarios overview

- **`Create login`**
  - `@Client` calls `[c.registry.CreateEmailLogin]` (user by verified email) with a `[Verified-value token]` and credentials, or `[c.registry.CreateLogin]` (device) with credentials; a `[(registry.Login)]` is persisted, `[(view.LoginIdx)]` is updated, and the profile workspace creation is started by the registry projector.

- **`Sign in by login or by active alias`**
  - `@Client` calls `[q.registry.IssuePrincipalToken]` with a sign-in identifier; a direct canonical match must be enabled, while an active `[(registry.LoginAlias)]` match remains usable regardless of canonical enablement. The password hash and `Profile workspace readiness gate` are then checked before a [Principal Token](./arch-tokens.md) is issued.

- **`Manage canonical Login enablement`**
  - `@System` calls `[c.registry.SetCanonicalLoginEnablement]` to set `Enabled` or `Disabled` idempotently. `@System` or `@WorkspaceOwner` of the target registry workspace reads the full Login CDoc through `[q.sys.GetCDoc]` and interprets `CanonicalLoginDisabled` as the enablement state.

- **`Run verifier sub-flow`**
  - `@Client` calls `[q.sys.InitiateEmailVerification]` to receive a verification code by email, then `[q.sys.IssueVerifiedValueToken]` to redeem the code for a `[Verified-value token]` that downstream commands (login creation, password reset) consume as proof of value ownership.

- **`Change or reset password`**
  - `@Client` calls `[c.registry.ChangePassword]` with the current password or `[c.registry.ResetPasswordByEmail]` with a `[Verified-value token]`; the `[(registry.Login)]` password hash is updated in place without consulting canonical enablement. `[q.registry.InitiateResetPasswordByEmail]` checks canonical enablement only when the submitted value resolves directly as the canonical Login.

- **`Manage login alias`**
  - `@System` calls `[c.registry.InitiateSetLoginAlias]`; the registry projector emits `[c.registry.PutLoginAliasIndex]` and `[c.registry.DeactivateLoginAliasIndex]` to publish or retire `[(registry.LoginAlias)]` records and updates the alias snapshot on `[(registry.Login)]`.

## Components

### Layers

```text
External actors
    |
    +-- @Client
    +-- @System
    +-- @WorkspaceOwner
    |
    v
Sign-in and lifecycle queries/commands
    |
    +-- [q.registry.IssuePrincipalToken]
    +-- [q.registry.InitiateResetPasswordByEmail]
    +-- [c.registry.CreateLogin]
    +-- [c.registry.CreateEmailLogin]
    +-- [c.registry.ChangePassword]
    +-- [c.registry.ResetPasswordByEmail]
    +-- [c.registry.SetCanonicalLoginEnablement]
    +-- [c.registry.InitiateSetLoginAlias]
    +-- [c.registry.PutLoginAliasIndex]
    +-- [c.registry.DeactivateLoginAliasIndex]
    +-- [c.registry.UpdateGlobalRoles]
    +-- [q.sys.GetCDoc]
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
  - Resolves the sign-in identifier against `[(registry.Login)]` directly, then against `[(registry.LoginAlias)]` if the direct lookup misses. A direct canonical match must have `CanonicalLoginEnablement=Enabled`; a disabled match returns `errLoginOrPasswordIsIncorrect`. Alias resolution validates the active alias binding but does not inspect canonical enablement. The query then verifies the password hash, enforces the `Profile workspace readiness gate`, builds `PrincipalPayload` (setting `Login` to the canonical login and `Alias` to the active-alias snapshot, empty when no alias, both captured at issue time) and delegates token issuance to [Token management](./arch-tokens.md).
  - impl: [pkg/registry/impl_issueprincipaltoken.go#provideIssuePrincipalTokenExec](../../../../pkg/registry/impl_issueprincipaltoken.go)

- `[q.registry.InitiateResetPasswordByEmail]`
  - Resolves the submitted email directly through `[(view.LoginIdx)]`, then through active `[(registry.LoginAlias)]` on a direct miss. A disabled direct canonical match returns the existing unknown-login response before verifier initiation; an active alias match continues to the verifier flow without inspecting canonical enablement. Later verified-token issue and `[c.registry.ResetPasswordByEmail]` remain unchanged, so a reset initiated before disablement can complete.
  - impl: [pkg/registry/impl_resetpassword.go#provideQryInitiateResetPasswordByEmailExec](../../../../pkg/registry/impl_resetpassword.go)

- `[c.registry.CreateLogin]`, `[c.registry.CreateEmailLogin]`
  - Validate the request, consume a `[Verified-value token]` when required, write `[(registry.Login)]`, and trigger profile workspace creation through the registry projector. A deactivated `[(registry.Login)]` with the same login name does not block creation; a fresh `[(registry.Login)]` and profile workspace are produced.
  - impl: [pkg/registry/impl_createlogin.go](../../../../pkg/registry/impl_createlogin.go)

- `[c.registry.ChangePassword]`, `[c.registry.ResetPasswordByEmail]`
  - Update the password hash on `[(registry.Login)]` without reading `CanonicalLoginEnablement`. Reset consumes a `[Verified-value token]` issued by the verifier sub-flow in place of the current password.
  - impl: [pkg/registry/impl_changepassword.go](../../../../pkg/registry/impl_changepassword.go), [pkg/registry/impl_resetpassword.go](../../../../pkg/registry/impl_resetpassword.go)

- `[c.registry.SetCanonicalLoginEnablement]`
  - Authority-gated by `@System`. Resolves the canonical Login directly and writes `CanonicalLoginEnablement=Enabled|Disabled`; setting the current value succeeds without another state change. The command does not modify `sys.IsActive`, `[(view.LoginIdx)]`, `[(registry.LoginAlias)]`, credentials, profile fields, or memberships.
  - decl: [pkg/registry/appws.vsql](../../../../pkg/registry/appws.vsql)

- `[c.registry.InitiateSetLoginAlias]`, `[c.registry.PutLoginAliasIndex]`, `[c.registry.DeactivateLoginAliasIndex]`
  - Authority-gated by `@System` only. `Initiate` updates the alias snapshot on `[(registry.Login)]`; the registry projector emits `Put`/`Deactivate` to maintain `[(registry.LoginAlias)]` so that the next sign-in either resolves to the new alias or fails to resolve the retired one. Alias and primary login share the same uniqueness namespace.
  - impl: [pkg/registry/impl_setloginalias.go](../../../../pkg/registry/impl_setloginalias.go)

- `[c.registry.UpdateGlobalRoles]`
  - Authority-gated by `@System`. Writes the comma-separated `GlobalRoles` field on `[(registry.Login)]` so that the next `[q.registry.IssuePrincipalToken]` snapshots them into `PrincipalPayload.GlobalRoles`. Consumed on every request by [Authorization](./arch-authz.md).
  - impl: [pkg/registry/impl_updateglobalroles.go](../../../../pkg/registry/impl_updateglobalroles.go)

- `[q.sys.GetCDoc]`
  - Existing whole-CDoc read tagged `WorkspaceOwnerFuncTag`. `@System` is allowed by the System authorization bypass; `@WorkspaceOwner` is allowed only in the target registry workspace. Management callers use the Login CDoc ID to inspect `CanonicalLoginDisabled` and LoginAlias lifecycle fields, while alias resolution uses the same query internally to fetch the source `[(registry.Login)]`.
  - decl: [pkg/sys/sys.vsql#GetCDoc](../../../../pkg/sys/sys.vsql)
  - impl: [pkg/sys/collection/cdoc_func.go](../../../../pkg/sys/collection/cdoc_func.go)

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
       -> on direct hit: require CanonicalLoginEnablement=Enabled
          - Disabled: return errLoginOrPasswordIsIncorrect
       -> on direct miss: [(registry.LoginAlias)] -> [(registry.Login)]
          - require active alias binding; do not read CanonicalLoginEnablement
       -> checkPasswordHash
       -> Profile workspace readiness gate: if ProfileWSID == 0 or WSError != "" return error result
       -> build PrincipalPayload(Login=canonical login, Alias=active alias snapshot, SubjectKind, ProfileWSID, GlobalRoles)
       -> [Token management].IssueToken (see arch-tokens.md)
  -> @Client: principalToken, profileWSID
```

The login resolution phase (before the readiness gate) returns the same error result for disabled canonical logins, deactivated logins, missing logins, inactive aliases, and incorrect passwords. Canonical enablement is a property of the direct route only: an active alias continues through its existing binding to `[(registry.Login)]`.

### Manage canonical Login enablement

```text
@System c.registry.SetCanonicalLoginEnablement(login, Enabled | Disabled)
  -> [c.registry.SetCanonicalLoginEnablement]
       -> [(view.LoginIdx)] / [(registry.Login)]: direct canonical lookup
       -> [(registry.Login)]: write CanonicalLoginEnablement only when its value differs
  -> @System: success

@System q.sys.GetCDoc(CDocLoginID) in the target registry workspace
  -> [q.sys.GetCDoc]
       -> [(registry.Login)]: full Login CDoc
  -> @System: CanonicalLoginDisabled absent or false means Enabled; true means Disabled

@WorkspaceOwner q.sys.GetCDoc(CDocLoginID) in the owned target registry workspace
  -> [q.sys.GetCDoc]
       -> [(registry.Login)]: full Login CDoc
  -> @WorkspaceOwner: CanonicalLoginDisabled and LoginAlias lifecycle fields
```

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

### Initiate password reset by canonical Login or active alias

```text
@Client q.registry.InitiateResetPasswordByEmail(identifier)
  -> [q.registry.InitiateResetPasswordByEmail]
       -> [(view.LoginIdx)] / [(registry.Login)] direct lookup
       -> on direct hit: require CanonicalLoginEnablement=Enabled
          - Disabled: return the existing unknown-login response
       -> on direct miss: [(registry.LoginAlias)] -> [(registry.Login)]
          - require active alias binding; do not read CanonicalLoginEnablement
       -> [q.sys.InitiateEmailVerification]: send verification code to identifier
  -> @Client: verificationToken, profileWSID, canonicalPseudoWSID
```

`[q.registry.IssueVerifiedValueTokenForResetPassword]`, `[q.sys.IssueVerifiedValueToken]`, and `[c.registry.ResetPasswordByEmail]` do not read `CanonicalLoginEnablement`; a flow initiated before canonical disablement therefore completes normally.

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

The verifier sub-flow lives in `pkg/sys/verifier` rather than `pkg/registry`; the authentication subsystem consumes its `[Verified-value token]` output but does not own the email-delivery side. Roles and ACL evaluation of `@System` for mutations and `@WorkspaceOwner` for target-workspace CDoc reads are owned by [arch-authz.md](./arch-authz.md). `[q.sys.GetCDoc]` intentionally returns the full Login CDoc rather than a field-scoped enablement result; disabled direct canonical entry points reuse existing public failures and do not disclose the state.
