---
change_id: 2607311008-canonical-login-state
type: feat
issue_url: https://untill.atlassian.net/browse/AIR-4634
domains: [prod]
scope: [auth]
---

# Change request: Canonical login enablement state

Refs:

- [AIR-4634: Add enable/disable state for canonical login](./issue-AIR-4634.md)

## Why

The auth context needs a reversible way to suspend use of a Login's canonical SignInIdentifier without destroying its identity or releasing that identifier. This allows the canonical sign-in and recovery entry points to be blocked while an active LoginAlias and all other Login operations continue unchanged.

## What

The authentication behavior gains an explicit `CanonicalLoginEnablement` state:

- System-authorized operations can disable and re-enable the canonical Login idempotently while retaining the Login identity, LoginAlias, Credential, ProfileWorkspace, and WorkspaceMembership records.
- System and the WorkspaceOwner of the target registry workspace can read the full Login CDoc, including canonical enablement and LoginAlias lifecycle state, through the existing workspace-scoped CDoc query.
- A disabled canonical Login rejects PrincipalToken issue and password-reset initiation when its canonical SignInIdentifier is submitted.
- An active LoginAlias remains available for PrincipalToken issue and password reset while the canonical Login is disabled.
- Password change, password-reset completion, and PrincipalToken validation and renewal are unaffected by canonical Login disablement.
- Public canonical sign-in and recovery-initiation operations treat a disabled canonical Login like an unknown Login or invalid Credential, without disclosing its disabled state.
- The disabled canonical SignInIdentifier remains reserved, so creating another Login with that identifier remains a conflict.
- Re-enabling restores the two canonical entry operations with the retained Credential and profile state.

## How

Decisions:

- Store `CanonicalLoginEnablement` as an optional, default-enabled state on the active registry Login, independent of `sys.IsActive`; existing Login records therefore remain enabled without a data migration.
- Keep the Login, Login index, and LoginAlias index active while the canonical Login is disabled. System-only enable and disable operations update only the requested state synchronously and succeed when that state is already set.
- Apply `CanonicalLoginEnablement` as an eligibility gate only when `IssuePrincipalToken` or `InitiateResetPasswordByEmail` resolves the submitted value directly as the canonical SignInIdentifier.
- Do not apply that eligibility gate after LoginAlias resolution or during password change, password-reset completion, PrincipalToken validation, refresh, or enrichment.
- Map rejected canonical entry operations to each endpoint's existing unknown-login or invalid-Credential response.
- Reuse `q.sys.GetCDoc` for Login state reads: System and the WorkspaceOwner of the target registry workspace can read the full Login CDoc, while owners of other workspaces and callers with neither authorization are rejected.
- Verify the contract through the authentication feature and integration coverage for Login CDoc read authorization, direct canonical rejection, unaffected alias routing, a reset completed after disablement, identifier reservation, and restoration after re-enabling.

Out of scope:

- Account-wide suspension, disabling an active LoginAlias without clearing it, or changing existing credential and PrincipalToken behavior.
- Immediate revocation of already-issued PrincipalTokens or a registry state lookup on authenticated requests.
- Bulk enablement-state management, a field-scoped canonical-enablement read API, and administrative user interfaces above the System APIs.

References:

- [authentication state and identifier resolution](../../../../../uspecs/specs/prod/auth/arch-authn.md)
- [authorization role composition](../../../../../uspecs/specs/prod/auth/arch-authz.md)
- [principal token lifecycle and validation](../../../../../uspecs/specs/prod/auth/arch-tokens.md)
- [registry authentication schema and API surface](../../../../../pkg/registry/appws.vsql)
- [principal-token issue and alias resolution](../../../../../pkg/registry/impl_issueprincipaltoken.go)
- [password-reset initiation and alias resolution](../../../../../pkg/registry/impl_resetpassword.go)
- [existing deactivation and credential guards](../../../../../pkg/sys/it/impl_deactivateworkspace_test.go)

## Domain design

- [x] create: [prod/auth/context.md](../../../../specs/prod/auth/context.md)
  - Bounded Context Specification for authentication: CanonicalLoginEnablement lifecycle, direct canonical entry constraints, unaffected LoginAlias behavior, and relationships with Credential, PrincipalToken, ProfileWorkspace, and WorkspaceMembership

## Functional design

- [x] update: [prod/auth/authn.feature](../../../../specs/prod/auth/authn.feature)
  - add: Canonical Login enablement rule covering System-only mutation and idempotent disable and enable operations
  - update: Login state visibility -> use one authorization rule for full Login CDoc reads, including canonical enablement and LoginAlias lifecycle fields
  - add: disabled canonical Login rule -> reject only direct canonical PrincipalToken issue and password-reset initiation without disclosing disablement
  - add: active LoginAlias scenarios -> sign-in and password reset remain available while the canonical Login is disabled
  - add: in-progress reset scenario -> a reset initiated before canonical disablement can complete afterward
  - add: identifier reservation and re-enablement scenarios -> the canonical identifier remains a conflict and its two entry operations are restored after re-enabling

## Technical design

- [x] update: [prod/auth/arch.md](../../../../specs/prod/auth/arch.md)
  - update: shared `registry.Login` concept -> include the canonical enablement state stored independently of `sys.IsActive`

- [x] update: [prod/auth/arch-authn.md](../../../../specs/prod/auth/arch-authn.md)
  - add: System-authorized `SetCanonicalLoginEnablement` with idempotent state transitions
  - reuse: `q.sys.GetCDoc` for full Login state reads by System or the target registry WorkspaceOwner
  - update: `IssuePrincipalToken` resolution -> reject a disabled direct canonical match without changing active LoginAlias resolution
  - update: password-reset resolution -> check canonical enablement only for direct canonical initiation; keep alias initiation and reset completion unchanged
  - update: error mapping -> disabled canonical entry points remain indistinguishable from existing unknown-login or invalid-Credential failures

