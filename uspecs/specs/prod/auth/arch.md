# Context architecture: prod/auth

Auth context architecture for authenticating users and devices, issuing principal
tokens, and maintaining credential lifecycle.

## External actors

Roles:

- `@Client`
  - External application or caller using Voedger HTTP APIs.

- `@System`
  - Trusted backend caller using a System Principal Token for internal authn operations.

## Scenarios overview

- **`Create login`**
  - Client creates a user or device login in the registry and starts profile workspace creation.

- **`Sign in`**
  - Client exchanges login credentials for a principal token once the profile workspace is ready.

- **`Manage login alias`**
  - System sets, updates, or clears a user alias used as an alternative sign-in identifier.

- **`Refresh token`**
  - Client exchanges a valid principal token for a new token with the same authn identity payload.

- **`Manage password`**
  - Client changes or resets user credentials through registry-backed flows.

## Components

### Layers

```text
External callers
    |
    +-- @Client
    |
    v
HTTP API
    |
    +-- [API v2 auth routes]
    |
    v
Authn processing
    |
    +-- [Auth login handler]
    +-- [Auth refresh handler]
    +-- [User login handler]
    +-- [Device login handler]
    +-- [Password handler]
    |
    v
Registry and tokens
    |
    +-- [Registry login commands]
    +-- [Registry alias commands]
    +-- [Alias index projector]
    +-- [Registry principal token query]
    +-- [Registry password commands]
    +-- [Registry reset password flow]
    +-- [Token service]
    |
    v
State and workspace lifecycle
    |
    +-- [(registry.Login)]
    +-- [(registry.LoginIdx)]
    +-- [(registry.LoginAlias)]
    +-- [[Profile workspace lifecycle]]
```

### HTTP API

- `[API v2 auth routes]`
  - Exposes login creation, device creation, password change, sign-in, and token refresh routes.
  - Path to file: [pkg/router/impl_apiv2.go](../../../../pkg/router/impl_apiv2.go)

### Authn processing

- `[Auth login handler]`
  - Converts `/auth/login` request body into a registry principal token query and maps registry readiness into API responses.
  - Path to file: [pkg/processors/query2/impl_auth_login_handler.go](../../../../pkg/processors/query2/impl_auth_login_handler.go)

- `[Auth refresh handler]`
  - Requires an existing principal token, calls the app refresh query, validates the new token, and returns the refreshed response fields.
  - Path to file: [pkg/processors/query2/impl_auth_refresh_handler.go](../../../../pkg/processors/query2/impl_auth_refresh_handler.go)

- `[User login handler]`
  - Validates the verified email token and forwards user login creation to the registry.
  - Path to file: [pkg/router/impl_apiv2.go](../../../../pkg/router/impl_apiv2.go)

- `[Device login handler]`
  - Generates device credentials and forwards device login creation to the registry.
  - Path to file: [pkg/router/impl_apiv2.go](../../../../pkg/router/impl_apiv2.go)

- `[Password handler]`
  - Converts public password-change requests into registry password commands.
  - Path to file: [pkg/router/impl_apiv2.go](../../../../pkg/router/impl_apiv2.go)

### Registry and tokens

- `[Registry login commands]`
  - Create `registry.Login` records, validate login shape and app workspace placement, hash passwords, and reject duplicate logins or collisions with active aliases.
  - Path to file: [pkg/registry/impl_createlogin.go](../../../../pkg/registry/impl_createlogin.go)

- `[Registry alias commands]`
  - System-authorized commands that initiate alias changes, write active alias indexes, and deactivate previous alias indexes across pseudo-workspaces.
  - Path to file: [pkg/registry/impl_setloginalias.go](../../../../pkg/registry/impl_setloginalias.go)

- `[Alias index projector]`
  - Applies login alias intents asynchronously, drives cross-workspace alias index writes, records alias errors, and commits the active alias snapshot on `registry.Login`.
  - Path to file: [pkg/registry/impl_setloginalias.go](../../../../pkg/registry/impl_setloginalias.go)

- `[Registry principal token query]`
  - Resolves sign-in by primary login or active alias, validates credentials, checks profile workspace readiness, and issues principal tokens.
  - Path to file: [pkg/registry/impl_issueprincipaltoken.go](../../../../pkg/registry/impl_issueprincipaltoken.go)

- `[Registry password commands]`
  - Validate current password and update stored password hashes.
  - Path to file: [pkg/registry/impl_changepassword.go](../../../../pkg/registry/impl_changepassword.go)

