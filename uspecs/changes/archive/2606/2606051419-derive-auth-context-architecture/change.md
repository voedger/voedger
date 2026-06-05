---
change_id: 2606050903-derive-auth-context-architecture
type: docs
issue_url: https://untill.atlassian.net/browse/AIR-4185
domains: [prod]
---

# Change request: Derive `auth` context architecture

Refs:

- [AIR-4185: voedger: derive architecture from auth context](./issue-AIR-4185.md)

## Why

The `auth` context listed in `uspecs/specs/prod/domain.md` covers authentication, authorization, token management, and workspace membership, but the existing `uspecs/specs/prod/auth/arch.md` covers only the authentication (authn) flows; authorization, principal token validation on request processing, and invite-based workspace membership have no Context Architecture or Context Subsystem Architecture reference. There is also no Context Architecture overview that ties the four subsystems together, so a reader cannot see how authn, authz, token management, and membership compose into a single context. Principal token issue and refresh have no arch coverage either — only validation is implicitly described inside the existing authn material. Finally, the existing `authn--td.md`, `authn.feature`, and `invites--td.md` were written before the four-subsystem split was made explicit in `domain.md` and may diverge from the new arch chapters as derivation surfaces contradictions, so they must be reconciled in the same change.

## What

Add architecture specifications for the `auth` context, derived from scratch from the current codebase across the entire repository:

- Add a Context Architecture summary at `uspecs/specs/prod/auth/arch.md` that overviews the full `auth` context (authn, authz, token management, workspace membership) and links to its subsystem architectures, replacing the current narrow authn-only content at the same path
- Add a Context Subsystem Architecture for authentication at `uspecs/specs/prod/auth/arch-authn.md` covering login creation (including login-alias set/update/clear), sign-in (by login or by active alias), password lifecycle, the verifier sub-flow that issues and consumes verified-value tokens (email verification, password reset), and the profile workspace readiness gate seen by sign-in (excluding principal token issue/refresh, which belongs to the token management chapter, and excluding profile workspace lifecycle itself, which stays owned by the `apps` context)
- The authentication chapter must explicitly document two login-lifecycle behaviors observable to clients:
  - when a profile workspace is deactivated, the deactivation propagates to `[(registry.Login)].IsActive=false` via `c.sys.OnChildWorkspaceDeactivated`; the deactivated login is treated as a missing login by subsequent `[q.registry.IssuePrincipalToken]` calls, which return the shared `errLoginOrPasswordIsIncorrect` response (HTTP 401, "login or password is incorrect") — the same response used for missing logins and wrong passwords to prevent enumeration
  - re-creating a login with the same name creates a new login with a fresh profile workspace; the previously deactivated login and its profile workspace remain in storage but become unreachable
- Add a Context Subsystem Architecture for authorization at `uspecs/specs/prod/auth/arch-authz.md` covering runtime ACL evaluation and the enforcement points exposed to other contexts (notably the `apps` context processors); VSQL schema parsing of role and grant declarations stays owned by the `apps` context, and the management of the subjects doc and joined-workspace records stays owned by the workspace membership chapter
- The authorization chapter must enumerate the four sources of a principal's effective roles by origin:
  - invite-granted roles — `[(cdoc.sys.Subject)]` rows in the request workspace, produced by the workspace membership subsystem
  - token-carried roles — `PrincipalPayload.Roles` (per-app) and `PrincipalPayload.GlobalRoles` (cross-workspace), snapshotted at sign-in by the authentication subsystem
  - request-context roles — derived at composition time from request state: `role.sys.ProfileOwner` when the request workspace is the principal's own profile, `role.sys.WorkspaceOwner` when the principal owns the request workspace, `role.sys.WorkspaceDevice` for allowed devices, `role.sys.System` for system tokens, plus `role.sys.AuthenticatedUser` and the `Host` principal
  - anonymous-grants — emitted when no token is presented: `role.sys.Anonymous`, the `sys.Guest` user, and any `[(cdoc.sys.Subject)]` rows matching `sys.Guest`
  - VSQL-declared role inheritance (e.g., `role.sys.ProfileOwner` implies `role.sys.WorkspaceOwner`) is expanded by the ACL engine at evaluation time, not during principal composition
