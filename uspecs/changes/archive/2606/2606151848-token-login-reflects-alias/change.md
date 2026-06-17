---
change_id: 2606151510-token-login-reflects-alias
type: feat
issue_url: https://untill.atlassian.net/browse/AIR-4270
domains: [prod]
scope: [auth]
breaking: true
---

# Change request: Principal token login reflects active alias at sign-in

Refs:

- [AIR-4270: voedger: principal token Login reflects alias used at sign-in](./issue-AIR-4270.md)

## Why

The frontend shows the login value carried by the principal token, but on alias sign-in that value is the canonical primary login rather than the alias the user just typed, which confuses users mid email-change. Presenting the alias the user signed in with, while keeping the canonical primary login available for backend identity, removes that confusion.

## What

Adjust the authentication token identity contract so a signed-in user is presented with the identifier they recognize:

- After sign-in, the principal token presents the identifier the user signed in by: their active alias when one is set, otherwise their primary login
- The canonical primary login remains available in the token alongside the presented identifier, so backend identity needs are unaffected
- The presented identifier and the canonical login are captured at token issue time and preserved unchanged when the token is refreshed
- Workspace authorization resolves a user's roles using both the presented identifier and the canonical login, so role assignment matches regardless of which identifier a subject was registered under
- When the user has no alias, the token and downstream behavior are unchanged from today

## How

Decisions:

- Rename the `PrincipalPayload.Login` Go field to `PresentedLogin` and keep the wire/JWT claim key as `Login` via a `json:"Login"` struct tag; this preserves the frontend claim contract while turning every existing backend `payload.Login` read into a compile error that must be revisited (the intended guard against reusing the presented value as internal identity)
- Replace the `Alias` field with `CanonicalLogin` in `PrincipalPayload`; its Go field name equals the claim key, so no tag is needed and the JWT claim key changes from `Alias` to `CanonicalLogin`
- In `q.registry.IssuePrincipalToken`, set `PresentedLogin` to the active alias snapshot and fall back to the canonical login when the alias is empty; set `CanonicalLogin` to the resolved canonical primary login, reusing the existing `signInLogin.alias` and `signInLogin.canonicalLogin` values without new resolution logic
- In workspace authorization (`iauthnzimpl`), resolve subject roles against both `PresentedLogin` and `CanonicalLogin` (deduplicated, skipping the `CanonicalLogin` read when it is empty or equal to `PresentedLogin`) instead of `Login` alone
- Keep the user principal's display Name and the n10n subject on the canonical primary login for stable internal identity; the alias-aware value is surfaced only through the token `Login` claim (the `PresentedLogin` field) consumed by the frontend
- Both internal-identity consumers (display Name, n10n subject) fall back to `PresentedLogin` when `CanonicalLogin` is empty; this is a back-compat bridge for tokens issued before the deploy (which carry no `CanonicalLogin` claim and whose `Login`/`PresentedLogin` already holds the canonical login) and for the system principal, and it mirrors the empty-value guard used in role resolution; for newly issued tokens `CanonicalLogin` is always set, so the alias is never used as internal identity
- Leave token refresh and enrich untouched: both round-trip the whole payload (enrich rewrites only `Roles`), so `PresentedLogin` and `CanonicalLogin` are preserved as an issue-time snapshot with no code change

Out of scope:

- Alias lifecycle management (set, update, clear) and the alias index, which are unchanged
- Frontend work to consume the adjusted token field
- API-token identity (`IsAPIToken`), which does not use `Login` for display

References:

- [principal token payload contract](../../../../../pkg/itokens-payloads/types.go)
- [sign-in token issue](../../../../../pkg/registry/impl_issueprincipaltoken.go)
- [sign-in login resolution helpers](../../../../../pkg/registry/impl_setloginalias.go)
- [workspace authorization and subject role resolution](../../../../../pkg/iauthnzimpl/impl.go)
- [token refresh and enrich preserve the payload](../../../../../pkg/sys/authnz/impl_enrichprincipaltoken.go)
- [sign-in and token claim integration tests](../../../../../pkg/sys/it/impl_signupin_test.go)
- [principal token contract specification](../../../../../uspecs/specs/prod/auth/arch-tokens.md)
- [authentication sign-in specification](../../../../../uspecs/specs/prod/auth/arch-authn.md)

## Functional design

