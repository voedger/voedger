# Context architecture: prod/auth

Auth context architecture for authenticating subjects, issuing and validating principal tokens, evaluating authorization decisions, and maintaining workspace membership.

## External actors

Roles:

- `@Client`
  - External application or caller using Voedger HTTP APIs.

- `@System`
  - Trusted backend caller using a System Principal Token for internal auth operations.

## Scenarios overview

- **`Authenticate`**
  - Subject establishes an authenticated identity and receives a principal token, including sign-in by login or by active alias and the verifier sub-flow for password reset.

- **`Authorize`**
  - Request processing composes a principal set from four origin-perspective sources — invite-granted roles (`[(cdoc.sys.Subject)]` rows written by `[[Workspace membership]]`), token-carried roles (`PrincipalPayload.Roles` and `GlobalRoles` snapshotted by `[[Authentication]]`), request-context roles (derived at composition time from request state), and anonymous-grants (when no token is presented) — and evaluates ACL rules against it.

- **`Manage tokens`**
  - Principal tokens are issued, refreshed, and validated; refresh preserves the identity payload including the `Login` and `Alias` captured at issue time.

- **`Manage membership`**
  - Invite lifecycle creates subjects doc and joined-workspace records, updates roles, and removes members.

## Components

### Layers

```text
External actors
    |
    +-- @Client
    +-- @System
    |
    v
Auth subsystems
    |
    +-- [[Authentication]]
    +-- [[Authorization]]
    +-- [[Token management]]
    +-- [[Workspace membership]]
    |
    v
Shared concepts
    |
    +-- [(registry.Login)]
    +-- [(cdoc.sys.Subject)]
    +-- [Principal Token]
    +-- [Auth boundary]
```

### Auth subsystems

- `[[Authentication]]`
  - Login creation (including login-alias set/update/clear), sign-in by login or by active alias, password lifecycle, the verifier sub-flow that issues and consumes verified-value tokens, and the profile workspace readiness gate seen by sign-in.
  - Path to file: [arch-authn.md](./arch-authn.md)

- `[[Authorization]]`
  - Runtime ACL evaluation, principal composition from the four sources of effective roles, and the enforcement points exposed to other contexts.
  - Path to file: [arch-authz.md](./arch-authz.md)

- `[[Token management]]`
  - Principal token issue, refresh, and validation, and the principal payload contract shared by authentication and authorization.
  - Path to file: [arch-tokens.md](./arch-tokens.md)

- `[[Workspace membership]]`
  - Invite lifecycle, the subjects doc, joined-workspace records, role updates, and member removal.
  - Path to file: [arch-membership.md](./arch-membership.md)

### Shared concepts

- `[(registry.Login)]`
  - Registry CDoc holding the primary sign-in identifier, password hash, subject kind, profile workspace fields, alias snapshot, and active flag. Produced and consumed by `[[Authentication]]` during sign-in.
  - decl: [pkg/registry/appws.vsql#Login](../../../../pkg/registry/appws.vsql)
  - impl: [pkg/registry/impl_createlogin.go#createLogin](../../../../pkg/registry/impl_createlogin.go)

- `[(cdoc.sys.Subject)]`
  - Workspace-scoped CDoc that grants a login a set of roles in a specific workspace. Produced by `[[Workspace membership]]`, consumed by `[[Authorization]]` on every request.
  - decl: [pkg/sys/invite/consts.go#QNameCDocSubject](../../../../pkg/sys/invite/consts.go)
  - impl: [pkg/sys/invite/impl_applyinviteevents.go](../../../../pkg/sys/invite/impl_applyinviteevents.go)

- `[Principal Token]`
  - Bearer token carrying `PrincipalPayload(Login, Alias, SubjectKind, ProfileWSID, Roles, GlobalRoles, IsAPIToken)`. Produced by `[[Authentication]]` and `[[Token management]]`, consumed by `[[Authorization]]` on every request.
  - decl: [pkg/itokens-payloads/types.go#PrincipalPayload](../../../../pkg/itokens-payloads/types.go)

- `[Auth boundary]`
  - The `pkg/iauthnz` interface that the `apps` context calls at every enforcement point to authenticate a request and obtain its principal set; the only programmatic surface that the `auth` context exposes to other contexts.
  - decl: [pkg/iauthnz/authn-interface.go#IAuthenticator](../../../../pkg/iauthnz/authn-interface.go)
  - impl: [pkg/iauthnzimpl/impl.go](../../../../pkg/iauthnzimpl/impl.go)

## Scenarios

### Authenticate

```text
@Client
  -> [[Authentication]]: sign-in by login or alias
  -> [(registry.Login)]
  -> [[Token management]]: issue [Principal Token]
  -> @Client: principalToken, expiresInSeconds, profileWSID
```

Details: [arch-authn.md](./arch-authn.md), [arch-tokens.md](./arch-tokens.md).

### Authorize

```text
@Client
  -> [Auth boundary]: AuthnRequest(token, requestWSID)
  -> [[Authorization]]: validate [Principal Token], compose principals from token + [(cdoc.sys.Subject)] + ACL-engine-emitted contextual roles
  -> @Client: principals, profileWSID
```

Details: [arch-authz.md](./arch-authz.md).

### Manage membership

```text
@Client (workspace owner) or @Client (invitee)
  -> [[Workspace membership]]: invite, join, update roles, leave, cancel
  -> [(cdoc.sys.Subject)]: roles in the inviting workspace
```

Details: [arch-membership.md](./arch-membership.md).

## Cross-cutting concerns

### Context dependencies

- The `apps` context calls `[Auth boundary]` at every authnz enforcement point in command, query, and actualizer processors; see [../apps/arch.md](../apps/arch.md).
- The `storage` context persists `[(registry.Login)]`, `[(cdoc.sys.Subject)]`, and joined-workspace records through `istructs.IAppStructs`.
- Profile workspace lifecycle (creation, profile-fields write-back) is owned by the `apps` context; `[[Authentication]]` only observes the readiness gate.

### Security

- Every Component in `Auth subsystems` MUST treat passwords, bearer tokens, verification tokens, and verified-value tokens as request secrets and MUST NOT expose password values in response bodies.
- Every `[Principal Token]` MUST be app-scoped and validated through the application token API.

### Testing

- Every externally observable scenario in `Auth subsystems` MUST be covered by integration tests or feature scenarios before behavior changes.