- Add a Context Subsystem Architecture for token management at `uspecs/specs/prod/auth/arch-tokens.md` covering principal token issue, refresh, validation, and the principal payload contract shared by the authentication and authorization subsystems; the principal payload contract carries the alias snapshot captured at token-issue time and is immune to subsequent alias changes (per the existing alias-snapshot scenarios in `authn.feature`); verified-value tokens are out of scope here and covered by the authentication chapter, since their only consumer is the verifier sub-flow
- Add a Context Subsystem Architecture for workspace membership at `uspecs/specs/prod/auth/arch-membership.md` covering the invite lifecycle, the subjects doc, joined-workspace records, role updates, and member removal
- Introduce the shared auth-context concepts once in `arch.md` — Principal Token, the subjects doc, the login record, and the `pkg/iauthnz` apps→auth enforcement boundary — and cross-link them from the subsystem chapters that produce or consume them; per-subsystem chapters reference the shared definition rather than restating it
- Treat the existing `auth/arch.md`, `auth/authn--td.md`, `auth/authn.feature`, and `auth/invites--td.md` as reference material consulted during derivation to capture key architectural points the codebase alone might not surface; do not treat them as authoritative sources, since they may be outdated; the new `arch.md` replaces the current one at the same path
- Update `auth/authn--td.md`, `auth/authn.feature`, and `auth/invites--td.md` in lockstep with the new arch chapters so that every divergence found during derivation is reconciled in this change (no stale or contradictory material left behind); each TD continues to live at its current path and keeps its scope (feature/technical design, not architecture)
- Give reviewers and contributors a single architecture reference for the `auth` context that complements `uspecs/specs/prod/domain.md` and mirrors the structure of `uspecs/specs/prod/apps/arch.md`

## How

Decisions:

- Adopt uspecs-td conventions: `arch.md` for the Context Architecture, `arch-{subsystem}.md` for Context Subsystem Architectures, mirroring `uspecs/specs/prod/apps/`
- Split the `auth` context into four subsystems matching `domain.md`: authentication (`arch-authn.md`), authorization (`arch-authz.md`), token management (`arch-tokens.md`), workspace membership (`arch-membership.md`)
- Derive each chapter from scratch from the current codebase; treat the existing `auth/arch.md`, `auth/authn--td.md`, `auth/authn.feature`, `auth/invites--td.md` as reference material only, not authoritative
- Introduce shared cross-cutting concepts (Principal Token, the subjects doc, the login record, the `pkg/iauthnz` apps→auth enforcement boundary) once in `arch.md` and cross-link them from the subsystem chapters that produce or consume them
- Place the verifier sub-flow (issues and consumes verified-value tokens for email verification and password reset) under the authentication chapter, since its only consumers are authn flows in `pkg/registry`
- Derive the authorization chapter's four-source effective-roles model from `pkg/iauthnzimpl/impl.go` (invite-granted, subjects doc, principal token incl. `GlobalRoles`, ACL-engine-emitted contextual roles plus VSQL inheritance)
- Reconcile existing TDs in the same change rather than deferring to a follow-up issue
- Derive the two login-lifecycle behaviors (listed in `## What`) from `pkg/registry/utils.go` `IsActive` / `errLoginDoesNotExist`

Out of scope:

- VSQL schema parsing of role and grant declarations (owned by the `apps` context)
- Profile workspace lifecycle — creation, profile-fields write-back (owned by the `apps` context); only the readiness gate seen by sign-in is in scope
- Authnz/ACL enforcement-point internals within command/query/actualizer processors (owned by the `apps` context processing chapter; auth chapters only describe the `pkg/iauthnz` boundary they expose)
- Any behavior change to auth code paths
- Updates to `uspecs/specs/prod/domain.md`

References:

