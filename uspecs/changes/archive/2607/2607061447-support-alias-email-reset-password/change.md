---
change_id: 2607021603-support-alias-email-reset-password
type: feat
issue_url: https://untill.atlassian.net/browse/AIR-4376
domains: [prod]
---

# Change request: Reset password by alias email

Refs:

- [AIR-4376: voedger: support alias email in reset-password by email flow](./issue-AIR-4376.md)
- [rsch.md](../../../archive/2606/2606301442-reset-password-by-alias/rsch.md)

## Why

Users can sign in with an active login alias, but the reset-password-by-email flow still treats the submitted email as a primary login only. This prevents an alias-email holder from resetting the password for the account that owns the alias, even though alias ownership and routing are already represented in the registry domain. The feature must preserve verification as the ownership proof and avoid trusting client-supplied canonical-login identifiers.

## What

Enable password reset to work when the supplied email is an active login alias, not only the primary login:

- A reset initiated with an alias email delivers the verification code to the alias inbox
- After verification, the password of the account that owns the alias is updated, and the user can sign in with the new password
- The primary-login reset flow continues to work unchanged
- A reset attempt using a previously assigned or cleared alias is rejected
- Verification remains the sole proof of ownership; resolving an alias never lets an unverified value alter another account's password
- The final reset command runs in the canonical login workspace selected by the returned `CanonicalPseudoWSID`

## How

Decisions:

- Extend the existing three-step reset flow in `pkg/registry/impl_resetpassword.go`; do not introduce a new password-reset command.
- Resolve alias emails server-side during `q.registry.InitiateResetPasswordByEmail` by falling back from `LoginIdx` to the local active `LoginAlias` row and returning `CanonicalPseudoWSID`.
- Keep verifier decoupling in `q.registry.IssueVerifiedValueTokenForResetPassword`: prove the code against the alias email, then issue the registry verified-value token for the canonical login.
- Leave `c.registry.ResetPasswordByEmail` as the local password write; the client routes it to the returned canonical pseudo workspace.
- Cover the feature with authn specs plus an integration test in the existing reset-password test file.

Out of scope:

- A password-bearing query wrapper around reset-password.
- A client-supplied canonical login or LoginID parameter.
- A cross-workspace password write command from the alias workspace.
- Frontend routing changes; those are tracked by AIR-4372.

References:

- [reset-password implementation](../../../../../pkg/registry/impl_resetpassword.go)
- [registry schema and reset result type](../../../../../pkg/registry/appws.vsql)
- [login-alias index implementation](../../../../../pkg/registry/impl_setloginalias.go)
- [reset-password integration tests](../../../../../pkg/sys/it/impl_resetpassword_test.go)
- [authn technical design](../../../../specs/prod/auth/authn--td.md)
- [authn functional scenarios](../../../../specs/prod/auth/authn.feature)
- [reset-password alias research](../../../archive/2606/2606301442-reset-password-by-alias/rsch.md)

## Functional design

- [x] update: [prod/auth/authn.feature](../../../../specs/prod/auth/authn.feature)
  - add: scenario for resetting a password by verified alias email, including code delivery to the alias inbox and successful sign-in with the new password
  - add: scenario for rejecting password reset initiation with a previous or cleared alias

## Technical design

- [x] update: [prod/auth/authn--td.md](../../../../specs/prod/auth/authn--td.md)
  - add: scenario for resetting a password by verified alias email as a sibling of the primary email reset scenario
  - update: `[/q.registry.InitiateResetPasswordByEmail/]` behavior to document primary-login lookup, alias fallback through active `[(registry.LoginAlias)]`, canonical login/profile resolution, and returned `CanonicalPseudoWSID`
  - update: `[/q.registry.IssueVerifiedValueTokenForResetPassword/]` behavior to document re-issuing the verified-value token for the canonical login after the alias email code is verified
  - document that `[/c.registry.ResetPasswordByEmail/]` remains unchanged and runs in the canonical login workspace selected by the client

## Construction

### Tests

- [x] update: [sys/it/impl_resetpassword_test.go](../../../../../pkg/sys/it/impl_resetpassword_test.go)
  - add: integration coverage for reset by active alias email, including code delivery to the alias inbox
  - add: assert `InitiateResetPasswordByEmailResult.CanonicalPseudoWSID` selects the canonical login workspace
  - add: route `c.registry.ResetPasswordByEmail` to `CanonicalPseudoWSID`, then verify the canonical login can sign in with the new password
  - add: initiation rejection coverage for a replaced alias and a cleared alias

### Schema

- [x] update: [registry/appws.vsql](../../../../../pkg/registry/appws.vsql)
  - add: `CanonicalPseudoWSID int64 NOT NULL` to `InitiateResetPasswordByEmailResult`

### Registry implementation

- [x] update: [registry/impl_resetpassword.go](../../../../../pkg/registry/impl_resetpassword.go)
  - update: `provideQryInitiateResetPasswordByEmailExec` to return `CanonicalPseudoWSID` for the primary-login path
  - update: `provideQryInitiateResetPasswordByEmailExec` to fall back from `LoginIdx` miss to active `LoginAlias`, read the canonical `Login` through `q.sys.GetCDoc`, and start verification for the submitted alias email
  - update: `provideIssueVerifiedValueTokenForResetPasswordExec` to re-read active `LoginAlias` after code verification and re-issue the registry verified-value token with the canonical login value
  - keep: `cmdResetPasswordByEmailExec` as the local password write in the workspace selected by `CanonicalPseudoWSID`
