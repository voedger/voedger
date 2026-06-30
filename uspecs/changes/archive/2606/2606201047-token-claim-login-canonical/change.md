---
change_id: 2606191522-token-claim-login-canonical
type: feat
issue_url: https://untill.atlassian.net/browse/AIR-4327
domains: [prod]
scope: [auth]
breaking: true
---

# Change request: Token Login claim carries canonical identity, new Alias claim carries the active alias

Refs:

- [AIR-4327: voedger: flip token claim contract to Login (canonical) + PresentedLogin (alias)](./issue-AIR-4327.md)

## Why

The previous change made the `Login` token claim carry the alias snapshot, silently switching every existing reader from the canonical identity to the alias and breaking the frontend change-password flow. Restoring `Login` to the canonical identity is safe-by-default: identity used for credentials, routing, authorization, quotas, and metrics stays correct with no migration, while showing the alias becomes an explicit opt-in.

## What

In the prod auth domain, principal-token claims are restructured so the canonical identity is the default:

- The `Login` claim again carries the canonical primary login (its historical meaning), so all identity consumers read the canonical login without changes.
- A new `Alias` claim carries the active alias snapshot at issue time, and is empty when no alias is set.
- Authorization resolves subject roles against both the canonical login and the active alias, so role assignments keyed on either remain effective.
- This supersedes the earlier alias-in-Login contract; tokens minted under the old contract self-heal within token TTL on re-sign-in or refresh.

## How

Decisions:

- In `pkg/itokens-payloads/types.go`, rename the `CanonicalLogin` field to `Login` with no json tag so the wire claim `Login` again carries the canonical identity; rename the `PresentedLogin` field to `Alias` and drop its `json:"Login"` tag so it serializes under a new `Alias` claim, left empty when no alias is set.
- In `pkg/itokens-payloads/consts.go`, set the system principal's canonical `Login` to `"system"` and leave `Alias` empty.
- In `pkg/registry/impl_issueprincipaltoken.go`, set `Login` directly from `loginForSignIn.canonicalLogin` and `Alias` from `loginForSignIn.alias` (no fallback to canonical), so `Alias` is empty when no alias exists.
- In `pkg/iauthnzimpl/impl.go`, key the internal identity (`loginName`) on `Login` with a fallback to `Alias` for legacy/system tokens, and keep resolving subject roles against both `Login` and `Alias` with the existing dedup.
- In `pkg/processors/n10n/impl_subscribeandwatch.go` and the `pkg/vit` test helper, read/set the internal identity from `Login` (fallback `Alias`).
- Remove the dormant `GRANT SELECT ON TABLE Login TO sys.ProfileOwner` in `pkg/registry/appws.vsql`: `ProfileOwner` is only ever emitted in a subject's own profile workspace, never in the registry app workspace where the `Login` table lives, so the grant confers no client read access today. Removing it makes the schema match the System-only read contract specified by the `Login alias state visibility` feature rule. Non-breaking: no client ever had effective read access through it.
- Token transition relies on token TTL plus self-healing on re-sign-in/refresh; no data migration and no explicit deployment cutover orchestration.
- Update the affected `*_test.go` payload constructions and the auth specs/diagrams to reflect `Login` = canonical and `Alias` = active alias.

Out of scope:

- Frontend `Alias` display reader for change-password (optional AIR-4292 follow-up).
- Any other authentication or authorization behavior beyond the claim contract.

References:

