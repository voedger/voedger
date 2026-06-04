---
change_id: 2605281424-login-alias-feature
type: feat
issue_url: https://untill.atlassian.net/browse/AIR-4113
---

# Change request: Login alias for user sign-in

Resolves:

- [AIR-4113: change-email: voedger: login aliase feature](./issue-AIR-4113.md)

## Why

Customers occasionally need to change the email address tied to their account (for example after a domain or personal email change) without losing access through the original login. A managed login alias lets users continue to sign in with either identifier during such transitions and supports broader account-identity workflows in the unTill Air context.

## What

Introduce a per-user login alias managed within the identity/authentication domain:

- A user can have at most one login alias, which can be created, updated, or cleared; alias-management operations require a System Principal Token (the `System` role)
- An alias uses the same sign-in identifier format and exact case-sensitive matching rules as a primary login; no alias-specific normalization is applied
- An alias is globally unique across the union of all user logins and aliases; an attempt to set an alias that collides with any existing login or alias is rejected
- An alias is active for sign-in whenever it is set; while set, sign-in succeeds using either the original login or the alias, and clearing the alias is the only way to disable it
- The issued authentication token always carries both the original login and an alias field; when no alias is set the alias field is empty
- The alias claim is a snapshot copied into issued principal tokens; refreshing a principal token preserves the alias value from the input token and does not re-read current alias state; updating or clearing the alias afterwards does not revoke or rewrite previously issued tokens, which remain valid until their normal expiration

## How

### Background (How)

```sql
VIEW LoginIdx (
    AppWSID int64 NOT NULL,
    AppIDLoginHash text NOT NULL,
    CDocLoginID ref(Login) NOT NULL,
    PRIMARY KEY((AppWSID), AppIDLoginHash)
) AS RESULT OF ProjectorLoginIdx;

SYNC PROJECTOR ProjectorLoginIdx AFTER INSERT ON Login INTENTS(sys.View(LoginIdx));
```

CreateLogin vs CreateEmailLogin:

- Both call the same createLogin helper and produce the same cdoc.registry.Login; they differ only in where the login string comes from and what it guarantees
- CreateEmailLogin takes it from the Email field and requires a verified email (the caller proved ownership via email verification) - used for normal email sign-up
- CreateLogin takes it from the Login field with no verification - used for non-email logins, including device logins. Since both share createLogin, your uniqueness precheck added there covers both at once

### Modifications

One section per object/aspect. Each section states the status (new / extended / behavior-updated / unchanged), the fields or signature, and the behavior and data-structure usage rules.

#### Sign-in identifier uniqueness rules

A sign-in identifier is either a primary login or an alias. The combined namespace is unique per `AppName`. Both write paths (`c.registry.CreateLogin` / `c.registry.CreateEmailLogin` for logins, the alias-index write triggered by `c.registry.InitiateSetLoginAlias` for aliases) apply the rules below symmetrically. Pseudo-WSID routing co-locates both indexes for any given identifier string, so each rule is a single workspace-local read.

Aliases reuse the existing primary-login validation and matching contract from `createLogin`: the same accepted characters and rejected forms apply, including the current lowercase-only letter set from `validLoginRegexp`; identifiers starting or ending with `-`, `.`, or space, containing `..`, or starting with `sys.` are rejected. No case folding or trimming is performed before routing, hashing, uniqueness checks, storage, or sign-in lookup. As a result, alias uniqueness and sign-in matching compare the exact submitted string after validation.

| Rule           | Enforced by                                                                                                     |
|----------------|-----------------------------------------------------------------------------------------------------------------|
| Login-vs-login | precheck against `view.registry.LoginIdx` in `pseudoWSID(login)`                                                |
| Login-vs-alias | precheck against active `cdoc.registry.LoginAlias` unique combination `(AppName, login)` in `pseudoWSID(login)` |
| Alias-vs-login | precheck against `view.registry.LoginIdx` in `pseudoWSID(alias)`                                                |
| Alias-vs-alias | precheck against active `cdoc.registry.LoginAlias` unique combination `(AppName, alias)` in `pseudoWSID(alias)` |
|                | the `UNIQUE (AppName, Alias)` constraint on active rows is the final guard against races                        |

