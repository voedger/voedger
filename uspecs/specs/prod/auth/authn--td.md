# Feature technical design: authn

Technical design for the authentication feature: login creation, sign-in, principal token issue and refresh, profile workspace readiness, and password lifecycle flows. The subsystem architecture for the same scope is in [arch-authn.md](./arch-authn.md); the principal payload contract and refresh semantics (including the alias snapshot captured at issue time) are owned by [arch-tokens.md](./arch-tokens.md); shared concepts (`[(registry.Login)]`, `[Principal Token]`, `[Auth boundary]`) are defined in [arch.md](./arch.md#shared-concepts). This document only adds the HTTP-side surface (routes, handlers, status-code mapping) that is unique to the public authn feature.

## External actors

Roles:

- `@Client`
  - External caller that uses Voedger HTTP APIs or registry-backed authn calls.

- `@System`
  - Trusted backend caller using a System Principal Token for internal registry-backed authn operations.

## Components

### Layers

```text
External callers
    |
    +-- @Client
    |
    v
Public API endpoints
    |
    +-- [/POST /api/v2/apps/{owner}/{app}/users/]
    +-- [/POST /api/v2/apps/{owner}/{app}/devices/]
    +-- [/POST /api/v2/apps/{owner}/{app}/users/change-password/]
    +-- [/POST /api/v2/apps/{owner}/{app}/auth/login/]
    +-- [/POST /api/v2/apps/{owner}/{app}/auth/refresh/]
    |
    v
Router and API dispatch
    |
    +-- [API v2 auth routes]
    |
    v
Authn request handlers
    |
    +-- [User login handler]
    +-- [Device login handler]
    +-- [Password handler]
    +-- [Auth login handler]
    +-- [Auth refresh handler]
    +-- [Device credential generator]
    |
    v
Registry operations
    |
    +-- [/c.registry.CreateEmailLogin/]
    +-- [/c.registry.CreateLogin/]
    +-- [/c.registry.InitiateSetLoginAlias/]
    +-- [/c.registry.PutLoginAliasIndex/]
    +-- [/c.registry.DeactivateLoginAliasIndex/]
    +-- [aproj.registry.ApplySetLoginAlias]
    +-- [/q.registry.IssuePrincipalToken/]
    +-- [/c.registry.ChangePassword/]
    +-- [/q.registry.InitiateResetPasswordByEmail/]
    +-- [/q.registry.IssueVerifiedValueTokenForResetPassword/]
    +-- [/c.registry.ResetPasswordByEmail/]
    |
    v
Token and verification operations
    |
    +-- [Token service]
    +-- [/q.sys.RefreshPrincipalToken/]
    +-- [/q.sys.InitiateEmailVerification/]
    +-- [/q.sys.IssueVerifiedValueToken/]
    +-- [/q.sys.GetCDoc/]
    |
    v
State and workspace lifecycle
    |
    +-- [(registry.Login)]
    +-- [(registry.LoginIdx)]
    +-- [(registry.LoginAlias)]
    +-- [[Profile workspace lifecycle]]
    |     |
    |     +-- [/c.sys.CreateWorkspaceID/]
```

### Public API endpoints

- `[/POST /api/v2/apps/{owner}/{app}/users/]`
  - Public user login creation endpoint. It accepts a verified email token, display name, and password.
  - impl: [pkg/router/impl_apiv2.go#requestHandlerV2_create_user](../../../../pkg/router/impl_apiv2.go)

- `[/POST /api/v2/apps/{owner}/{app}/devices/]`
  - Public device login creation endpoint. It rejects request bodies and returns generated device credentials.
  - impl: [pkg/router/impl_apiv2.go#requestHandlerV2_create_device](../../../../pkg/router/impl_apiv2.go)

- `[/POST /api/v2/apps/{owner}/{app}/users/change-password/]`
  - Public password change endpoint for existing user logins.
  - impl: [pkg/router/impl_apiv2.go#requestHandlerV2_changePassword](../../../../pkg/router/impl_apiv2.go)

- `[/POST /api/v2/apps/{owner}/{app}/auth/login/]`
  - Public sign-in endpoint. It forwards login/password arguments to query processing through `APIPath_Auth_Login`.
  - impl: [pkg/router/impl_apiv2.go#requestHandlerV2_auth_login](../../../../pkg/router/impl_apiv2.go)

- `[/POST /api/v2/apps/{owner}/{app}/auth/refresh/]`
  - Public principal token refresh endpoint. It forwards bearer-token refresh requests through `APIPath_Auth_Refresh`.
  - impl: [pkg/router/impl_apiv2.go#requestHandlerV2_auth_refresh](../../../../pkg/router/impl_apiv2.go)

### Router and API dispatch

- `[API v2 auth routes]`
  - Registers the public authn routes and assigns API path identifiers for the query processor.
  - decl: [pkg/processors/consts.go#APIPath_Auth_Login](../../../../pkg/processors/consts.go)
  - impl: [pkg/router/impl_apiv2.go#registerHandlersV2](../../../../pkg/router/impl_apiv2.go)

### Authn request handlers

- `[User login handler]`
  - Parses user login creation input, validates the verified email token, and
    forwards `registry.CreateEmailLogin` to the registry app.
  - impl: [pkg/router/impl_apiv2.go#requestHandlerV2_create_user](../../../../pkg/router/impl_apiv2.go)

- `[Device login handler]`
  - Rejects non-empty device creation bodies, generates device credentials, and forwards `registry.CreateLogin` with device subject kind.
  - impl: [pkg/router/impl_apiv2.go#requestHandlerV2_create_device](../../../../pkg/router/impl_apiv2.go)

- `[Password handler]`
  - Parses public password-change input and forwards `registry.ChangePassword` with logged and unlogged arguments.
  - impl: [pkg/router/impl_apiv2.go#requestHandlerV2_changePassword](../../../../pkg/router/impl_apiv2.go)

- `[Auth login handler]`
  - Converts public sign-in input into `registry.IssuePrincipalToken`, maps profile workspace readiness to public responses, and formats authn output.
  - decl: [pkg/processors/consts.go#APIPath_Auth_Login](../../../../pkg/processors/consts.go)
  - impl: [pkg/processors/query2/impl_auth_login_handler.go#authLoginHandler](../../../../pkg/processors/query2/impl_auth_login_handler.go)

- `[Auth refresh handler]`
  - Requires an existing bearer token, invokes `sys.RefreshPrincipalToken`, validates the new token, and formats authn output.
  - decl: [pkg/processors/consts.go#APIPath_Auth_Refresh](../../../../pkg/processors/consts.go)
  - impl: [pkg/processors/query2/impl_auth_refresh_handler.go#authRefreshHandler](../../../../pkg/processors/query2/impl_auth_refresh_handler.go)

- `[Device credential generator]`
  - Generates device login and password values used by device login creation.
  - impl: [pkg/coreutils/random.go#DeviceRandomLoginPwd](../../../../pkg/coreutils/random.go)

### Registry operations

- `[/c.registry.CreateEmailLogin/]`
  - Creates a user `[(registry.Login)]` record from a verified email value, rejects collisions with active `[(registry.LoginAlias)]`, and starts profile workspace creation.
  - decl: [pkg/registry/appws.vsql#CreateEmailLogin](../../../../pkg/registry/appws.vsql)
  - impl: [pkg/registry/impl_createlogin.go#execCmdCreateEmailLogin](../../../../pkg/registry/impl_createlogin.go)

- `[/c.registry.CreateLogin/]`
  - Creates a non-email login record, including device logins, after app, login format, duplicate, login-vs-alias collision, and profile-cluster validation.
  - decl: [pkg/registry/appws.vsql#CreateLogin](../../../../pkg/registry/appws.vsql)
  - impl: [pkg/registry/impl_createlogin.go#execCmdCreateLogin](../../../../pkg/registry/impl_createlogin.go)

- `[/c.registry.InitiateSetLoginAlias/]`
  - System-authorized command that resolves the source `[(registry.Login)]`, validates the requested alias format, sets the alias in-progress lock, and records the alias intent for asynchronous application.
  - decl: [pkg/registry/appws.vsql#InitiateSetLoginAlias](../../../../pkg/registry/appws.vsql)
  - impl: [pkg/registry/impl_setloginalias.go#execCmdInitiateSetLoginAlias](../../../../pkg/registry/impl_setloginalias.go)

- `[/c.registry.PutLoginAliasIndex/]`
  - Internal System command that applies alias-vs-login and alias-vs-alias uniqueness in `pseudoWSID(alias)` and inserts or reactivates the active `[(registry.LoginAlias)]` row.
  - decl: [pkg/registry/appws.vsql#PutLoginAliasIndex](../../../../pkg/registry/appws.vsql)
  - impl: [pkg/registry/impl_setloginalias.go#execCmdPutLoginAliasIndex](../../../../pkg/registry/impl_setloginalias.go)

- `[/c.registry.DeactivateLoginAliasIndex/]`
  - Internal System command that deactivates the previous alias index row owned by the source `[(registry.Login)]`.
  - decl: [pkg/registry/appws.vsql#DeactivateLoginAliasIndex](../../../../pkg/registry/appws.vsql)
  - impl: [pkg/registry/impl_setloginalias.go#execCmdDeactivateLoginAliasIndex](../../../../pkg/registry/impl_setloginalias.go)

- `[aproj.registry.ApplySetLoginAlias]`
  - Async projector triggered `AFTER EXECUTE ON c.registry.InitiateSetLoginAlias`; drives cross-workspace alias-index put/deactivate calls and commits `Login.Alias`, `Login.AliasError`, and `Login.AliasInProc`.
  - decl: [pkg/registry/appws.vsql#ApplySetLoginAlias](../../../../pkg/registry/appws.vsql)
  - impl: [pkg/registry/impl_setloginalias.go#applySetLoginAlias](../../../../pkg/registry/impl_setloginalias.go)

- `[/q.registry.IssuePrincipalToken/]`
  - Resolves sign-in by primary login or active alias, validates password and profile readiness, applies TTL policy, and issues principal tokens. On the alias path, it reads `[(registry.LoginAlias)]`, fetches the source `[(registry.Login)]` with `q.sys.GetCDoc`, validates `Login.Alias` against the submitted identifier, and snapshots the canonical login into `Login` and the active alias (empty when none is set) into `Alias`; payload-field semantics are owned by [arch-tokens.md](./arch-tokens.md).
  - decl: [pkg/registry/appws.vsql#IssuePrincipalToken](../../../../pkg/registry/appws.vsql)
  - impl: [pkg/registry/impl_issueprincipaltoken.go#provideIssuePrincipalTokenExec](../../../../pkg/registry/impl_issueprincipaltoken.go)

- `[/c.registry.ChangePassword/]`
  - Validates current password and writes the new password hash.
  - decl: [pkg/registry/appws.vsql#ChangePassword](../../../../pkg/registry/appws.vsql)
  - impl: [pkg/registry/impl_changepassword.go#cmdChangePasswordExec](../../../../pkg/registry/impl_changepassword.go)

- `[/q.registry.InitiateResetPasswordByEmail/]`
  - Resolves password-reset identity by primary login first, then by active `[(registry.LoginAlias)]` in the submitted email's pseudo workspace. Starts email verification for the submitted email and returns the ready profile workspace plus the canonical login pseudo workspace selected for the final reset command.
  - decl: [pkg/registry/appws.vsql#InitiateResetPasswordByEmail](../../../../pkg/registry/appws.vsql)
  - impl: [pkg/registry/impl_resetpassword.go#provideQryInitiateResetPasswordByEmailExec](../../../../pkg/registry/impl_resetpassword.go)

- `[/q.registry.IssueVerifiedValueTokenForResetPassword/]`
  - Exchanges a reset verification code for a verified value token. On the alias path, the verifier proves the submitted alias email and registry re-issues the token with the canonical login as the verified value, keeping the original entity, field, and verification kind.
  - decl: [pkg/registry/appws.vsql#IssueVerifiedValueTokenForResetPassword](../../../../pkg/registry/appws.vsql)
  - impl: [pkg/registry/impl_resetpassword.go#provideIssueVerifiedValueTokenForResetPasswordExec](../../../../pkg/registry/impl_resetpassword.go)

- `[/c.registry.ResetPasswordByEmail/]`
  - Applies a password reset locally after verified value token validation. It writes the password for the login carried by the token value and relies on the client routing the command to the returned canonical pseudo workspace.
  - decl: [pkg/registry/appws.vsql#ResetPasswordByEmail](../../../../pkg/registry/appws.vsql)
  - impl: [pkg/registry/impl_resetpassword.go#cmdResetPasswordByEmailExec](../../../../pkg/registry/impl_resetpassword.go)

### Token and verification operations

- `[Token service]`
  - Token primitives and `PrincipalPayload` are owned by [arch-tokens.md](./arch-tokens.md#token-primitives); referenced here as the layer that issues and validates the tokens carried by the authn HTTP flows.

- `[/q.sys.RefreshPrincipalToken/]`
  - Issues a replacement principal token from the existing principal token payload and duration, preserving identity fields including `Login` and `Alias`.
  - decl: [pkg/sys/sys.vsql#RefreshPrincipalToken](../../../../pkg/sys/sys.vsql)
  - impl: [pkg/sys/authnz/impl_refreshprincipaltoken.go#provideRefreshPrincipalTokenExec](../../../../pkg/sys/authnz/impl_refreshprincipaltoken.go)

- `[/q.sys.InitiateEmailVerification/]`
  - Starts email verification and returns a verification token used by password reset.
  - decl: [pkg/sys/sys.vsql#InitiateEmailVerification](../../../../pkg/sys/sys.vsql)
  - impl: [pkg/sys/verifier/impl.go#provideQryInitiateEmailVerification](../../../../pkg/sys/verifier/impl.go)

- `[/q.sys.IssueVerifiedValueToken/]`
  - Validates a verification code and returns a verified value token.
  - decl: [pkg/sys/sys.vsql#IssueVerifiedValueToken](../../../../pkg/sys/sys.vsql)
  - impl: [pkg/sys/verifier/impl.go#provideQryIssueVerifiedValueToken](../../../../pkg/sys/verifier/impl.go)

- `[/q.sys.GetCDoc/]`
  - Reads a CDoc by ID from the target workspace; alias flows use it to fetch the canonical `[(registry.Login)]` after a local alias-index hit.
  - decl: [pkg/sys/sys.vsql#GetCDoc](../../../../pkg/sys/sys.vsql)
  - impl: [pkg/sys/collection/cdoc_func.go#execQryCDoc](../../../../pkg/sys/collection/cdoc_func.go)

### State and workspace lifecycle

- `[(registry.Login)]`
  - Shared concept; see [arch.md#shared-concepts](./arch.md#shared-concepts). The `AliasInProc` / `Alias` / `AliasError` fields used by the alias commands of this feature are described in [arch-authn.md](./arch-authn.md#sign-in-and-lifecycle-queriescommands).

- `[(registry.LoginIdx)]`
  - Sync-projector-maintained registry view that resolves active login records by application workspace and login hash; produced and consumed by the registry operations of this feature. See [arch-authn.md](./arch-authn.md#registry-records-and-indexes) for the architectural role.
  - decl: [pkg/registry/appws.vsql#LoginIdx](../../../../pkg/registry/appws.vsql)
  - impl: [pkg/registry/impl_createlogin.go#projectorLoginIdx](../../../../pkg/registry/impl_createlogin.go)

- `[(registry.LoginAlias)]`
  - Per-alias registry CDoc used as the alias lookup index. Its uniqueness and reactivation semantics are owned by [arch-authn.md](./arch-authn.md#registry-records-and-indexes); referenced here as the record consumed during sign-in by alias and reset-password-by-alias flows, and written by `[/c.registry.PutLoginAliasIndex/]` / `[/c.registry.DeactivateLoginAliasIndex/]`.

- `[[Profile workspace lifecycle]]`
  - Asynchronous profile workspace creation path triggered by login records and reflected back into `[(registry.Login)]` readiness fields.
  - decl: [pkg/registry/appws.vsql#InvokeCreateWorkspaceID_registry](../../../../pkg/registry/appws.vsql)
  - impl: [pkg/sys/workspace/impl.go#ApplyInvokeCreateWorkspaceID](../../../../pkg/sys/workspace/impl.go)

- `[/c.sys.CreateWorkspaceID/]`
  - Creates the target profile workspace ID for a login-driven profile workspace.
  - decl: [pkg/sys/sys.vsql#CreateWorkspaceID](../../../../pkg/sys/sys.vsql)
  - impl: [pkg/sys/workspace/impl.go#execCmdCreateWorkspaceID](../../../../pkg/sys/workspace/impl.go)

## Scenarios

### Login creation

#### Client creates a user login from a verified email token

```text
@Client
  -> [/POST /api/v2/apps/{owner}/{app}/users/]: verifiedEmailToken, displayName, password
  -> [API v2 auth routes]
  -> [User login handler]
  -> [Token service]: validate verified email token
  -> [/c.registry.CreateEmailLogin/]
  -> [(registry.Login)]
  -> [(registry.LoginIdx)]
  -> [[Profile workspace lifecycle]]
  -> [/c.sys.CreateWorkspaceID/]
  -> @Client: 201 Created
```

#### Client creates a device login

```text
@Client
  -> [/POST /api/v2/apps/{owner}/{app}/devices/]
  -> [API v2 auth routes]
  -> [Device login handler]
  -> [Device credential generator]
  -> [/c.registry.CreateLogin/]: subject kind Device
  -> [(registry.Login)]
  -> [(registry.LoginIdx)]
  -> [[Profile workspace lifecycle]]
  -> @Client: generated login and password
```

#### Login creation rejects duplicate login

```text
@Client
  -> [User login handler]
  -> [/c.registry.CreateEmailLogin/]
  -> [(registry.LoginIdx)]: active login already exists
  -> @Client: 409 Conflict
```

#### Login creation rejects an existing active alias

```text
@Client
  -> [User login handler] or [Device login handler]
  -> [/c.registry.CreateEmailLogin/] or [/c.registry.CreateLogin/]
  -> [(registry.LoginAlias)]: active alias already uses the requested login string
  -> @Client: 409 Conflict
```

#### User login creation rejects malformed request

```text
@Client
  -> [/POST /api/v2/apps/{owner}/{app}/users/]: missing verifiedEmailToken, displayName, or password
  -> [User login handler]
  -> @Client: 400 Bad Request
```

#### Device login creation rejects request body

```text
@Client
  -> [/POST /api/v2/apps/{owner}/{app}/devices/]: non-empty body
  -> [Device login handler]
  -> @Client: 400 Bad Request
```

### Login alias management

#### System sets the first login alias

```text
@System
  -> [/c.registry.InitiateSetLoginAlias/]: Login = jsmith, Alias = j.smith
  -> [(registry.Login)]: set AliasInProc
  -> [aproj.registry.ApplySetLoginAlias]
  -> [/c.registry.PutLoginAliasIndex/]
  -> [(registry.LoginAlias)]: active alias index for j.smith
  -> [(registry.Login)]: Alias = j.smith, AliasInProc = 0, AliasError = ""
```

#### System replaces an existing login alias

```text
@System
  -> [/c.registry.InitiateSetLoginAlias/]: Login = jsmith, Alias = john.smith
  -> [(registry.Login)]: set AliasInProc
  -> [aproj.registry.ApplySetLoginAlias]
  -> [/c.registry.PutLoginAliasIndex/]: create active alias index for john.smith
  -> [/c.registry.DeactivateLoginAliasIndex/]: deactivate old alias index j.smith
  -> [(registry.Login)]: Alias = john.smith, AliasInProc = 0, AliasError = ""
```

#### System clears a login alias

```text
@System
  -> [/c.registry.InitiateSetLoginAlias/]: Login = jsmith, Alias = ""
  -> [(registry.Login)]: set AliasInProc
  -> [aproj.registry.ApplySetLoginAlias]
  -> [/c.registry.DeactivateLoginAliasIndex/]: deactivate old alias index j.smith
  -> [(registry.Login)]: Alias = "", AliasInProc = 0, AliasError = ""
```

#### Alias management rejects caller without System Principal Token

```text
@Client
  -> [/c.registry.InitiateSetLoginAlias/]
  -> @Client: rejected before alias state changes
```

#### Alias creation or update rejects alias-vs-login collision

```text
@System
  -> [/c.registry.InitiateSetLoginAlias/]: Alias = existing-login
  -> [aproj.registry.ApplySetLoginAlias]
  -> [/c.registry.PutLoginAliasIndex/]
  -> [(registry.LoginIdx)]: active login already uses the alias string
  -> [(registry.Login)]: AliasError records conflict when best-effort write succeeds
```

#### Alias creation or update rejects alias-vs-alias collision

```text
@System
  -> [/c.registry.InitiateSetLoginAlias/]: Alias = existing-alias
  -> [aproj.registry.ApplySetLoginAlias]
  -> [/c.registry.PutLoginAliasIndex/]
  -> [(registry.LoginAlias)]: active alias already exists for another source login
  -> [(registry.Login)]: AliasError records conflict when best-effort write succeeds
```

#### Alias creation rejects invalid alias format

```text
@System
  -> [/c.registry.InitiateSetLoginAlias/]: Alias = invalid identifier
  -> [(registry.Login)]: alias format validation fails before AliasInProc is set
  -> @System: 400 Bad Request
```

### Sign-in and profile readiness

#### Subject signs in after profile workspace is ready

```text
@Client
  -> [/POST /api/v2/apps/{owner}/{app}/auth/login/]: login, password
  -> [API v2 auth routes]
  -> [Auth login handler]
  -> [/q.registry.IssuePrincipalToken/]
  -> [(registry.LoginIdx)]
  -> [(registry.Login)]: password hash matches and profileWSID is non-zero
  -> [Token service]: issue principal token
  -> @Client: principalToken, expiresInSeconds, profileWSID
```

#### User signs in with active alias

```text
@Client
  -> [/POST /api/v2/apps/{owner}/{app}/auth/login/]: alias, password
  -> [API v2 auth routes]
  -> [Auth login handler]
  -> [/q.registry.IssuePrincipalToken/]
  -> [(registry.LoginIdx)]: primary-login miss
  -> [(registry.LoginAlias)]: active alias hit
  -> [/q.sys.GetCDoc/]: read source [(registry.Login)] in SourceAppWSID
  -> [(registry.Login)]: Alias equals submitted alias; password hash matches and profileWSID is non-zero
  -> [Token service]: issue principal token with Login (canonical login) and Alias (active alias)
  -> @Client: principalToken, expiresInSeconds, profileWSID
```

#### Sign-in rejects previous alias after alias update

```text
@Client
  -> [Auth login handler]
  -> [/q.registry.IssuePrincipalToken/]: previous alias, password
  -> [(registry.LoginAlias)]: previous alias missing or inactive
  -> @Client: 401 Unauthorized
```

#### Sign-in rejects cleared alias

```text
@Client
  -> [Auth login handler]
  -> [/q.registry.IssuePrincipalToken/]: cleared alias, password
  -> [(registry.LoginAlias)]: cleared alias missing or inactive
  -> @Client: 401 Unauthorized
```

#### Sign-in reports profile workspace not ready

```text
@Client
  -> [Auth login handler]
  -> [/q.registry.IssuePrincipalToken/]
  -> [(registry.Login)]: profileWSID is zero and WSError is empty
  -> [Auth login handler]
  -> @Client: 409 Conflict
```

#### Sign-in reports profile workspace creation error

```text
@Client
  -> [Auth login handler]
  -> [/q.registry.IssuePrincipalToken/]
  -> [(registry.Login)]: WSError is non-empty
  -> [Auth login handler]
  -> @Client: profile workspace creation error
```

### Principal token contract

#### Principal token carries authn identity fields

```text
@Client
  -> [Auth login handler]
  -> [/q.registry.IssuePrincipalToken/]
  -> [(registry.Login)]: login, alias, subject kind, profileWSID
  -> [Token service]: PrincipalPayload(Login, Alias, SubjectKind, ProfileWSID)
  -> @Client: principalToken
```

#### Principal token uses default TTL when no custom TTL is requested

```text
@Client
  -> [Auth login handler]
  -> [/q.registry.IssuePrincipalToken/]: TTLHours omitted or zero
  -> [Token service]: issue token with DefaultPrincipalTokenExpiration
  -> @Client: expiresInSeconds
```

#### Principal token rejects TTL above the maximum

```text
@Client
  -> [Auth login handler]
  -> [/q.registry.IssuePrincipalToken/]: TTLHours above maxTokenTTLHours
  -> @Client: 400 Bad Request
```

#### Client refreshes a principal token

```text
@Client
  -> [/POST /api/v2/apps/{owner}/{app}/auth/refresh/]: existing bearer token
  -> [API v2 auth routes]
  -> [Auth refresh handler]
  -> [/q.sys.RefreshPrincipalToken/]
  -> [Token service]: validate existing token and issue replacement preserving Login, Alias, SubjectKind, ProfileWSID from the input token
  -> [Auth refresh handler]
  -> @Client: new principalToken, expiresInSeconds, profileWSID
```

#### Existing principal token keeps alias snapshot after alias changes

```text
@Client
  -> [Token service]: principal token already issued with Login = jsmith, Alias = j.smith
  -> @System: update or clear login alias
  -> [Token service]: existing token remains valid until normal expiration
  -> @Client: existing token payload still carries Login = jsmith, Alias = j.smith
```

### Password lifecycle

#### Client changes user password

```text
@Client
  -> [/POST /api/v2/apps/{owner}/{app}/users/change-password/]: login, oldPassword, newPassword
  -> [API v2 auth routes]
  -> [Password handler]
  -> [/c.registry.ChangePassword/]
  -> [(registry.LoginIdx)]
  -> [(registry.Login)]: old password matches; write new password hash
  -> @Client: 200 OK
```

#### Client resets password by verified email

```text
@Client
  - all 3 calls routed to sys/registry/pseudoWSID(Email); null auth
  -> [/q.registry.InitiateResetPasswordByEmail/]: AppName, Email, Language
      -> [(registry.LoginIdx)]: GetLoginHash(Email); CDocLoginID
      -> [(registry.Login)]: CDocLoginID; ProfileWSID = Login.WSID
      -> [/q.sys.InitiateEmailVerification/]: at loginApp/ProfileWSID; Email, TargetWSID=ProfileWSID, ForRegistry=true
          - ForRegistry=true: token signed under sys/registry, WSID overridden to pseudoWSID(Email)
          - code emailed via c.sys.SendEmailVerificationCode
      -> out: InitiateResetPasswordByEmailResult
          - VerificationToken: VerificationKind, WSID, ID, Entity, Field, Value, Hash256
            - VerificationKind: EMail
            - Entity: registry.ResetPasswordByEmailUnloggedParams; Field: Email; Value: Email
            - WSID: pseudoWSID(Email)
            - Hash256: hash of the code
          - ProfileWSID
          - CanonicalPseudoWSID = pseudoWSID(Email)

  -> [/q.registry.IssueVerifiedValueTokenForResetPassword/]: AppName, VerificationToken, code, ProfileWSID
      -> [/q.sys.IssueVerifiedValueToken/]: at loginApp/ProfileWSID; VerificationToken, code, ForRegistry=true
          -> [Token service]: validate VerificationToken + hash(code); re-issue under sys/registry, strip Hash256
      -> out: VerifiedValueToken
          - VerifiedValueToken: VerificationKind, WSID, ID, Entity, Field, Value=Email

  -> [/c.registry.ResetPasswordByEmail/]: at CanonicalPseudoWSID; VerifiedValueToken, NewPwd (UNLOGGED), AppName
      -> [(registry.Login)]: login = token.Value (= Email); write PwdHash
      -> out: 200 OK
```

#### Client resets password by verified alias email

```text
@Client
  - steps 1 and 2 routed to sys/registry/pseudoWSID(alias); null auth
  -> [/q.registry.InitiateResetPasswordByEmail/]: AppName, alias, Language
      -> [(registry.LoginIdx)]: GetLoginHash(alias); primary-login miss
      -> [(registry.LoginAlias)]: (AppName, Alias=alias); active alias hit
          - canonicalLogin = LoginAlias.Login
          - SourceAppWSID = LoginAlias.SourceAppWSID
          - CanonicalPseudoWSID = pseudoWSID(canonicalLogin)
      -> [/q.sys.GetCDoc/]: read canonical [(registry.Login)] at SourceAppWSID; ProfileWSID = Login.WSID
      -> [/q.sys.InitiateEmailVerification/]: at loginApp/ProfileWSID; alias, TargetWSID=ProfileWSID, ForRegistry=true
          - code emailed to the alias inbox
          - VerificationToken Value = alias
      -> out: InitiateResetPasswordByEmailResult
          - VerificationToken, ProfileWSID
          - CanonicalPseudoWSID

  -> [/q.registry.IssueVerifiedValueTokenForResetPassword/]: AppName, VerificationToken, code, ProfileWSID
      -> [/q.sys.IssueVerifiedValueToken/]: at loginApp/ProfileWSID; VerificationToken, code, ForRegistry=true
          -> [Token service]: validate VerificationToken + hash(code); VerifiedValueToken Value = alias
      -> [(registry.LoginAlias)]: (AppName, Alias=alias); active alias hit
      -> [Token service]: re-issue registry VerifiedValueToken with original Entity, Field, and VerificationKind; Value = canonicalLogin
      -> out: VerifiedValueToken (Value = canonicalLogin)

  -> [/c.registry.ResetPasswordByEmail/]: at CanonicalPseudoWSID; VerifiedValueToken, NewPwd (UNLOGGED), AppName
      -> [(registry.Login)]: login = token.Value (= canonicalLogin); write PwdHash
      -> out: 200 OK
```

#### Password change rejects malformed request

```text
@Client
  -> [/POST /api/v2/apps/{owner}/{app}/users/change-password/]: missing login, oldPassword, or newPassword
  -> [Password handler]
  -> @Client: 400 Bad Request
```

#### Password change rejects unknown login or wrong current password

```text
@Client
  -> [Password handler]
  -> [/c.registry.ChangePassword/]
  -> [(registry.LoginIdx)]
  -> [(registry.Login)]: login missing or password mismatch
  -> @Client: 401 Unauthorized
```

#### Password reset initiation rejects an inactive alias

```text
@Client
  -> [/q.registry.InitiateResetPasswordByEmail/]: AppName, previous-or-cleared-alias, Language
  -> [(registry.LoginIdx)]: primary-login miss
  -> [(registry.LoginAlias)]: alias row missing or inactive
  -> @Client: 400 Bad Request
```

#### Password reset initiation rejects unknown login

```text
@Client
  -> [/q.registry.InitiateResetPasswordByEmail/]
  -> [(registry.LoginIdx)]: login missing
  -> [(registry.LoginAlias)]: alias row missing or inactive
  -> @Client: 400 Bad Request
```

#### Password reset verification rejects wrong verification code

```text
@Client
  -> [/q.registry.IssueVerifiedValueTokenForResetPassword/]: wrong verification code
  -> [/q.sys.IssueVerifiedValueToken/]
  -> [Token service]: verification token and code do not match
  -> @Client: 400 Bad Request
```

### Exception flows

#### User login creation rejects an invalid verified email token

```text
@Client
  -> [/POST /api/v2/apps/{owner}/{app}/users/]: invalid verifiedEmailToken
  -> [User login handler]
  -> [Token service]: verified value token validation fails
  -> @Client: 400 Bad Request
```

#### Login creation rejects an invalid login name

```text
@Client
  -> [/c.registry.CreateLogin/]: invalid login
  -> [(registry.Login)]: login format validation fails before write
  -> @Client: 400 Bad Request
```

#### Sign-in rejects unknown login or wrong password

```text
@Client
  -> [Auth login handler]
  -> [/q.registry.IssuePrincipalToken/]
  -> [(registry.LoginIdx)]
  -> [(registry.Login)]: login missing or password mismatch
  -> @Client: 401 Unauthorized
```

#### Principal token refresh requires an existing token

```text
@Client
  -> [/POST /api/v2/apps/{owner}/{app}/auth/refresh/]: missing bearer token
  -> [Auth refresh handler]
  -> @Client: 401 Unauthorized
```

## Cross-cutting concerns

This feature inherits context-wide security, consistency, token, and
error-handling rules from [auth architecture](./arch.md).

### Security

- Every Component in `Public API endpoints` and `Authn request handlers` treats passwords, bearer tokens, verification tokens, and verified value tokens as request secrets.
- Every Component in `Registry operations` stores password evidence only through `[(registry.Login)]` password hashes and never returns plaintext passwords except generated device credentials at creation time.
- Every `[Token service]` principal token issue includes the authn identity fields required by this feature: login, canonical login, subject kind, and profile workspace ID.

### Error handling and resilience

- Every Component in `Public API endpoints` maps malformed public request bodies to `400 Bad Request`.
- Every `[Auth login handler]` maps ready profile workspace responses to authn JSON output and maps zero profile workspace ID to `409 Conflict`.
- Every `[Auth refresh handler]` maps a missing bearer token to `401 Unauthorized`.
- Every `[/q.registry.IssuePrincipalToken/]` maps missing login or password mismatch to `401 Unauthorized` and maps TTL above `maxTokenTTLHours` to
  `400 Bad Request`.

### Consistency

- Every `[[Profile workspace lifecycle]]` update is asynchronous; login creation can return `201 Created` before sign-in can issue a usable principal token.
- Every `[(registry.LoginIdx)]` lookup used by sign-in, password change, and primary-login password reset resolves to the active `[(registry.Login)]` for the requested app and login hash.
- Every alias password reset proves the submitted alias email through the verifier, then carries the canonical login in the verified value token so the final password write remains local to `CanonicalPseudoWSID`.
- Every password change or reset updates future credential checks but does not revoke already issued principal tokens.

### Testing

- Every Scenario in [authn.feature](./authn.feature) has integration coverage or must gain integration coverage before authn behavior changes.
- Every login-alias Scenario has registry integration coverage, including alias management, cross-namespace uniqueness, alias sign-in, stale-alias rejection, and token alias snapshots.
- Every password reset by alias Scenario has registry integration coverage for alias-to-canonical reset, alias inbox code delivery, and inactive-alias rejection.
- Every Component in `Public API endpoints` that maps a public status code has API-level integration coverage for success and rejection paths.
- Every token contract Scenario validates the emitted token payload or returned authn response fields at API or registry integration level.