- [principal token payload type](../../../../../pkg/itokens-payloads/types.go)
- [system principal payload defaults](../../../../../pkg/itokens-payloads/consts.go)
- [principal token issue](../../../../../pkg/registry/impl_issueprincipaltoken.go)
- [registry app-workspace schema and ACL grants](../../../../../pkg/registry/appws.vsql)
- [authorization subject/role resolution](../../../../../pkg/iauthnzimpl/impl.go)
- [n10n subject login resolution](../../../../../pkg/processors/n10n/impl_subscribeandwatch.go)
- [principal token claim assertions](../../../../../pkg/sys/it/impl_signupin_test.go)
- [token architecture spec](../../../../../uspecs/specs/prod/auth/arch-tokens.md)
- [Conventional Commits v1.0.0](https://www.conventionalcommits.org/en/v1.0.0/)

## Functional design

- [x] update: [auth/authn.feature](../../../../specs/prod/auth/authn.feature)
  - update: "Principal token carries authn identity fields" scenario -> the issued principal token identifies its login (canonical), subject kind, and profileWSID (core always-present fields; alias semantics covered by the dedicated scenario below)
  - update: "Client refreshes a principal token" scenario -> the new token preserves login (canonical), alias, subject kind, and profileWSID from the input token
  - replace: merge the two scenarios "Principal token uses the active alias as login after alias sign-in" and "Principal token uses the active alias as login when signing in with the original login" into one scenario outline owning alias semantics -> the token's login is always the canonical login; its alias is the active alias when an alias is active (regardless of which identifier was used to sign in) and empty when no alias is set
  - update: "Existing principal token retains login and canonical login after alias changes" scenario -> the existing token retains the login (canonical) and alias captured at issue time
  - add: "Login alias state visibility" rule -> System can read a login's alias state (active alias, in-progress flag, error); a non-System caller's read is rejected (documents the pre-existing System-only read access of the registry Login record; not introduced by the token-claim change)

## Technical design

- [x] update: [auth/arch-tokens.md](../../../../specs/prod/auth/arch-tokens.md)
  - update: `[PrincipalPayload]` contract -> `Login` = canonical login (immutable internal identity); rename the `CanonicalLogin` field to `Alias` = active alias snapshot at issue time, empty when no alias; both captured at issue time and never re-read on refresh
  - update: the Issue and Refresh scenario prose and pseudocode to build and preserve `Login` (canonical) + `Alias`, replacing `Login` (alias snapshot) + `CanonicalLogin`

- [x] update: [auth/arch-authn.md](../../../../specs/prod/auth/arch-authn.md)
  - update: the `[q.registry.IssuePrincipalToken]` description and the sign-in pseudocode -> set `Login` = canonical login and `Alias` = active-alias snapshot (empty when no alias), dropping the alias-as-`Login` / `CanonicalLogin` wording

- [x] update: [auth/arch-authz.md](../../../../specs/prod/auth/arch-authz.md)
  - update: the `[Subjects reader]` composition -> match `[(cdoc.sys.Subject)]` rows against `PrincipalPayload.Login` and `PrincipalPayload.Alias` (renamed from `CanonicalLogin`), keeping the same dedup

- [x] update: [auth/arch.md](../../../../specs/prod/auth/arch.md)
  - update: the `PrincipalPayload(...)` field list and the refresh note -> replace `CanonicalLogin` with `Alias`; `Login` now denotes the canonical login

- [x] update: [auth/authn--td.md](../../../../specs/prod/auth/authn--td.md)
  - update: the token issue/refresh flows and worked examples -> `Login` carries the canonical login and `Alias` carries the active alias, replacing `Login` (active alias) + `CanonicalLogin`

## Construction

### Tests

- [x] update: [sys/it/impl_signupin_test.go](../../../../../pkg/sys/it/impl_signupin_test.go)
  - update: `assertPrincipalTokenClaims` -> rename params to `(expectedLogin, expectedAlias)`; assert `payload.Login` and the `Login` claim equal the canonical login, `payload.Alias` and the `Alias` claim equal the active alias (empty when none), and that no `CanonicalLogin` claim is present
  - update: the alias sign-in assertions so the token's `Login` is the canonical login and `Alias` is the active alias, for both alias and original-login sign-in
  - add: test for the `Login alias state visibility` rule -> System reads a login's alias state (asserts `Alias`/`AliasInProc`/`AliasError`). The non-System-rejected half is already covered by `TestAuthnz/"foreign app"` (regular user cross-app read of `registry.Login` -> 403, independent of the dormant grant), so no duplicate was added

- [x] update: [iauthnzimpl/impl_test.go](../../../../../pkg/iauthnzimpl/impl_test.go)
  - update: `PrincipalPayload` fixtures `PresentedLogin:` -> `Login:`
  - add/update: a case asserting subject roles resolve against both `Login` and `Alias`

- [x] update: rename `PresentedLogin:` -> `Login:` (canonical) and drop `CanonicalLogin:` in the remaining payload fixtures
  - [itokens-payloads/impl_test.go](../../../../../pkg/itokens-payloads/impl_test.go)
  - [processors/command/impl_test.go](../../../../../pkg/processors/command/impl_test.go)
  - [processors/query/impl_test.go](../../../../../pkg/processors/query/impl_test.go)
  - [sys/it/impl_childworkspace_test.go](../../../../../pkg/sys/it/impl_childworkspace_test.go)
  - [sys/it/impl_qpv2_test.go](../../../../../pkg/sys/it/impl_qpv2_test.go)
  - [sys/it/impl_sqlquery_test.go](../../../../../pkg/sys/it/impl_sqlquery_test.go)
  - [sys/it/impl_workspace_test.go](../../../../../pkg/sys/it/impl_workspace_test.go)

### Payload contract

- [x] update: [itokens-payloads/types.go](../../../../../pkg/itokens-payloads/types.go)
  - rename the `CanonicalLogin` field -> `Login` (no json tag; canonical wire claim)
  - rename the `PresentedLogin` field -> `Alias`, dropping the `json:"Login"` tag so it serializes as the `Alias` claim, empty when no alias
  - rewrite the doc comments: `Login` = canonical immutable internal identity; `Alias` = active alias snapshot, display-only

- [x] update: [itokens-payloads/consts.go](../../../../../pkg/itokens-payloads/consts.go)
  - `systemPrincipalPayload`: set `Login: "system"`, leave `Alias` empty

### Token issue and consumers

- [x] update: [registry/impl_issueprincipaltoken.go](../../../../../pkg/registry/impl_issueprincipaltoken.go)
  - set `Login = loginForSignIn.canonicalLogin` and `Alias = loginForSignIn.alias`; remove the `presentedLogin` fallback that copied the canonical login into the alias field

- [x] update: [iauthnzimpl/impl.go](../../../../../pkg/iauthnzimpl/impl.go)
  - resolve subject roles against `Login` (primary) and `Alias` (when non-empty and `!= Login`), keeping the existing dedup
  - key `loginName` on `Login`, falling back to `Alias` for legacy/system tokens

- [x] update: [n10n/impl_subscribeandwatch.go](../../../../../pkg/processors/n10n/impl_subscribeandwatch.go)
  - read the subject login from `Login`, falling back to `Alias`

- [x] update: [vit/impl.go](../../../../../pkg/vit/impl.go)
  - set `Login: prn.Name` in the VIT-issued `PrincipalPayload`; leave `Alias` empty

### Schema

- [x] update: [registry/appws.vsql](../../../../../pkg/registry/appws.vsql)
  - remove the dormant `GRANT SELECT ON TABLE Login TO sys.ProfileOwner`: no client holds `ProfileOwner` in the registry app workspace where `Login` lives, so it grants no access; removal aligns the schema with the System-only read contract