Implementation note: the four rows above are exercised through a single Go helper (working name `assertIdentifierAvailable(state, appName, identifier, targetAppWSID)`) called from `createLogin` and the alias-index write helper in `pkg/registry`. The helper picks the relevant pair of branches per call site -- login writes apply rows 1-2, alias index writes apply rows 3-4 -- so a row addition or wording change is picked up by both paths without duplication.

#### extended: `cdoc.registry.Login`

New fields parallel today's `WSID` / `WSError`. The requested alias value is not stored on `Login`; it lives in the `InitiateSetLoginAlias` command argument and flows through the projector into `LoginAlias`.

````sql
ALTER WORKSPACE sys.AppWorkspaceWS (
    TABLE Login INHERITS sys.CDoc (
        -- ... existing fields ...
        AliasInProc int32,             -- in-progress lock; non-zero while an alias write is in flight
        Alias varchar,                 -- committed alias snapshot; empty when no alias is set
        AliasError varchar(1024)       -- last alias-write failure reason; empty on success
    );
);
````

#### new: `cdoc.registry.LoginAlias`

Lives in `sys.AppWorkspaceWS` next to `Login`. `CDocLoginID` is a cross-workspace pointer (the target `Login` lives in `SourceAppWSID`), so it is `int64`, not VSQL `ref(Login)`. Active `LoginAlias` rows are the alias lookup index; previous alias rows are deactivated with `sys.IsActive = false` and can be reactivated when the same alias value is assigned again. `Login.Alias` is the authoritative active alias value. Each active row also snapshots the plaintext primary login in `LoginAlias.Login`, which is the source for `PrincipalPayload.Login` on alias sign-in.

````sql
ALTER WORKSPACE sys.AppWorkspaceWS (
    TABLE LoginAlias INHERITS sys.CDoc (
        AppName varchar NOT NULL,
        SourceAppWSID int64 NOT NULL,  -- workspace of the source Login record
        CDocLoginID int64 NOT NULL,    -- record id of the source Login in SourceAppWSID
        Login varchar NOT NULL,        -- plaintext primary login snapshot; source for PrincipalPayload.Login on alias sign-in
        Alias varchar NOT NULL,
        UNIQUE (AppName, Alias)        -- active alias-vs-alias uniqueness
    );
);
````

The `UNIQUE (AppName, Alias)` constraint is enforced for active rows via the generic uniques mechanism (`pkg/sys/uniques/impl.go`). All `LoginAlias` rows with the same `Alias` land in the same workspace (`GetPseudoWSID(NullWSID, Alias, clusterID)`), so the constraint scope is correct. The cross-namespace rules (login-vs-alias, alias-vs-login) are covered in `#### Sign-in identifier uniqueness rules` below. Sign-in resolves only active `LoginAlias` rows and must validate the resolved `Login.Alias` against the sign-in identifier submitted to `q.registry.IssuePrincipalToken` before accepting the alias path.

#### unchanged: `view.registry.LoginIdx` and `ProjectorLoginIdx`

Continue to index primary logins only, with PK `(AppWSID, AppIDLoginHash)` and value `CDocLoginID ref(Login)`.

#### extended: `payloads.PrincipalPayload` (Go, `pkg/itokens-payloads/types.go`)

Field added:

- `Alias string` -- alias snapshot at token issue time; empty when no alias is set

`Login` carries the canonical primary login of the resolved `Login` record (not necessarily the submitted sign-in identifier). On the primary-login path it is the submitted login; on the alias path it is sourced from `cdoc.registry.LoginAlias.Login`, which snapshots the primary login when the alias index is written. `cdoc.registry.Login` continues to store only `LoginHash`, so the plaintext primary login is recovered from the alias index rather than from `Login`.

