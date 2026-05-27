# Feature technical design: authn

Technical design for user and device authentication, principal token issue and
refresh, profile workspace readiness, and password lifecycle.

## External actors

Roles:

- `@Client`
  - External application or caller using Voedger HTTP APIs.

## Scenarios overview

- **`Client creates a user login from a verified email token`**
  - Validates a verified email token and creates a user login in the registry.

- **`Client creates a device login`**
  - Generates device credentials and creates a device login in the registry.

- **`Login creation rejects duplicate login`**
  - Registry rejects active duplicate login records.

- **`User login creation rejects malformed request`**
  - Public user creation handler rejects missing required fields.

- **`Device login creation rejects request body`**
  - Public device creation handler rejects bodies because credentials are generated server-side.

- **`Subject signs in after profile workspace is ready`**
  - Registry validates credentials and returns a principal token for ready profiles.

- **`Sign-in reports profile workspace not ready`**
  - Sign-in returns conflict while profile workspace creation has not finished.

- **`Sign-in reports profile workspace creation error`**
  - Sign-in surfaces profile workspace creation failure.

- **`Principal token carries authn identity fields`**
  - Principal token payload contains login, subject kind, and profile workspace ID.

- **`Principal token uses default TTL when no custom TTL is requested`**
  - Token issue applies default principal token expiration.

- **`Principal token rejects TTL above the maximum`**
  - Registry rejects token TTL requests above the configured maximum.

- **`Client refreshes a principal token`**
  - Refresh validates an existing token and issues a replacement with the same authn identity payload.

- **`Client changes user password`**
  - Registry validates the current password and writes a new password hash.

- **`Password change rejects malformed request`**
  - Public password change handler rejects missing or non-string fields.

- **`Password change rejects unknown login or wrong current password`**
  - Registry rejects password changes without valid current credentials.

- **`Client resets password by verified email`**
  - Client initiates email verification, converts code to verified value token, and resets the password.

- **`Password reset initiation rejects unknown login`**
  - Registry rejects reset initiation for missing login.

- **`Password reset verification rejects wrong verification code`**
  - Verification rejects invalid reset codes.

- **`User login creation rejects an invalid verified email token`**
  - User creation rejects a token that cannot be validated as a verified email token.

- **`Login creation rejects an invalid login name`**
  - Registry rejects login names outside the accepted format.

- **`Sign-in rejects unknown login or wrong password`**
  - Registry rejects sign-in without valid credentials.

- **`Principal token refresh requires an existing token`**
  - Refresh rejects requests without a bearer token.

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
Authn handlers
    |
    +-- [Auth login handler]
    +-- [Auth refresh handler]
    +-- [Device credential generator]
    |
    v
Registry and tokens
    |
    +-- [Registry login commands]
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
    +-- [[Profile workspace lifecycle]]
```

### HTTP API

- `[API v2 auth routes]`
  - Handles `/users`, `/devices`, `/users/change-password`, `/auth/login`, and `/auth/refresh`.
  - Path to file: [pkg/router/impl_apiv2.go](../../../../pkg/router/impl_apiv2.go)

### Authn handlers

- `[Auth login handler]`
  - Reads login/password args, calls the registry principal token query, and maps profile readiness responses.
  - Path to file: [pkg/processors/query2/impl_auth_login_handler.go](../../../../pkg/processors/query2/impl_auth_login_handler.go)

- `[Auth refresh handler]`
  - Requires a principal token, calls the app refresh query, validates the returned token, and formats the API response.
  - Path to file: [pkg/processors/query2/impl_auth_refresh_handler.go](../../../../pkg/processors/query2/impl_auth_refresh_handler.go)

- `[Device credential generator]`
  - Generates device login and password values for device login creation.
  - Path to file: [pkg/coreutils/random.go](../../../../pkg/coreutils/random.go)

### Registry and tokens

- `[Registry login commands]`
  - Validate login placement, target app, subject kind, login format, and duplicates, then create `[(registry.Login)]`.
  - Path to file: [pkg/registry/impl_createlogin.go](../../../../pkg/registry/impl_createlogin.go)

- `[Registry principal token query]`
  - Validates login/password, checks profile readiness fields, applies TTL policy, and issues principal tokens.
  - Path to file: [pkg/registry/impl_issueprincipaltoken.go](../../../../pkg/registry/impl_issueprincipaltoken.go)

- `[Registry password commands]`
  - Validates old password and updates the password hash.
  - Path to file: [pkg/registry/impl_changepassword.go](../../../../pkg/registry/impl_changepassword.go)

- `[Registry reset password flow]`
  - Initiates reset verification, exchanges code for a verified value token, and resets the password.
  - Path to file: [pkg/registry/impl_resetpassword.go](../../../../pkg/registry/impl_resetpassword.go)

- `[Token service]`
  - Validates verified value tokens and issues principal tokens with authn identity payload fields.
  - Path to file: [pkg/itokens-payloads/types.go](../../../../pkg/itokens-payloads/types.go)

### State and workspace lifecycle

- `[(registry.Login)]`
  - Stores app name, subject kind, login hash, password hash, profile cluster, profile workspace ID, workspace error, and initialization data.
  - Path to file: [pkg/registry/impl_createlogin.go](../../../../pkg/registry/impl_createlogin.go)

- `[(registry.LoginIdx)]`
  - Resolves active login records by app workspace and login hash.
  - Path to file: [pkg/registry/impl_createlogin.go](../../../../pkg/registry/impl_createlogin.go)

- `[[Profile workspace lifecycle]]`
  - Creates the profile workspace asynchronously and updates login readiness fields.
  - Path to file: [pkg/sys/workspace/impl.go](../../../../pkg/sys/workspace/impl.go)

## Scenarios

### Client creates a user login from a verified email token

```text
@Client
  -> [API v2 auth routes]: verifiedEmailToken, displayName, password
  -> [Token service]: validate verified email token
  -> [Registry login commands]: CreateEmailLogin
  -> [(registry.Login)] and [(registry.LoginIdx)]
  -> [[Profile workspace lifecycle]]
