# Context subsystem architecture: prod/auth/authz

Authorization subsystem architecture covering runtime ACL evaluation, the enforcement points exposed to other contexts, and the four sources of effective roles composed for every request. Context-level overview and shared concepts: [arch.md](./arch.md). Identity payload contract consumed at validation time: [arch-tokens.md](./arch-tokens.md). Origin of the subjects doc and joined-workspace records consumed during composition: [arch-membership.md](./arch-membership.md).

## External actors

Roles:

- `@Client`
  - Caller whose request flows through `[Auth boundary]` at every authnz enforcement point in command, query, and actualizer processors.

Systems:

- `*apps`
  - The `apps` context (command, query, and actualizer processors) is the only programmatic consumer of `[Auth boundary]`; it calls `Authenticate` before invoking the ACL engine. See [../apps/arch.md](../apps/arch.md).

## Scenarios overview

- **`Authenticate request`**
  - `*apps` calls `[Auth boundary].Authenticate(ctx, app, appTokens, AuthnRequest{Host, RequestWSID, Token})`; the subsystem validates the token via [Token management](./arch-tokens.md), composes principals from the four role sources, and returns `(principals, profileWSID)` for ACL evaluation.

- **`Compose principals - anonymous`**
  - When the request has no token, the `Guest` user, the `Anonymous` role, the `Host` principal, and any `[(cdoc.sys.Subject)]` rows that match `sys.Guest` in `RequestWSID` are emitted.

- **`Compose principals - API token`**
  - When `PrincipalPayload.IsAPIToken=true`, principal composition uses only `Roles` filtered to `RequestWSID` plus `AuthenticatedUser`; no Host, no Subject, no derived workspace roles.

- **`Compose principals - user or device token`**
  - When `IsAPIToken=false`, the subsystem emits `AuthenticatedUser`, the `Host` principal, all four role sources, and the workspace-derived roles (`ProfileOwner` / `WorkspaceOwner` / `WorkspaceDevice`) appropriate to the `RequestWSID`.

## Components

### Layers

```text
External actors
    |
    +-- @Client (indirectly via *apps)
    +-- *apps
    |
    v
Boundary
    |
    +-- [Auth boundary] (IAuthenticator)
    |
    v
Composition
    |
    +-- [Token validator]
    +-- [Subjects reader]
    +-- [Workspace role deriver]
    +-- [Role inheritance map]
    |
    v
Role sources
    |
    +-- [Invite-granted]    (cdoc.sys.Subject in RequestWSID)
    +-- [Token-carried]     (PrincipalPayload.Roles + GlobalRoles)
    +-- [Request-context]   (Host, AuthenticatedUser, ProfileOwner, WorkspaceOwner, WorkspaceDevice, System)
    +-- [Anonymous-grants]  (Anonymous, Guest + sys.Guest subjects-doc rows)
```

### Boundary