#### updated: `c.registry.CreateLogin` (and `c.registry.CreateEmailLogin`)

Signature unchanged. Behavior extended: apply the login-vs-login and login-vs-alias rules from `#### Sign-in identifier uniqueness rules` (via `assertIdentifierAvailable`) before inserting the `Login` CDoc. The change applies transitively to `c.registry.CreateEmailLogin` since both share the `createLogin` helper in `pkg/registry/impl_createlogin.go`.

#### new: `c.registry.InitiateSetLoginAlias`

Authorization: `System` role (System Principal Token required).

````sql
ALTER WORKSPACE sys.AppWorkspaceWS (
    TYPE InitiateSetLoginAliasParams (
        Login text NOT NULL,           -- the primary login whose alias is being set
        AppName text NOT NULL,
        Alias text                     -- requested alias; empty string clears the alias
    );
    EXTENSION ENGINE BUILTIN (
        COMMAND InitiateSetLoginAlias (InitiateSetLoginAliasParams);
    );
    GRANT EXECUTE ON COMMAND InitiateSetLoginAlias TO sys.System;
);
````

Behavior:

- The command is routed to `pseudoWSID(Login)` before execution
- Resolve the source `Login` locally through `view.registry.LoginIdx` using `(AppName, Login)`; reject when no active `Login` is found
- When `Alias` is non-empty, validate it with the same sign-in identifier format rules used by `createLogin`; reject invalid alias values before setting `AliasInProc`
- Reject if `Login.AliasInProc != 0` (concurrent in-progress alias write)
- Set `Login.AliasInProc = 1`; do not write `Login.Alias` in the command
- Treat the requested `Alias` argument as the operation intent consumed by `aproj.registry.ApplySetLoginAlias`: non-empty means create/update, empty string means clear

#### new: `aproj.registry.ApplySetLoginAlias`

Trigger: `AFTER EXECUTE ON c.registry.InitiateSetLoginAlias`.

````sql
ALTER WORKSPACE sys.AppWorkspaceWS (
    EXTENSION ENGINE BUILTIN (
        PROJECTOR ApplySetLoginAlias AFTER EXECUTE ON c.registry.InitiateSetLoginAlias;
    );
);
````

Behavior (all steps idempotent):

1. Read the source `Login` CDoc resolved by `c.registry.InitiateSetLoginAlias`; `SourceAppWSID` is the current `pseudoWSID(InitiateSetLoginAliasParams.Login)`, `CDocLoginID` is the resolved `Login` record id, `oldAlias = Login.Alias`, and `newAlias = InitiateSetLoginAliasParams.Alias`
2. If `newAlias == oldAlias`, write back `AliasInProc = 0`, `AliasError = ""`, and return success
3. If `newAlias != ""`, treat this as create/update and create the new active alias index:
   - determine `targetAppWSID = GetPseudoWSID(NullWSID, newAlias, clusterID)`
   - federation-call `targetAppWSID/c.registry.PutLoginAliasIndex(AppName, SourceAppWSID, newAlias, CDocLoginID, InitiateSetLoginAliasParams.Login)`
   - if the alias-index command returns any error, keep `Alias = oldAlias`, write `AliasError = <reason>` and `AliasInProc = 0` on the source `Login`, and complete the projector; alias-index command errors are finalized business-rule rejections such as collisions, not retryable source commits
4. If `oldAlias != ""`, deactivate the old alias index before committing the source alias:
   - determine `oldTargetAppWSID = GetPseudoWSID(NullWSID, oldAlias, clusterID)`
   - federation-call `oldTargetAppWSID/c.registry.DeactivateLoginAliasIndex(AppName, SourceAppWSID, oldAlias, CDocLoginID)`
   - if old-index deactivation returns any error, keep `Alias = oldAlias`, write `AliasError = <reason>` and `AliasInProc = 0` on the source `Login`, and complete the projector; old-index command errors are finalized business-rule rejections, not retryable source commits