- [auth context in the domain specification](../../../../../uspecs/specs/prod/domain.md)
- [existing auth arch reference material](../../../../../uspecs/specs/prod/auth/arch.md)
- [existing authn technical design](../../../../../uspecs/specs/prod/auth/authn--td.md)
- [existing authn feature](../../../../../uspecs/specs/prod/auth/authn.feature)
- [existing invites technical design](../../../../../uspecs/specs/prod/auth/invites--td.md)
- [apps context architecture as structural model](../../../../../uspecs/specs/prod/apps/arch.md)
- [authnz interface used at enforcement points](../../../../../pkg/iauthnz)
- [authnz implementation with contextual role emission and role-source composition](../../../../../pkg/iauthnzimpl/impl.go)
- [registry package: login lifecycle, principal token issue/refresh, reset-password](../../../../../pkg/registry)
- [authnz shared constants and types](../../../../../pkg/sys/authnz)
- [invite package: invite lifecycle, subjects doc, joined-workspace records](../../../../../pkg/sys/invite)
- [verifier package: verified-value token issuance and consumption](../../../../../pkg/sys/verifier)
- [token payload contracts](../../../../../pkg/itokens-payloads/types.go)
- [tokens interface](../../../../../pkg/itokens)

## Functional design

- [x] update: [auth/authn.feature](../../../../specs/prod/auth/authn.feature)
  - update: reconcile scenarios with the new `arch-authn.md` and `arch-tokens.md` so no scenario contradicts the derived architecture (verifier sub-flow, principal token issue/refresh, profile workspace readiness gate)
  - add: scenarios covering the two login-lifecycle behaviors required by `## What` (deactivated-login and recreate-same-name)

## Technical design

- [x] update: [auth/arch.md](../../../../specs/prod/auth/arch.md)
  - rewrite: replace the current authn-only content with a Context Architecture overview of the full `auth` context (authn, authz, token management, workspace membership) and links to its subsystem architectures per `## What`
  - add: shared cross-cutting auth-context concepts defined once and cross-linked from the subsystem chapters that produce or consume them per `## What`

- [x] create: [auth/arch-authn.md](../../../../specs/prod/auth/arch-authn.md)
  - Context Subsystem Architecture for authentication; scope per `## What` (including login-alias set/update/clear, sign-in by login or by active alias, the verifier sub-flow, the profile workspace readiness gate, and the two login-lifecycle behaviors)

- [x] create: [auth/arch-authz.md](../../../../specs/prod/auth/arch-authz.md)
  - Context Subsystem Architecture for authorization; scope per `## What` (runtime ACL evaluation, enforcement points exposed to other contexts, and the four sources of effective roles)

- [x] create: [auth/arch-tokens.md](../../../../specs/prod/auth/arch-tokens.md)
  - Context Subsystem Architecture for token management; scope per `## What` (principal token issue/refresh/validation and the principal payload contract, including the alias snapshot captured at token-issue time and its immunity to subsequent alias changes)

- [x] create: [auth/arch-membership.md](../../../../specs/prod/auth/arch-membership.md)
  - Context Subsystem Architecture for workspace membership; scope per `## What` (invite lifecycle, subjects doc, joined-workspace records, role updates, member removal)

- [x] update: [auth/authn--td.md](../../../../specs/prod/auth/authn--td.md)
  - update: reconcile with the new `arch-authn.md` and `arch-tokens.md` so no statement contradicts the derived architecture, including the alias snapshot semantics in the principal payload contract
  - remove: any material that duplicates content now owned by `arch-authn.md`, `arch-tokens.md`, or the shared concepts in `arch.md`; replace with references

- [x] update: [auth/invites--td.md](../../../../specs/prod/auth/invites--td.md)
  - update: reconcile with the new `arch-membership.md` and `arch-authz.md` so no statement contradicts the derived architecture (subjects doc and joined-workspace records as the membership data sources consumed by authz)
  - remove: any material that duplicates content now owned by `arch-membership.md`, `arch-authz.md`, or the shared concepts in `arch.md`; replace with references