- `[Registry reset password flow]`
  - Initiates email verification, verifies reset codes, and applies password reset with a verified value token.
  - Path to file: [pkg/registry/impl_resetpassword.go](../../../../pkg/registry/impl_resetpassword.go)

- `[Token service]`
  - Issues and validates app-scoped tokens carrying principal and verified-value payloads.
  - Path to file: [pkg/itokens-payloads/types.go](../../../../pkg/itokens-payloads/types.go)

### State and workspace lifecycle

- `[(registry.Login)]`
  - Registry record holding login app, subject kind, password hash, profile cluster, profile workspace fields, initialization data, and active alias state.
  - Path to file: [pkg/registry/impl_createlogin.go](../../../../pkg/registry/impl_createlogin.go)

- `[(registry.LoginIdx)]`
  - Registry view used to resolve login records by application workspace and login hash.
  - Path to file: [pkg/registry/impl_createlogin.go](../../../../pkg/registry/impl_createlogin.go)

- `[(registry.LoginAlias)]`
  - Active alias lookup index that maps an alternative sign-in identifier to the source `registry.Login` record and snapshots the primary login string.
  - Path to file: [pkg/registry/impl_setloginalias.go](../../../../pkg/registry/impl_setloginalias.go)

- `[[Profile workspace lifecycle]]`
  - Asynchronous workspace creation path triggered by login records and reflected back to the login profile fields.
  - Path to file: [pkg/sys/workspace/impl.go](../../../../pkg/sys/workspace/impl.go)

## Scenarios

### Create login

```text
@Client
  -> [API v2 auth routes]
  -> [User login handler] or [Device login handler]
  -> [Registry login commands]
  -> [(registry.Login)]
  -> [(registry.LoginIdx)]
  -> [[Profile workspace lifecycle]]
```

### Sign in

```text
@Client
  -> [API v2 auth routes]
  -> [Auth login handler]
  -> [Registry principal token query]: resolve primary login or active alias
  -> [(registry.Login)]
  -> [(registry.LoginAlias)]: alias path only
  -> [Token service]
  -> @Client: principalToken, expiresInSeconds, profileWSID
```

### Manage login alias

```text
@System
  -> [Registry alias commands]: initiate set, update, or clear
  -> [(registry.Login)]: mark alias operation in progress
  -> [Alias index projector]
  -> [Registry alias commands]: put new alias index in pseudoWSID(alias)
  -> [Registry alias commands]: deactivate previous alias index in pseudoWSID(old alias)
  -> [(registry.LoginAlias)]
  -> [(registry.Login)]: commit Alias, AliasInProc, AliasError
```

### Refresh token

```text
@Client
  -> [API v2 auth routes]
  -> [Auth refresh handler]
  -> [Token service]: validate existing token and issue replacement
  -> @Client: principalToken, expiresInSeconds, profileWSID
```

### Manage password

```text
@Client
  -> [API v2 auth routes]
  -> [Password handler]
  -> [Registry password commands] or [Registry reset password flow]
  -> [(registry.Login)]
```

## Cross-cutting concerns

### Security

- Every Component in `Authn processing` must treat credentials and verified value tokens as request secrets and must not expose password values in response bodies.
- Every Component in `Registry and tokens` must keep password evidence stored only as salted password hashes.
- Every `[Token service]` issue must be app-scoped and validated through the application token API.
- Authorization policy, ACL evaluation, role resolution, and invite membership are downstream concerns and are not owned by authn architecture.

Token payloads may contain fields later consumed by authorization, but authn owns only identity establishment, token issue/refresh behavior, and identity payload production.

### Error handling and resilience

- Every Component in `Authn processing` maps malformed public request bodies to `400 Bad Request`.
- `[Auth login handler]` maps profile workspace not-ready state to `409 Conflict`.
- `[Auth refresh handler]` maps missing bearer token to `401 Unauthorized`.
- `[Auth login handler]` uses retried federation for principal token issue to tolerate transient local startup failures.

### Consistency

- `[[Profile workspace lifecycle]]` is asynchronous; login creation can succeed before sign-in can issue a usable principal token.
- `[(registry.LoginIdx)]` must be consistent with active `[(registry.Login)]` records for sign-in, password change, and reset lookup.

### Testing

- Every externally observable scenario in this architecture must be covered by integration tests or feature scenarios before implementation behavior is changed.