5. Commit the source `Login` alias:
   - for create/update: write `Alias = newAlias`
   - for clear: write `Alias = ""`
   - write `AliasError = ""`
   - write `AliasInProc = 0`
   - if the source write returns any error, best-effort write `AliasError = <reason>`, ignore that write's result, and return projector failure with the original error

Source commit errors return failure so the async projector can be reapplied. Alias-index command rejections are finalized by recording `AliasError` and clearing `AliasInProc`; failure to write that diagnostic state returns failure. If a new alias index was created before a later failure, `q.registry.IssuePrincipalToken` still rejects sign-in by that alias until the source `Login.Alias` commit succeeds, because alias fallback validates `Login.Alias` against the submitted sign-in identifier.

#### new: `c.registry.PutLoginAliasIndex`

Authorization: `System` role.

Caller: internal command invoked by `aproj.registry.ApplySetLoginAlias` during `c.registry.InitiateSetLoginAlias` processing when the requested alias is non-empty.

````sql
ALTER WORKSPACE sys.AppWorkspaceWS (
    TYPE PutLoginAliasIndexParams (
        AppName text NOT NULL,
        SourceAppWSID int64 NOT NULL,
        Alias text NOT NULL,
        CDocLoginID int64 NOT NULL,    -- record id of the source Login in SourceAppWSID
        Login text NOT NULL            -- plaintext primary login; snapshotted into LoginAlias.Login
    );
    EXTENSION ENGINE BUILTIN (
        COMMAND PutLoginAliasIndex (PutLoginAliasIndexParams);
    );
    GRANT EXECUTE ON COMMAND PutLoginAliasIndex TO sys.System;
);
````

Behavior: runs in `targetAppWSID = GetPseudoWSID(NullWSID, Alias, clusterID)`.

1. Apply the alias-vs-login and alias-vs-alias rules from `#### Sign-in identifier uniqueness rules` (via `assertIdentifierAvailable`)
2. Idempotency: look up a `LoginAlias` row by unique combination `(AppName, Alias)`. If an active row is found for the same `(SourceAppWSID, CDocLoginID)`, return success without writing; if an active row belongs to another source login, reject (alias already taken by another user)
3. If an inactive row exists for `(AppName, Alias)`, reactivate it and refresh `(SourceAppWSID, CDocLoginID, Login, Alias)` from the command args
4. Otherwise insert a `LoginAlias` row with `(AppName, SourceAppWSID, CDocLoginID, Login, Alias)`

#### new: `c.registry.DeactivateLoginAliasIndex`

Authorization: `System` role.

Caller: internal command invoked by `aproj.registry.ApplySetLoginAlias` during `c.registry.InitiateSetLoginAlias` processing when the source `Login` had a previous alias.

````sql
ALTER WORKSPACE sys.AppWorkspaceWS (
    TYPE DeactivateLoginAliasIndexParams (
        AppName text NOT NULL,
        SourceAppWSID int64 NOT NULL,
        Alias text NOT NULL,
        CDocLoginID int64 NOT NULL     -- record id of the source Login in SourceAppWSID
    );
    EXTENSION ENGINE BUILTIN (
        COMMAND DeactivateLoginAliasIndex (DeactivateLoginAliasIndexParams);
    );
    GRANT EXECUTE ON COMMAND DeactivateLoginAliasIndex TO sys.System;
);
````

Behavior: runs in `targetAppWSID = GetPseudoWSID(NullWSID, Alias, clusterID)`.

1. Look up the active `LoginAlias` by unique combination `(AppName, Alias)`
2. If no active row exists, return success
3. If the row points to `(SourceAppWSID, CDocLoginID)`, update it with `sys.IsActive = false` and return success
4. If the row points to another login, reject; the command must not deactivate another user's alias index

