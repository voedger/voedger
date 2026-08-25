# voedger: align authn feature, technical design, and test traceability

- URL: https://untill.atlassian.net/browse/AIR-4655
- ID: AIR-4655
- State: in-progress
- Author: Maksim Geraskin
- Assignees: Maksim Geraskin
- Labels: authn, uspecs, voedger

## Goal

Align the authentication Functional Design, Feature Technical Design, and Go integration-test traceability with the uSpecs and repository `feature-tests` skill rules.

## Artifacts

* `uspecs/specs/prod/auth/authn.feature`
* `uspecs/specs/prod/auth/authn--td.md`
* `pkg/sys/it/authn_test.go`
* new `pkg/sys/it/impl_authn_feature_test.go`
* optional shared `pkg/sys/it/authn_test_helpers_test.go`

The test-file organization should follow the direction established by `pkg/sys/it/impl_invites_feature_test.go` and `pkg/sys/it/impl_invite_test.go`, with a stricter boundary between feature and technical tests.

## Current state

`authn.feature` contains 50 scenarios. The TD contains 47 scenario entries.

A case-sensitive comparison found 10 TD headings that do not exactly match a feature scenario and 13 feature scenario names without an exact TD heading. Most are wording/casing drift, one Scenario Outline was split into two TD headings, and four scenarios are genuinely absent from the TD.

The four missing TD scenarios are:

* `Login creation succeeds for a deactivated login name`
* `User signs in with original login while alias is active`
* `Principal token carries the canonical login and the active alias`
* `Sign-in rejects a deactivated login with the same error as a missing login`

The collision Scenario Outline `Alias creation or update rejects a colliding identifier` is represented by two non-matching TD headings and should be represented by one exact scenario entry with branching inside its single flow diagram.

The Go tests already contain scenario-tagged subtests for all 50 scenarios. All 71 `authn: scn:` names start with an exact, case-sensitive feature scenario name, and the Gherkin step and Examples-row comments are present verbatim.

However, feature-scenario tests and implementation-oriented tests are interleaved in `authn_test.go`. Rule-specific top-level tests also execute scenarios belonging to other Rules, so the complete Go test path does not consistently reflect the feature structure. Some helpers reuse mutable state across subtests, making clean scenario isolation and ownership harder to see.

## Selected implementation approach

Treat `authn.feature` as canonical and use an invite-style feature/technical test split.

### Feature tests: `impl_authn_feature_test.go`

* Define one top-level `TestAuthn` feature test.
* Put every scenario test under it as an explicit `t.Run("authn: scn: <exact scenario name>[: <row disambiguator>]", ...)`.
* Keep only tests that realize a scenario from `authn.feature`.
* Preserve verbatim Gherkin step comments, exact Scenario Outline header/row comments, and placeholder mappings.
* Group construction helpers by Rule if useful, but keep scenario subtests directly under the truthful feature root.
* Give each scenario an isolated fixture/state unless sharing is an explicit infrastructure requirement; scenario correctness must not depend on subtest execution order.

### Technical tests: `authn_test.go`

* Keep implementation, compatibility, malformed-transport, internal-state, and additional edge-case coverage that has no matching feature scenario.
* Examples include wrong AppWSID/unknown application, JSON escaping, compatibility or old-token cases, direct command/status checks, stored representation checks, and implementation-specific idempotency cases.
* Do not use `scn:` in technical test names.
* If a technical case actually expresses required user-facing behavior, add or identify the feature scenario and move the case to `impl_authn_feature_test.go`.

### Shared helpers: `authn_test_helpers_test.go`

* Move low-level VIT setup, feature fixtures, authn operations, and reusable assertions here when they are used by both files.
* Helpers must use `t.Helper()` when they can fail a test.
* Feature tests may reuse technical plumbing, but must not depend on a technical test function or on state created by another test.

### TD alignment

1. Make every TD Rule and Scenario/Scenario Outline heading match the feature byte-for-byte and case-sensitively.
2. Merge the two collision TD entries into the single matching Scenario Outline entry.
3. Add the four missing TD scenario flows.
4. Add a static conformance check to prevent future drift.

Do not copy the current invites split mechanically: `impl_invites_feature_test.go` still contains some untagged technical regression cases. For authn, enforce the stricter rule that the feature test file contains every and only feature-scenario tests.

## Acceptance criteria

* Every Rule heading and Scenario/Scenario Outline heading in `authn--td.md` matches `authn.feature` exactly and appears under the matching Rule.
* The TD has one entry per feature scenario, including one entry per Scenario Outline rather than one entry per branch or Examples row.
* All four currently missing TD scenarios have technical flows.
* `impl_authn_feature_test.go` exists with one `TestAuthn` root.
* Every `authn: scn:` subtest is located in `impl_authn_feature_test.go`.
* Every feature scenario has at least one matching subtest; each Scenario Outline Examples row has explicit, uniquely disambiguated coverage.
* Every `authn: scn:` name equals a feature scenario name or starts with the exact scenario name followed by a brief row/case disambiguator.
* Scenario Outline tests retain byte-for-byte Examples header/row comments and placeholder mappings required by the `feature-tests` skill.
* `impl_authn_feature_test.go` contains no untagged technical test cases.
* `authn_test.go` contains no `scn:` test names.
* Scenario tests do not depend on cross-subtest ordering or mutable state produced by another scenario.
* Shared failing helpers call `t.Helper()`.
* An automated check fails on missing, extra, reordered/mis-grouped, or case-different TD scenario headings and on missing or unknown `scn:` test names.

