---
change_id: 2608251223-align-authn-spec-test-traceability
type: refactor
issue_url: https://untill.atlassian.net/browse/AIR-4655
domains: [prod]
scope: [auth]
---

# Change request: Authentication and invitation specification traceability alignment

Refs:

- [AIR-4655: voedger: align authn feature, technical design, and test traceability](./issue-AIR-4655.md)

## Why

Authentication scenarios, their technical design, and their integration-test traceability have drifted apart, while the invitations technical design lacks per-scenario flows. Aligning both features makes their behavior easier to review and protects specification-to-design-to-test mappings from future drift.

## What

- The refactor makes no behavior change: existing authentication and invitation API behavior, principal token semantics, invitation lifecycle, and compatibility guarantees remain unchanged.
- Every authentication and invitation Rule and Scenario has an exact, case-sensitive technical-design counterpart in the same order and grouping.
- Every specified authentication and invitation scenario has uniquely identifiable integration coverage; authentication scenarios additionally use isolated fixtures and a strict feature/technical test-file boundary.
- Automated conformance validation detects missing, extra, reordered, mis-grouped, case-different, or untraceable authentication and invitation scenarios.

## How

Decisions:

- Realize authentication scenario isolation with a fresh, lazily constructed VIT fixture per scenario and `t.Cleanup`, avoiding feature-wide mutable test state.
- Use a neutral package-level authn helper layer for operations and assertions shared by feature, technical, or invites suites; keep scenario fixture builders feature-local and technical-only helpers with their owning suite.
- Add Rule-grouped per-scenario flows to `invites--td.md` in the exact order of `invites.feature`, while retaining its existing data, state-machine, projector, and consistency sections as shared technical context referenced by those flows.
- Put reusable conformance logic in `pkg/goutils/testingu/featureconformance`, exposing a configuration with the feature name, feature and technical-design paths, feature and technical-test paths, and opt-in policy flags; derive the scenario prefix from the feature name.
- Give each adopted feature a thin runner near its tests. For this change, add authn and invites runners in `pkg/sys/it/authn_conformance_test.go` and `pkg/sys/it/invites_conformance_test.go`; both check ordered technical-design identities, literal scenario tags, verbatim step comments, and exact Scenario Outline rows and placeholder mappings, while authn additionally enables the strict scenario-only feature-file policy. Use standard-library line and Go syntax parsing without initializing VIT or adding a Gherkin dependency.

Exact test migration map:

- Preserve the complete pre-refactor authn inventory: the nine `TestAuthn_*` entry points become one `TestAuthn` feature entry point plus eight `TestAuthnTechnical_*` entry points; all 71 literal `authn: scn:` case identities remain byte-for-byte identical, and all 23 untagged technical cases remain covered. The old malformed-login transport case additionally retains its nine dynamically named request-body subtests.
- Move feature scenario cases from the former entry points into Rule-owned registration functions called by `TestAuthn`:
  - `TestAuthn_LoginCreation`: 12 cases -> `testAuthnLoginCreation` (8), `testAuthnSignInAndProfileReadiness` (1), `testAuthnPrincipalTokenContract` (1), and `testAuthnExceptionFlows` (2).
  - `TestAuthn_LoginAliasManagement`: 18 cases -> `testAuthnLoginAliasManagement` (9), `testAuthnLoginCreation` (1), `testAuthnSignInAndProfileReadiness` (4), and `testAuthnPrincipalTokenContract` (4).
  - `TestAuthn_LoginStateVisibility`: 5 cases -> `testAuthnLoginStateVisibility` (5).
  - `TestAuthn_CanonicalLoginEnablementManagement`: 6 cases -> `testAuthnCanonicalLoginEnablementManagement` (6).
  - `TestAuthn_DisabledCanonicalLoginBehavior`: 9 cases -> `testAuthnDisabledCanonicalLoginBehavior` (9).
  - `TestAuthn_SignInAndProfileReadiness`: 6 cases -> `testAuthnSignInAndProfileReadiness` (3), `testAuthnPrincipalTokenContract` (2), and `testAuthnExceptionFlows` (1).
  - `TestAuthn_PrincipalTokenContract`: 4 cases -> `testAuthnPrincipalTokenContract` (2) and `testAuthnExceptionFlows` (2).
  - `TestAuthn_PasswordLifecycle`: 10 cases -> `testAuthnPasswordLifecycle` (10).
  - `TestAuthn_ExceptionFlows`: 1 case -> `testAuthnExceptionFlows` (1).