- [x] update: [auth/authn.feature](../../../../specs/prod/auth/authn.feature)
  - update: "the issued principal token identifies login, alias, subject kind, and profileWSID" (sign-in by login outline) -> identifies login, canonical login, subject kind, and profileWSID
  - update: "Principal token carries original login and alias after alias sign-in" scenario -> the token presents the active alias as the login and carries the original login as the canonical login
  - update: "the new principalToken preserves login, alias, subject kind, and profileWSID from the input token" (refresh) -> preserves login, canonical login, subject kind, and profileWSID
  - update: "Existing principal token keeps alias snapshot after alias changes" scenario -> the token retains the login and canonical login captured at issue time
  - add: scenario "Principal token uses the active alias as login when signing in with the original login" asserting the issued token's login is the active alias and its canonical login is the original login

## Technical design

- [x] update: [auth/arch-tokens.md](../../../../specs/prod/auth/arch-tokens.md)
  - rename: `[PrincipalPayload]` field `Alias` -> `CanonicalLogin`; redefine `Login` as the identifier the subject is known by (active alias snapshot at issue time, or the canonical login when no alias is set) and define `CanonicalLogin` as the canonical login resolved at issue time
  - update: "Issue principal token" overview and the issue diagram so `Login` is set to the alias-or-canonical value and `CanonicalLogin` carries the canonical login snapshot
  - update: "Refresh principal token" overview, the `[q.sys.RefreshPrincipalToken]` entry, the refresh diagram, and the refresh note to preserve `Login` and `CanonicalLogin` verbatim instead of the `Alias` snapshot

- [x] update: [auth/arch.md](../../../../specs/prod/auth/arch.md)
  - update: `[Principal Token]` shared-concept payload field list `Alias` -> `CanonicalLogin`
  - update: "Manage tokens" overview wording from the alias snapshot to the captured `CanonicalLogin`

- [x] update: [auth/arch-authn.md](../../../../specs/prod/auth/arch-authn.md)
  - update: the `[q.registry.IssuePrincipalToken]` build step to set `Login` = active-alias snapshot with canonical fallback and `CanonicalLogin` = canonical login
  - update: the "Sign in by login or by active alias" diagram `build PrincipalPayload(...)` to list `Login` (alias-or-canonical) and `CanonicalLogin`

- [x] update: [auth/arch-authz.md](../../../../specs/prod/auth/arch-authz.md)
  - update: the `[Subjects reader]` component and the `[Invite-granted]` role source to resolve subject roles by matching `[(cdoc.sys.Subject)]` rows against both `PrincipalPayload.Login` and `PrincipalPayload.CanonicalLogin`
  - update: the Authenticate flow step that reads subjects to evaluate both fields in `RequestWSID`

- [x] update: [auth/authn--td.md](../../../../specs/prod/auth/authn--td.md)
  - update: the `[/q.registry.IssuePrincipalToken/]` entry and the alias sign-in scenario to snapshot `Login` (alias-or-canonical) and `CanonicalLogin`, deferring payload-field semantics to arch-tokens.md
  - update: the "Principal token contract" scenarios (identity fields, refresh, alias-snapshot immutability) to reference `Login` and `CanonicalLogin`
  - update: the Security cross-cutting bullet to list canonical login instead of alias

## Construction

### Tests

- [x] update: [sys/it/impl_signupin_test.go](../../../../../pkg/sys/it/impl_signupin_test.go)
  - update: `assertPrincipalTokenClaims` - rename its two value parameters from `(expectedLogin, expectedAlias)` to `(expectedPresentedLogin, expectedCanonicalLogin)` (their meaning flips from canonical-then-alias to presented-then-canonical); assert the Go field `payload.PresentedLogin` and the raw `Login` claim both equal the presented identifier, assert the raw `CanonicalLogin` claim equals the canonical login, and assert the legacy `Alias` claim key is absent
  - update: all `assertPrincipalTokenClaims` call sites in `TestLoginAlias` (alias sign-in and refresh-snapshot) so the presented value is the active alias and the canonical value is the canonical primary login
  - update: the existing `primaryToken` assertion (already signs in by the original login while an alias is active) so its presented value is the active alias and its canonical value is the original login; this site already covers the new `authn.feature` scenario, so no new case is added

- [x] update: [iauthnzimpl/impl_test.go](../../../../../pkg/iauthnzimpl/impl_test.go)
  - update: the `PrincipalPayload{Login: ...}` initializers field `Login` -> `PresentedLogin` so they compile under the rename