#### update: `q.registry.IssuePrincipalToken`

Wire types `IssuePrincipalTokenParams` and `IssuePrincipalTokenResult` unchanged.

Behavior:

- The submitted sign-in identifier may be either the primary login or its alias
- Name resolution:
  - Look up the submitted sign-in identifier in `view.registry.LoginIdx` (primary-login path)
  - On primary-login miss, fall back to an active `LoginAlias` by unique combination `(AppName, submitted sign-in identifier)`
  - The `LoginAlias` row carries the alias index, the snapshotted primary login (`LoginAlias.Login`), and the back-pointer `(SourceAppWSID, CDocLoginID)`
  - Because the source `Login` may live in another workspace, the resolver reads it from `SourceAppWSID` with one federation call to the existing `q.sys.GetCDoc(ID = CDocLoginID)`, authorized by a System Principal Token
  - `Login` is a `sys.CDoc`, so `q.sys.GetCDoc` returns its full non-system field set, carrying everything needed to finish sign-in in place (`PwdHash`, `WSID`, `WSError`, `Alias`, `SubjectKind`, `GlobalRoles`); no new registry query is introduced
  - The password check, profile-readiness check, alias validation, and token issue all run within this `IssuePrincipalToken` invocation in `pseudoWSID(alias)`, mirroring the primary-login path
  - The salted `PwdHash` is used transiently and is never persisted outside the source workspace
  - The existing `sys.IsActive` filter applies to the resolved `Login`
  - On the alias path, `PrincipalPayload.Login` is taken from `LoginAlias.Login` (the source `Login` stores only `LoginHash`)
- Alias fallback validation: after dereferencing `LoginAlias`, accept the alias path only when `Login.Alias` equals the submitted sign-in identifier; otherwise treat the alias lookup as outdated and fail sign-in as if the alias did not exist
  - Example: if the submitted sign-in identifier is `alias@example.com`, the active `LoginAlias(Alias = "alias@example.com")` row resolves to a `Login`, and that `Login` currently has `Alias = "alias@example.com"`, sign-in by alias is accepted; if the resolved `Login.Alias` is empty or contains another value, sign-in by that alias is rejected
- Token payload (`PrincipalPayload`): always carries both
  - `Login` -- canonical primary login; the submitted login on the primary path, sourced from `cdoc.registry.LoginAlias.Login` on the alias path
  - `Alias` -- snapshot of `Login.Alias` at issue time; empty when no alias is set
- Snapshot semantics: previously issued tokens are not revoked or rewritten when `Login.Alias` later changes

#### updated: `q.sys.RefreshPrincipalToken`

Wire types `RefreshPrincipalTokenResult` and the public refresh endpoint response shape are unchanged.

Behavior: when a valid principal token is refreshed, the replacement token preserves the existing `PrincipalPayload` identity fields, including `Login` and `Alias`. `Alias` is copied from the input token payload; refresh does not query registry state and does not re-read the current `Login.Alias`. If the alias was updated or cleared after the input token was issued, the refreshed token keeps the input token's alias snapshot until its own normal expiration.

### Out of scope

- Self-service alias management by end users
- Revoking or rewriting previously issued tokens when the alias changes
- Effects on `EnrichPrincipalToken` claim shape
- Denormalizing `PwdHash` into `cdoc.registry.LoginAlias` (possible future optimization; current choice is the single federation read of the source `Login` CDoc via `q.sys.GetCDoc`, which carries the salted `PwdHash` transiently without persisting it)

### References