- Move or rename every former untagged authn technical case as follows:
  - `wrong AppWSID`, `unknown application`, `wrong application name`, and `allowed special chars in login` -> the same-named subtests under `TestAuthnTechnical_LoginCreation`.
  - `passwords with special JSON characters` -> the same-named subtest under `TestAuthnTechnical_PasswordAndLoginTransport`; `Bad request` -> `login rejects malformed transport`; `Login with special JSON characters in password` -> `login with special JSON characters in password`. The malformed-transport matrix preserves all nine old bodies, including the `UnknownField` + real login + `badpwd` body.
  - The two old `400 bad request on bad appQName` cases -> `TestAuthnTechnical_ResetPasswordTransport/initiation rejects malformed app QName` and `TestAuthnTechnical_ResetPasswordTransport/verified-token issue rejects malformed app QName` respectively.
  - `Old token` -> `TestAuthnTechnical_PrincipalToken/expired token cannot be refreshed`; `wrong password` -> `TestAuthnTechnical_PrincipalToken/direct token query rejects wrong password`.
  - `410 Gone on work in deactivated profile` -> `TestAuthnTechnical_DeactivatedLoginCommands/work in deactivated profile returns 410`; `c.registry.ChangePassword -> 401`, `q.registry.InitiateResetPasswordByEmail -> 400`, `c.registry.ResetPasswordByEmail -> 401`, and `c.registry.UpdateGlobalRoles -> 401` -> the corresponding `ChangePassword`, `InitiateResetPasswordByEmail`, `ResetPasswordByEmail`, and `UpdateGlobalRoles` deactivated-login subtests.
  - `wrong password through an active alias is rejected`, `setting the same alias is idempotent`, `clearing when no alias is set is idempotent`, and `cleared alias can be reused by another login` -> the same-named subtests under `TestAuthnTechnical_LoginAlias`.
  - `exec a simple operation in the device profile` and `refresh the device principal token` -> the same-named subtests under `TestAuthnTechnical_DevicePrincipal`.
  - `existing Login without stored state is enabled` -> the standalone `TestAuthnTechnical_DefaultCanonicalLoginState` entry point.
- Preserve assertion-level coverage that is easy to lose during fixture isolation: alias-collision creation cases still prove that the existing login or alias owner authenticates and the colliding password is rejected; alias-based password reset still asserts that `CanonicalPseudoWSID` is the canonical login workspace; malformed-login transport still checks every old body and expected validation fragment.
- Preserve the non-authn suites exactly while moving only shared plumbing:
  - `impl_signupin_test.go` retains its five top-level tests and eight literal subtests; its 15 shared alias, login-state, and principal-token helpers move by the same function names to `authn_test_helpers_test.go`.
  - `impl_resetpassword_test.go` retains `TestResetPasswordLimits` and its two literal subtests; its five shared reset-password helpers move by the same function names to `authn_test_helpers_test.go`.
  - `impl_invites_feature_test.go` retains `TestInvites` and all 40 literal subtests unchanged: 36 invitation scenario cases and four explicitly out-of-scope technical regressions.
- Add rather than migrate `featureconformance` unit coverage and the authn and invites conformance runners; these source-only checks protect the migration inventory without replacing any executable behavior test.

Assumptions:

- None

Out of scope:

- Relocating existing untagged technical regression cases from the invitations feature-test file.
- Adding conformance runners to feature suites other than authn and invites.

References:

- [canonical authentication scenarios](../../../../../uspecs/specs/prod/auth/authn.feature)
- [current authentication technical flows](../../../../../uspecs/specs/prod/auth/authn--td.md)
- [canonical invitation scenarios](../../../../../uspecs/specs/prod/auth/invites.feature)
- [current invitation technical design](../../../../../uspecs/specs/prod/auth/invites--td.md)
- [current mixed authentication integration suite](../../../../../pkg/sys/it/authn_test.go)
- [isolated feature-fixture pattern](../../../../../pkg/sys/it/impl_invites_feature_test.go)
- [technical invitation regressions](../../../../../pkg/sys/it/impl_invite_test.go)
- [shared sign-in and alias test plumbing](../../../../../pkg/sys/it/impl_signupin_test.go)
- [shared password-reset test plumbing](../../../../../pkg/sys/it/impl_resetpassword_test.go)
- [shared Go testing utilities](../../../../../pkg/goutils/testingu)
- [repository feature-test traceability rules](../../../../../.claude/skills/feature-tests/SKILL.md)

## Technical design

- [x] update: [prod/auth/authn--td.md](../../../../specs/prod/auth/authn--td.md)
  - fix: Rule and Scenario or Scenario Outline identities to match `authn.feature` exactly, case-sensitively, and in the same order and grouping
  - merge: the split alias-collision entries into one Scenario Outline flow with its branches represented inside that flow
  - add: technical flows for the four authentication scenarios currently missing from the technical design

- [x] update: [prod/auth/invites--td.md](../../../../specs/prod/auth/invites--td.md)
  - add: Rule-grouped technical flows for every Scenario and Scenario Outline in `invites.feature`, with exact case-sensitive identities and matching order
  - retain: existing data, state-machine, projector, versioning, federation, and consistency design as shared technical context for the scenario flows

## Construction

### Tests

- [x] create: [featureconformance/conformance_test.go](../../../../../pkg/goutils/testingu/featureconformance/conformance_test.go)
  - unit coverage for the reusable feature-conformance utility using temporary feature, technical-design, and Go source fixtures
  - verify acceptance of exact Rule and Scenario order, literal scenario tags, verbatim step comments, Scenario Outline headers and rows, and placeholder mappings
  - verify diagnostics for missing, extra, duplicate, ambiguously disambiguated, reordered, mis-grouped, case-different, dynamically constructed, or otherwise untraceable scenarios, plus the opt-in scenario-only feature-file policy
  - prove the checks are source-only and require neither VIT initialization nor a Gherkin dependency