```

### Client creates a device login

```text
@Client
  -> [API v2 auth routes]
  -> [Device credential generator]
  -> [Registry login commands]: CreateLogin with subject kind Device
  -> [(registry.Login)] and [(registry.LoginIdx)]
  -> @Client: generated login and password
```

### Login creation rejects duplicate login

```text
[Registry login commands]
  -> [(registry.LoginIdx)]: resolve existing active login
  -> @Client: 409 Conflict
```

### User login creation rejects malformed request

```text
@Client
  -> [API v2 auth routes]: missing verifiedEmailToken, displayName, or password
  -> @Client: 400 Bad Request
```

### Device login creation rejects request body

```text
@Client
  -> [API v2 auth routes]: non-empty device creation body
  -> @Client: 400 Bad Request
```

### Subject signs in after profile workspace is ready

```text
@Client
  -> [Auth login handler]: login, password
  -> [Registry principal token query]
  -> [(registry.Login)]: password hash matches and profileWSID is non-zero
  -> [Token service]: issue principal token
  -> @Client: principalToken, expiresInSeconds, profileWSID
```

### Sign-in reports profile workspace not ready

```text
[Registry principal token query]
  -> [(registry.Login)]: profileWSID is zero and workspace error is empty
  -> [Auth login handler]
  -> @Client: 409 Conflict
```

### Sign-in reports profile workspace creation error

```text
[Registry principal token query]
  -> [(registry.Login)]: workspace error is non-empty
  -> [Auth login handler]
  -> @Client: profile workspace creation error
```

### Principal token carries authn identity fields

```text
[Registry principal token query]
  -> [(registry.Login)]: login, subject kind, profileWSID
  -> [Token service]: PrincipalPayload(Login, SubjectKind, ProfileWSID)
  -> @Client: principalToken
```

### Principal token uses default TTL when no custom TTL is requested

```text
[Registry principal token query]
  -> [Token service]: issue token with DefaultPrincipalTokenExpiration
  -> @Client: expiresInSeconds
```

### Principal token rejects TTL above the maximum

```text
[Registry principal token query]
  -> [Token service]: requested TTL exceeds max token TTL
  -> @Client: 400 Bad Request
```

### Client refreshes a principal token

```text
@Client
  -> [Auth refresh handler]: existing bearer token
  -> [Token service]: validate existing token and issue replacement
  -> @Client: new principalToken, expiresInSeconds, profileWSID
```

### Client changes user password

```text
@Client
  -> [API v2 auth routes]: login, oldPassword, newPassword
  -> [Registry password commands]
  -> [(registry.Login)]: compare old password hash and write new hash
  -> @Client: 200 OK
```

### Password change rejects malformed request

```text
@Client
  -> [API v2 auth routes]: missing or non-string login, oldPassword, or newPassword
  -> @Client: 400 Bad Request
```

### Password change rejects unknown login or wrong current password

```text
[Registry password commands]
  -> [(registry.LoginIdx)] or [(registry.Login)]: login missing or password mismatch
  -> @Client: 401 Unauthorized
```

### Client resets password by verified email

```text
@Client
  -> [Registry reset password flow]: initiate reset by email
  -> [Token service]: issue verification token
  -> [Registry reset password flow]: verify code and reset password with verified value token
  -> [(registry.Login)]: write new password hash
```

### Password reset initiation rejects unknown login

```text
[Registry reset password flow]
  -> [(registry.LoginIdx)]: login missing
  -> @Client: 400 Bad Request
```

### Password reset verification rejects wrong verification code

```text
[Registry reset password flow]
  -> [Token service]: verification token and code do not match
  -> @Client: 400 Bad Request
```

### User login creation rejects an invalid verified email token

```text
@Client
  -> [API v2 auth routes]: invalid verifiedEmailToken
  -> [Token service]: validation fails
  -> @Client: 400 Bad Request
```

### Login creation rejects an invalid login name

```text
[Registry login commands]
  -> [(registry.Login)]: login format validation fails before write
  -> @Client: 400 Bad Request
```

### Sign-in rejects unknown login or wrong password

```text
[Registry principal token query]
  -> [(registry.LoginIdx)] or [(registry.Login)]: login missing or password mismatch
  -> @Client: 401 Unauthorized
```

### Principal token refresh requires an existing token

```text
@Client
  -> [Auth refresh handler]: missing bearer token
  -> @Client: 401 Unauthorized
```

## Cross-cutting concerns

This feature inherits context-wide security, consistency, token, and error-handling
rules from [auth architecture](./arch.md).

### Feature behavior

- Every Scenario in [authn.feature](./authn.feature) must preserve the documented public status code and response fields.
- Every `[Registry principal token query]` issue must produce the authn identity fields consumed by the public authn response: login, subject kind, and profile workspace ID.
- Every `[Registry reset password flow]` must validate the verified value token before changing `[(registry.Login)]`.

### Password lifecycle

- Every password update affects future credential checks but does not revoke already issued principal tokens.
- Every password reset changes credentials only after the email verification code is exchanged for a verified value token.

### Testing

- Every Scenario in `authn.feature` must have matching integration coverage or trace to existing integration coverage before authn behavior is changed.
- Every feature-specific status-code mapping must be covered at API or registry integration level.