- [Login CDoc declaration](../../../../../pkg/registry/appws.vsql)
- [create-login command and login-index projector](../../../../../pkg/registry/impl_createlogin.go)
- [async projector precedent (registry side)](../../../../../pkg/registry/impl_invokecreateworkspaceid.go)
- [federation + writeback helper (workspace side)](../../../../../pkg/sys/workspace/impl.go)
- [issue-principal-token sign-in path](../../../../../pkg/registry/impl_issueprincipaltoken.go)
- [q.sys.GetCDoc read-single-CDoc query](../../../../../pkg/sys/collection/cdoc_func.go)
- [PrincipalPayload struct](../../../../../pkg/itokens-payloads/types.go)
- [RefreshPrincipalToken implementation](../../../../../pkg/sys/authnz/impl_refreshprincipaltoken.go)
- [system principal token helpers](../../../../../pkg/itokens-payloads/utils.go)
- [pseudo-WSID routing](../../../../../pkg/coreutils/appwsid.go)
- [System role declaration](../../../../../pkg/sys/sys.vsql)
- [authn technical design](../../../../../uspecs/specs/prod/auth/authn--td.md)
- [clarification decisions](./decisions.md)

## Domain specifications

- [x] update: [prod/domain.md](../../../../specs/prod/domain.md)
  - add: `Sign-in Identifier` concept under Authentication -- the value a subject provides to identify itself at sign-in, either a primary login or a login alias
  - add: `Login Alias` concept under Authentication -- an alternative sign-in identifier that resolves to an existing login
  - update: `User` concept to say a human subject authenticates with a sign-in identifier and credentials
  - update: `Login` concept to define it as the primary registry sign-in identifier for a subject in an application
  - update: `Principal Token` concept to include the alias identity field alongside login, subject kind, and profile workspace

## Functional design

- [x] update: [prod/auth/authn.feature](../../../../specs/prod/auth/authn.feature)
  - add: scenarios for System creating, updating, and clearing a user's login alias
  - add: scenario for rejecting alias management without a System Principal Token
  - add: scenarios for sign-in with original login and with active alias
  - add: scenarios for rejecting alias sign-in after alias update or clear
  - add: scenarios for rejecting alias creation/update when the alias collides with an existing login or alias
  - add: scenario for rejecting login creation when the requested login collides with an existing active alias
  - add: scenario for rejecting an alias that violates the existing sign-in identifier format rules
  - update: token contract scenarios so issued and refreshed principal tokens include login and alias, with refresh preserving the input token's alias snapshot
  - add: scenario for existing principal tokens remaining valid and retaining their alias snapshot after alias update or clear

## Technical design

- [x] update: [prod/auth/authn--td.md](../../../../specs/prod/auth/authn--td.md)
  - update: `Layers` diagram to add the alias-management commands, the `aproj.registry.ApplySetLoginAlias` projector, and `[(registry.LoginAlias)]`
  - add: `[/c.registry.InitiateSetLoginAlias/]` Component under `Registry operations` -- System-authorized command that resolves the source `Login`, validates the requested alias format, sets the in-progress lock, and records the alias intent
  - add: `[/c.registry.PutLoginAliasIndex/]` Component under `Registry operations` -- internal System command that applies alias-vs-login and alias-vs-alias uniqueness and inserts or reactivates the active `[(registry.LoginAlias)]` row in `pseudoWSID(alias)`
  - add: `[/c.registry.DeactivateLoginAliasIndex/]` Component under `Registry operations` -- internal System command that deactivates the previous alias index row owned by the source `Login`
  - add: `[aproj.registry.ApplySetLoginAlias]` Component -- async projector triggered AFTER EXECUTE ON `c.registry.InitiateSetLoginAlias` that drives the cross-workspace put and deactivate calls and commits `Login.Alias`, `Login.AliasError`, and `Login.AliasInProc`
  - add: `[(registry.LoginAlias)]` Component under `State and workspace lifecycle` -- active alias lookup index snapshotting `(AppName, SourceAppWSID, CDocLoginID, Login, Alias)` with `UNIQUE (AppName, Alias)` over active rows
  - update: `[/c.registry.CreateLogin/]` and `[/c.registry.CreateEmailLogin/]` descriptions to note the login-vs-alias collision precheck against active `[(registry.LoginAlias)]` before writing the `Login`
  - update: `[(registry.Login)]` description to include the new `AliasInProc`, `Alias`, and `AliasError` fields
  - update: `[/q.registry.IssuePrincipalToken/]` description to cover alias fallback resolution via active `[(registry.LoginAlias)]`, the cross-workspace `q.sys.GetCDoc` read of the source `Login`, and the `Login.Alias`-vs-submitted-identifier validation
  - add: `Login alias management` Scenario group with flows for set, update, clear, and rejection without a System Principal Token
  - add: sign-in Scenarios for the alias path and for rejecting alias sign-in after the alias is updated or cleared
  - add: uniqueness rejection Scenarios for alias-vs-login, alias-vs-alias, login-vs-alias, and invalid alias format
  - update: `Principal token contract` Scenarios to capture alias snapshot semantics -- refresh preserves the input token alias without re-reading current state, and previously issued tokens stay valid with their alias snapshot after the alias is updated or cleared
  - update: `Cross-cutting concerns` `Testing` to require coverage of the new alias Scenarios