- [x] update: [itokens-payloads/impl_test.go](../../../../../pkg/itokens-payloads/impl_test.go)
  - update: the `PrincipalPayload{Login: ...}` initializers field `Login` -> `PresentedLogin` so they compile under the rename

- [x] update: [processors/command/impl_test.go](../../../../../pkg/processors/command/impl_test.go)
  - update: the `PrincipalPayload{Login: ...}` initializer field `Login` -> `PresentedLogin` so it compiles under the rename

- [x] update: [processors/query/impl_test.go](../../../../../pkg/processors/query/impl_test.go)
  - update: the `PrincipalPayload{Login: ...}` initializers field `Login` -> `PresentedLogin` so they compile under the rename

- [x] update: [sys/it/impl_childworkspace_test.go](../../../../../pkg/sys/it/impl_childworkspace_test.go)
  - update: the `PrincipalPayload{Login: ...}` initializer field `Login` -> `PresentedLogin` so it compiles under the rename

- [x] update: [sys/it/impl_sqlquery_test.go](../../../../../pkg/sys/it/impl_sqlquery_test.go)
  - update: the `PrincipalPayload{Login: ...}` initializer field `Login` -> `PresentedLogin` so it compiles under the rename

- [x] update: [sys/it/impl_qpv2_test.go](../../../../../pkg/sys/it/impl_qpv2_test.go)
  - update: the `PrincipalPayload{Login: ...}` initializers field `Login` -> `PresentedLogin` so they compile under the rename

- [x] update: [sys/it/impl_workspace_test.go](../../../../../pkg/sys/it/impl_workspace_test.go)
  - update: the `PrincipalPayload{Login: ...}` initializer field `Login` -> `PresentedLogin` so it compiles under the rename

### Implementation

- [x] update: [itokens-payloads/types.go](../../../../../pkg/itokens-payloads/types.go)
  - rename: the `PrincipalPayload.Login` field to `PresentedLogin` with a `json:"Login"` struct tag (preserves the `Login` claim) and rename the `Alias` field to `CanonicalLogin` (no tag; the Go name equals the new claim key, so the JWT claim key changes from `Alias` to `CanonicalLogin`)
  - add: doc comments stating `PresentedLogin` is the frontend-facing presented identity (active alias snapshot, or canonical login when no alias; not to be used for identity, authorization, quotas, or metrics) and `CanonicalLogin` is the immutable internal identity, plus a one-line note explaining why `PresentedLogin` carries a `json:"Login"` tag
- [x] update: [itokens-payloads/consts.go](../../../../../pkg/itokens-payloads/consts.go)
  - update: the `systemPrincipalPayload` initializer field `Login` -> `PresentedLogin` (compile-breaker from the rename); `CanonicalLogin` is left unset for the system principal
- [x] update: [registry/impl_issueprincipaltoken.go](../../../../../pkg/registry/impl_issueprincipaltoken.go)
  - update: the `PrincipalPayload` build so `PresentedLogin` is `loginForSignIn.alias` when non-empty and falls back to `loginForSignIn.canonicalLogin`, and `CanonicalLogin` is `loginForSignIn.canonicalLogin`
- [x] update: [iauthnzimpl/impl.go](../../../../../pkg/iauthnzimpl/impl.go)
  - update: `Authenticate` to read subject roles for both `principalPayload.PresentedLogin` and `principalPayload.CanonicalLogin`, deduplicating the appended role principals and skipping the `CanonicalLogin` read when it is empty or equal to `PresentedLogin`
  - update: the user principal display `Name` (`loginName`) to come from `principalPayload.CanonicalLogin`, falling back to `PresentedLogin` when `CanonicalLogin` is empty, so internal identity stays canonical for new tokens while legacy tokens and the system principal still resolve a name
- [x] update: [processors/n10n/impl_subscribeandwatch.go](../../../../../pkg/processors/n10n/impl_subscribeandwatch.go)
  - update: `subjectLogin` to come from `principalPayload.CanonicalLogin`, falling back to `PresentedLogin` when `CanonicalLogin` is empty, so the notification subject identity stays canonical for new tokens and remains stable for legacy tokens
- [x] update: [vit/impl.go](../../../../../pkg/vit/impl.go)
  - update: the `RefreshTokens` `PrincipalPayload` initializer to set `PresentedLogin` = `prn.Name` and `CanonicalLogin` = `prn.Name`, so VIT-issued tokens keep canonical internal identity for the display `Name` and n10n subject