- [x] update: [prod/auth/authn--td.md](../../../../specs/prod/auth/authn--td.md)
  - update: registry schema and operation inventory -> store default-enabled `CanonicalLoginEnablement`, add its System-only setter, and reuse workspace-owner-authorized `q.sys.GetCDoc` for reads
  - add: component interactions for direct canonical gating in PrincipalToken issue and password-reset initiation, including unaffected alias paths
  - add: integration coverage for Login CDoc read authorization, idempotent management, non-disclosing canonical rejection, unaffected alias sign-in and reset, in-progress reset completion, identifier reservation, and re-enablement

## Construction

### Tests

- [x] create: [sys/it/authn_test.go](../../../../../pkg/sys/it/authn_test.go)
  - executable integration-test counterpart of `authn.feature`, with top-level `TestAuthn_<Rule>` tests and exact `authn: scn:` scenario traceability
  - consolidate authentication feature scenarios and their local test setup from the former implementation-oriented test files, eliminating logical duplication
  - cover full Login CDoc reads through `q.sys.GetCDoc` for System, the target registry WorkspaceOwner, a WorkspaceOwner of another workspace, and a caller with neither authorization
  - cover System-only idempotent `SetCanonicalLoginEnablement`, including default-enabled existing Login records, disabled canonical sign-in and password-reset initiation, unaffected active LoginAlias operations, in-progress reset completion, identifier reservation, and restoration after re-enabling

- [x] update: [sys/it/impl_changepassword_test.go](../../../../../pkg/sys/it/impl_changepassword_test.go)
  - move `authn.feature` password-change scenario coverage to `authn_test.go`; retain implementation-specific command and rate-limit tests

- [x] update: [sys/it/impl_cpv2_test.go](../../../../../pkg/sys/it/impl_cpv2_test.go)
  - move REST user-login creation feature scenarios to `authn_test.go`; retain command-processor-specific tests

- [x] update: [sys/it/impl_deactivateworkspace_test.go](../../../../../pkg/sys/it/impl_deactivateworkspace_test.go)
  - move authentication scenarios for deactivated and recreated Logins to `authn_test.go`; retain workspace-deactivation implementation tests and shared helpers

- [x] update: [sys/it/impl_qpv2_test.go](../../../../../pkg/sys/it/impl_qpv2_test.go)
  - move authentication login and principal-token refresh feature scenarios to `authn_test.go`; retain query-processor-specific tests

- [x] update: [sys/it/impl_resetpassword_test.go](../../../../../pkg/sys/it/impl_resetpassword_test.go)
  - move password lifecycle and disabled canonical Login reset scenarios to `authn_test.go`; retain password-reset limits and shared reset drivers

- [x] update: [sys/it/impl_signupin_test.go](../../../../../pkg/sys/it/impl_signupin_test.go)
  - move login creation, LoginAlias, Login state, canonical enablement, device, and sign-in feature scenarios to `authn_test.go`; retain technical sign-up and LoginAlias edge-case tests and shared helpers

### Registry schema and management

- [x] update: [registry/appws.vsql](../../../../../pkg/registry/appws.vsql)
  - add: optional `CanonicalLoginDisabled bool` to `registry.Login`, preserving `Enabled` for existing records when the field is absent or `false`
  - add: `SetCanonicalLoginEnablementParams` with `Login`, `AppName`, and `Enabled bool`, declare the command, and grant execution only to `sys.System`

- [x] update: [registry/consts.go](../../../../../pkg/registry/consts.go)
  - add: `CanonicalLoginDisabled` field name and `SetCanonicalLoginEnablement` command QName constants

- [x] update: [registry/provide.go](../../../../../pkg/registry/provide.go)
  - register the `SetCanonicalLoginEnablement` command implementation

- [x] create: [registry/impl_canonicalloginenablement.go](../../../../../pkg/registry/impl_canonicalloginenablement.go)
  - Canonical Login enablement mutation and shared eligibility policy
  - implement `provideCanonicalLoginEnablement` and `execCmdSetCanonicalLoginEnablement` to validate direct canonical routing, resolve the Login, and idempotently write `CanonicalLoginDisabled = !Enabled` without changing other Login state
  - implement one `isCanonicalLoginEnabled` predicate that maps an absent or `false` field to enabled and `true` to disabled
  - verify management authorization, idempotence, default-enabled compatibility, and shared eligibility behavior through the preceding Construction test items

### Canonical entry eligibility

- [x] update: [registry/impl_issueprincipaltoken.go](../../../../../pkg/registry/impl_issueprincipaltoken.go)
  - apply `isCanonicalLoginEnabled` only after a direct canonical Login match and reuse `errLoginOrPasswordIsIncorrect` when disabled
  - leave the active LoginAlias resolution path and PrincipalToken payload behavior unchanged

- [x] update: [registry/impl_resetpassword.go](../../../../../pkg/registry/impl_resetpassword.go)
  - apply `isCanonicalLoginEnabled` only after a direct canonical Login match in `InitiateResetPasswordByEmail` and reuse the existing unknown-login error when disabled
  - return the canonical Login pseudo WSID from direct reset initiation, matching the alias path and the `CanonicalPseudoWSID` result contract
  - leave active LoginAlias initiation, verified-token issue, and final password mutation unchanged so an in-progress reset can complete after disablement