- [x] create: [sys/it/authn_test_helpers_test.go](../../../../../pkg/sys/it/authn_test_helpers_test.go)
  - neutral package-level authentication test operations and assertions shared by feature, technical, reset-password, sign-in, alias, or invitations coverage
  - extract only cross-suite plumbing from the existing authn-related test files; keep the authn scenario fixture in `impl_authn_feature_test.go` and technical-only helpers with their owning suites
  - call `t.Helper()` from every helper that can fail a test

- [x] create: [sys/it/impl_authn_feature_test.go](../../../../../pkg/sys/it/impl_authn_feature_test.go)
  - executable counterpart of `authn.feature` with one top-level `TestAuthn` and explicit direct subtests for all 50 scenario identities
  - move every `authn: scn:` case from `authn_test.go`, preserving exact scenario names, row disambiguators, verbatim step comments, byte-for-byte Scenario Outline header and row comments, and placeholder mappings
  - construct a fresh lazy authn feature fixture per scenario and register teardown with `t.Cleanup`, so no scenario depends on another subtest's state or execution order
  - contain only scenario-tagged feature cases and feature-local fixture builders; retain implementation, compatibility, malformed-transport, and other untagged regressions outside this file

- [x] update: [sys/it/authn_test.go](../../../../../pkg/sys/it/authn_test.go)
  - move all feature-scenario cases to `impl_authn_feature_test.go` and remove every `authn: scn:` name from this technical test file
  - retain and organize untagged implementation, compatibility, malformed-transport, internal-state, JSON-escaping, old-token, direct-command, and additional edge-case coverage
  - replace cross-suite helper definitions with calls to the neutral helper layer while keeping technical-only helpers local

- [x] update: [sys/it/impl_signupin_test.go](../../../../../pkg/sys/it/impl_signupin_test.go)
  - move shared sign-in, Login Alias, principal-token, and Login-state operations or assertions to `authn_test_helpers_test.go`
  - retain implementation-specific sign-up/sign-in and Login Alias edge tests plus helpers used only by this suite

- [x] update: [sys/it/impl_resetpassword_test.go](../../../../../pkg/sys/it/impl_resetpassword_test.go)
  - move shared password-reset operations and assertions to `authn_test_helpers_test.go`
  - retain password-reset limit regressions and helpers used only by this technical suite

- [x] update: [sys/it/impl_invites_feature_test.go](../../../../../pkg/sys/it/impl_invites_feature_test.go)
  - align every invitations scenario subtest with the exact `invites.feature` identity, verbatim step comments, byte-for-byte Scenario Outline rows, and placeholder mappings required by the conformance runner
  - preserve the existing isolated per-scenario fixture and invitation behavior coverage
  - retain the existing untagged invitation technical regressions in this file as explicitly out of scope for relocation

- [x] create: [sys/it/authn_conformance_test.go](../../../../../pkg/sys/it/authn_conformance_test.go)
  - thin, source-only `TestAuthnConformance` runner over `authn.feature`, `authn--td.md`, `impl_authn_feature_test.go`, and every other `pkg/sys/it/*_test.go` source as the technical-test set
  - invoke `featureconformance.Test` with feature name `authn` and enable the strict policy that the feature test file contains every and only scenario-tagged subtests
  - fail on authn scenario tags in configured technical-test files without constructing VIT

- [x] create: [sys/it/invites_conformance_test.go](../../../../../pkg/sys/it/invites_conformance_test.go)
  - thin, source-only `TestInvitesConformance` runner over `invites.feature`, `invites--td.md`, `impl_invites_feature_test.go`, and `impl_invite_test.go`
  - invoke `featureconformance.Test` with feature name `invites` while allowing the existing untagged technical regressions in the feature-test file
  - validate invitation TD identities, scenario tags, step comments, Scenario Outline rows, and placeholder mappings without constructing VIT

### Shared conformance utility

- [x] create: [featureconformance/conformance.go](../../../../../pkg/goutils/testingu/featureconformance/conformance.go)
  - reusable source-only conformance checker for Gherkin-backed Go feature tests
  - expose `Config` with feature name, feature and technical-design paths, feature and technical-test path lists, and opt-in policy flags, plus `Test(t, Config)` as the runner entry point; derive the `<feature>: scn:` prefix from the feature name
  - parse ordered Rule, Scenario, Scenario Outline, step, Examples-row, and placeholder identities with standard-library line processing, and parse literal Go subtest names and associated comments with `go/parser`, `go/ast`, and `go/token`
  - report precise file and identity diagnostics for TD drift, missing, duplicate, ambiguous, or unknown coverage, misplaced or non-literal scenario tags, non-verbatim steps, incomplete outline rows or mappings, and strict feature-file violations

- [x] update: [testingu/README.md](../../../../../pkg/goutils/testingu/README.md)
  - list the `featureconformance` package and its purpose alongside the existing testing utilities
  - document the minimal `featureconformance.Config` and `featureconformance.Test` usage pattern for future feature runners