- `[Auth boundary]`
  - Shared concept; see [arch.md#shared-concepts](./arch.md#shared-concepts). The single programmatic surface exposed to `*apps`; implemented by `implIAuthenticator.Authenticate`.

### Composition

- `[Token validator]`
  - When `Token` is non-empty, calls `[IAppTokens].ValidateToken(token, &PrincipalPayload)` (see [arch-tokens.md](./arch-tokens.md)). Failures short-circuit composition with the validation error.
  - impl: [pkg/iauthnzimpl/impl.go#Authenticate](../../../../pkg/iauthnzimpl/impl.go)

- `[Subjects reader]`
  - Reads `[(cdoc.sys.Subject)]` from `RequestWSID` for the resolved login (or `sys.Guest` when anonymous) through the injected `subjectRolesGetter`, and emits one `Principal{Kind: Role, WSID: RequestWSID, QName: role}` per matched role.
  - impl: [pkg/iauthnzimpl/impl.go#rolesFromSubjects](../../../../pkg/iauthnzimpl/impl.go)

- `[Workspace role deriver]`
  - Reads the workspace descriptor (`appdef.QNameCDocWorkspaceDescriptor`) for `RequestWSID`, compares its `OwnerWSID` against `ProfileWSID`, and emits `ProfileOwner` (same), `WorkspaceOwner` (user owner match), or `WorkspaceDevice` (device, per app's `isDeviceAllowed`); always emits `Host` (unless API token) and `AuthenticatedUser` (when token is present).
  - impl: [pkg/iauthnzimpl/impl.go#Authenticate](../../../../pkg/iauthnzimpl/impl.go)

- `[Role inheritance map]`
  - Static map (`ProfileOwner -> WorkspaceOwner`, `WorkspaceDevice -> WorkspaceOwner`, `RoleWorkspaceOwner -> WorkspaceOwner`) applied during ACL evaluation so that a single emitted role grants its parent role's privileges.
  - decl: [pkg/iauthnz/authn-types.go#rolesInheritance](../../../../pkg/iauthnz/authn-types.go)

### Role sources

Four sources by origin contribute to a request's principal set; each is enumerated below with the read path used by `[Workspace role deriver]`.

- `[Invite-granted]`
  - Origin: `[[Workspace membership]]` writes `[(cdoc.sys.Subject)]` rows in the inviting workspace as part of the join flow; see [arch-membership.md](./arch-membership.md).
  - Composition: `[Subjects reader]` reads `[(cdoc.sys.Subject)]` rows in `RequestWSID` matching the resolved login and emits one `Principal{Kind: Role, WSID: RequestWSID, QName: role}` per matched role.

- `[Token-carried]`
  - Origin: `[[Authentication]]` snapshots roles into `PrincipalPayload` at sign-in (per-app `Roles` from the registry login record) and `[c.registry.UpdateGlobalRoles]` updates the `GlobalRoles` field consumed at the next sign-in; see [arch-authn.md](./arch-authn.md).
  - Composition: `PrincipalPayload.Roles []{WSID, QName}` is filtered to `RequestWSID` for API tokens, or to `OwnerWSID` of `RequestWSID` and re-keyed to `RequestWSID` for user/device tokens; `PrincipalPayload.GlobalRoles []QName` is emitted in any workspace as `Principal{Kind: Role, WSID: RequestWSID, QName: role}` when `IsAPIToken=false`.

- `[Request-context]`
  - Origin: computed at composition time from request state (`RequestWSID`, `Host`, token presence, `ProfileWSID`, workspace descriptor's `OwnerWSID`, app-supplied `isDeviceAllowed`), not from any persisted record.
  - Composition: `Host` (unless API token), `AuthenticatedUser` (when token is present), `ProfileOwner` (when `RequestWSID==ProfileWSID`), `WorkspaceOwner` (user, descriptor's `OwnerWSID==ProfileWSID`), `WorkspaceDevice` (device, `isDeviceAllowed` true), and the `System` short-circuit (when the token carries `WorkspaceOwner.System` or when `ProfileWSID==NullWSID`).

- `[Anonymous-grants]`
  - Origin: emitted when no token is presented; combines the reserved `Anonymous` role and the `Guest` user (`sys.Guest @ GuestWSID`) defined in [pkg/iauthnz/authn-types.go](../../../../pkg/iauthnz/authn-types.go) with any `[(cdoc.sys.Subject)]` rows in `RequestWSID` that match `sys.Guest`.
  - Composition: `Anonymous` and `Guest` are added unconditionally on the no-token branch; the `[Subjects reader]` is invoked with the `sys.Guest` login so any guest-bound role grants persisted by `[[Workspace membership]]` apply.

VSQL role inheritance (e.g., `ProfileOwner -> WorkspaceOwner`) is expanded later by the ACL engine, not during composition; see `[Role inheritance map]` above.

## Scenarios

### Authenticate request and compose principals

```text
*apps (command/query/actualizer processor)
  -> [Auth boundary].Authenticate(ctx, app, appTokens, AuthnRequest{Host, RequestWSID, Token})
       -> if Token == "":
            emit [Anonymous-grants]: User(sys.Guest @ GuestWSID), Role(Anonymous)
            emit [Anonymous-grants] via [Subjects reader] for sys.Guest in RequestWSID
            emit [Request-context]: Host
            return (principals, NullWSID)
       -> [Token validator]: appTokens.ValidateToken(Token, &PrincipalPayload)
       -> emit [Request-context]: Role(AuthenticatedUser @ RequestWSID)
       -> if PrincipalPayload.IsAPIToken:
            emit [Token-carried] PrincipalPayload.Roles filtered to RequestWSID
            return (principals, NullWSID)
       -> emit [Token-carried] PrincipalPayload.GlobalRoles in RequestWSID
       -> emit [Invite-granted] via [Subjects reader] for PrincipalPayload.Login in RequestWSID
       -> profileWSID = PrincipalPayload.ProfileWSID
       -> if Role(System) already in principals: return
       -> emit User|Device principal at profileWSID
       -> read cdoc.WorkspaceDescriptor in RequestWSID
       -> emit [Request-context]: ProfileOwner (if RequestWSID==profileWSID), WorkspaceOwner (user, OwnerWSID==profileWSID), WorkspaceDevice (device, app's isDeviceAllowed)
       -> emit [Token-carried] PrincipalPayload.Roles filtered to OwnerWSID of RequestWSID
       -> emit [Request-context]: Host
       -> return (principals, profileWSID)
```

### ACL evaluation by \*apps

```text
*apps (command/query processor)
  -> [Auth boundary].Authenticate -> principals
  -> appdef ACL engine: evaluate rule.AllowedRoles vs principals (applying [Role inheritance map])
  -> on deny: 403; on allow: proceed to extension
```

## Notes

The `System` short-circuit (line ~119 of `pkg/iauthnzimpl/impl.go`) returns immediately when the token carries `WorkspaceOwner.System` in `Roles`: a system caller is not augmented with subject or workspace-owner roles. The `profileWSID == NullWSID` short-circuit additionally emits `System` for any user-kind subject with no profile (used by registry-app internal flows).

`Anonymous`, `Guest`, `Everyone`, and `AuthenticatedUser` come from `pkg/iauthnz/authn-types.go` and are reserved names; ACL rules in `apps` reference these QNames to express coverage levels.
