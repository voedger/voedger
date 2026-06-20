# voedger: flip token claim contract to Login (canonical) + Alias (active alias)

- URL: https://untill.atlassian.net/browse/AIR-4327
- ID: AIR-4327
- State: To Do
- Author: Maksim Geraskin
- Assignees: Maksim Geraskin
- Labels: none

> **Local copy note:** This change-folder copy uses the `Alias` claim name agreed during spec authoring; the new claim was originally proposed in the ticket as `PresentedLogin`. Rationale is recorded as a comment on AIR-4327, and the live ticket text may still read `PresentedLogin`. References to the *existing* `PresentedLogin` field under "Current contract" are intentionally left unchanged, since that is its name in the currently deployed code.

## Description

Revisit the principal-token claim contract so the Login claim carries the canonical (immutable internal) identity and a new Alias claim carries the active alias snapshot at issue time. This inverts the contract shipped in change 2606151848-token-login-reflects-alias and supersedes that decision.

### Current contract

In pkg/itokens-payloads/types.go, PrincipalPayload.PresentedLogin carries a json:"Login" tag (wire claim Login = alias snapshot, or canonical when no alias), and CanonicalLogin is a separate claim. Net: every existing reader of the Login claim silently switched from canonical to alias.

### Proposed contract

* `Login` (wire claim) = canonical primary login (its historical meaning).
* `Alias` (new wire claim) = active alias snapshot at issue time; empty when no alias is set.

### Rationale

Safe-by-default: the dangerous default (Login = identity used for credentials, routing, authorization, quotas, metrics) stays correct with no migration, and "show the alias" becomes an explicit opt-in via Alias. The current contract inverts this and forces every identity consumer to migrate off Login (this is what broke the frontend change-password flow, since the registry GetCDocLogin cannot resolve an alias).

### Scope

* `pkg/itokens-payloads/types.go`: rename CanonicalLogin -> Login (no tag) and the PresentedLogin field -> Alias (drop the json:"Login" tag, serializing as the new Alias claim); update doc comments and consts.go initializer.
* `pkg/registry/impl_issueprincipaltoken.go`: set Login = loginForSignIn.canonicalLogin and Alias = alias snapshot (empty when no alias).
* `pkg/iauthnzimpl`: resolve subject roles against both Login and Alias (same dedup as today, renamed fields).
* `uspecs/specs/prod/auth/arch-tokens.md` and related specs/tests: update claim descriptions.
* Token transition window: tokens minted under the current contract carry Login = alias; after the switch they would be misread as canonical. Define the cutover (bounded by token TTL and self-healing on re-sign-in / refresh) and whether the current contract is already deployed.

Supersedes change 2606151848-token-login-reflects-alias. Frontend impact (AIR-4292): change-password then needs no frontend change; the only optional follow-up is an Alias reader for display.

---

Co-authored by [Augment Code](https://www.augmentcode.com/?utm_source=atlassian&utm_medium=jira_issue&utm_campaign=jira)