- [x] update: [prod/auth/arch.md](../../../../specs/prod/auth/arch.md)
  - add: `Manage login alias` entry to `Scenarios overview` -- System sets, updates, or clears a user alias used as an alternative sign-in identifier
  - add: `[Registry alias commands]` and `[Alias index projector]` Components under `Registry and tokens`, `[(registry.LoginAlias)]` under `State and workspace lifecycle`, and reflect them in the `Layers` diagram
  - update: `Sign in` Scenario to note resolution by primary login or active alias
  - add: `Manage login alias` Scenario showing the System command, the projector, and the cross-workspace alias index writes

## Construction

### Tests

- [x] update: [sys/it/impl_signupin_test.go](../../../../../pkg/sys/it/impl_signupin_test.go)
  - add: System set, update, and clear of a user alias via `c.registry.InitiateSetLoginAlias`
  - add: alias management rejected without a System Principal Token
  - add: sign-in with the original login and with the active alias both succeed and yield tokens whose `PrincipalPayload` carries `Login` and `Alias`
  - add: sign-in by a previous alias rejected after the alias is updated or cleared
  - add: alias create/update rejected when the alias collides with an existing login or alias, and login creation rejected when the requested login collides with an existing active alias
  - add: alias rejected when it violates the existing sign-in identifier format rules
  - add: a principal token issued before an alias change stays valid and keeps its alias snapshot after the alias is updated or cleared, and refresh preserves that snapshot
  - add: alias edge coverage for idempotent same-alias set, idempotent clear with no alias, unknown source login, wrong pseudo workspace, in-progress alias update conflict, alias wrong-password sign-in rejection, and reuse of a cleared alias by another login

### Schema and constants

- [x] update: [pkg/registry/appws.vsql](../../../../../pkg/registry/appws.vsql)
  - add: `AliasInProc int32`, `Alias varchar`, and `AliasError varchar(1024)` fields to the `Login` table
  - add: `LoginAlias` table inheriting `sys.CDoc` with `AppName`, `SourceAppWSID int64`, `CDocLoginID int64`, `Login`, `Alias`, and `UNIQUE (AppName, Alias)`
  - add: `InitiateSetLoginAliasParams`, `PutLoginAliasIndexParams`, and `DeactivateLoginAliasIndexParams` types
  - add: `COMMAND InitiateSetLoginAlias`, `COMMAND PutLoginAliasIndex`, `COMMAND DeactivateLoginAliasIndex`, and `PROJECTOR ApplySetLoginAlias AFTER EXECUTE ON InitiateSetLoginAlias` in the BUILTIN extension block
  - add: `GRANT EXECUTE ON COMMAND` to `sys.System` for the three new commands

- [x] update: [pkg/registry/consts.go](../../../../../pkg/registry/consts.go)
  - add: QNames for `InitiateSetLoginAlias`, `PutLoginAliasIndex`, `DeactivateLoginAliasIndex`, the `ApplySetLoginAlias` projector, and the `LoginAlias` CDoc
  - add: field-name constants for the new `Login` and `LoginAlias` fields (`Alias`, `AliasInProc`, `AliasError`, `SourceAppWSID`)

### Token payload

- [x] update: [pkg/itokens-payloads/types.go](../../../../../pkg/itokens-payloads/types.go)
  - add: `Alias string` field to `PrincipalPayload`, empty when no alias is set; `q.sys.RefreshPrincipalToken` needs no change because it round-trips the whole payload

### Registry implementation

- [x] create: [pkg/registry/impl_setloginalias.go](../../../../../pkg/registry/impl_setloginalias.go)
  - Purpose: alias lifecycle command, async projector, and index commands for the login-alias feature
  - `execCmdInitiateSetLoginAlias`: resolve the source `Login` via `view.registry.LoginIdx` by `(AppName, Login)`, validate alias format when non-empty, reject when `AliasInProc != 0`, then set `AliasInProc = 1`
  - `applySetLoginAlias` async projector: create the new alias index through `c.registry.PutLoginAliasIndex`, deactivate the previous index through `c.registry.DeactivateLoginAliasIndex`, then commit `Login.Alias`, `AliasError`, and `AliasInProc`, mirroring the federation and writeback pattern in [registry/impl_invokecreateworkspaceid.go](../../../../../pkg/registry/impl_invokecreateworkspaceid.go) and [sys/workspace/impl.go](../../../../../pkg/sys/workspace/impl.go)
  - `execCmdPutLoginAliasIndex`: apply alias-vs-login and alias-vs-alias uniqueness via the shared helper, then idempotently insert or reactivate the active `LoginAlias` row in `pseudoWSID(alias)`
  - `execCmdDeactivateLoginAliasIndex`: deactivate the active `LoginAlias` row owned by `(SourceAppWSID, CDocLoginID)`, idempotent and owner-checked
  - `assertIdentifierAvailable`: shared precheck against `view.registry.LoginIdx` and active `LoginAlias`, used by both login and alias write paths

- [x] update: [pkg/registry/impl_createlogin.go](../../../../../pkg/registry/impl_createlogin.go)
  - add: login-vs-alias precheck through `assertIdentifierAvailable` in `createLogin` before inserting the `Login` CDoc, covering both `CreateLogin` and `CreateEmailLogin`

- [x] update: [registry/impl_issueprincipaltoken.go](../../../../../pkg/registry/impl_issueprincipaltoken.go)
  - add: alias fallback when the submitted identifier misses `view.registry.LoginIdx` -- resolve the active `LoginAlias`, read the source `Login` from `SourceAppWSID` through federated `q.sys.GetCDoc`, and accept only when `Login.Alias` equals the submitted identifier
  - update: set `PrincipalPayload.Login` from the canonical primary login (`LoginAlias.Login` on the alias path) and `PrincipalPayload.Alias` from `Login.Alias`

- [x] update: [pkg/registry/provide.go](../../../../../pkg/registry/provide.go)
  - add: register the three new command functions and the `ApplySetLoginAlias` async projector with federation and itokens, following `provideAsyncProjectorInvokeCreateWorkspaceID`

## Quick start

Set, update, or clear a login alias (requires a System Principal Token):

````text
POST api/v2/apps/sys/registry/workspaces/{pseudoWSID(Login)}/commands/registry.InitiateSetLoginAlias
{"args":{"Login":"user@example.com","AppName":"untill/airs-bp","Alias":"newuser@example.com"}}
````

- Set or update: pass a non-empty `Alias`
- Clear: pass an empty `Alias`

After the alias is set, sign-in via `/auth/login` succeeds with either the original login or the alias, and the issued principal token carries both `Login` and `Alias`.
