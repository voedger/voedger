---
change_id: 2608071450-alias-deactivated-profile
type: test
issue_url: https://untill.atlassian.net/browse/AIR-4675
domains: [prod]
scope: [auth]
---

# Change request: Login alias reuse after profile deactivation

Refs:

- [AIR-4675: implement an integration test that will check setting alias to the email of a deactivated profile](./issue-AIR-4675.md)

## Why

Integration coverage does not establish whether a deactivated Profile Workspace's Login can safely become the Login Alias of another active Login. Coverage is needed to guard against resolving the alias to the deactivated profile or exposing its workspace-scoped data.

## What

Add integration-level coverage demonstrating that:

- A deactivated Profile Workspace's Login identifier can be assigned as the Login Alias of another active Login.
- Sign-in through the reused Login Alias resolves the Principal to the active Login's Profile Workspace.
- Workspace-scoped data accessed through the alias belongs to the active profile and excludes data from the deactivated profile.

## How

Decisions:

- Express the behavior as an Authentication feature scenario and preserve one-to-one Gherkin-to-Go integration-test traceability.
- Exercise profile deactivation, Login Alias projection, sign-in, and child-workspace resolution through their public integration surfaces and established asynchronous wait helpers instead of directly mutating registry state.
- Use the alias-issued Principal Token as the authorization context for all post-alias profile and workspace reads; reserve privileged authorization for the System-managed alias operation.

Assumptions:

- None

References:

- [authentication behavior specification](../../../../../uspecs/specs/prod/auth/authn.feature)
- [authentication integration-test patterns](../../../../../pkg/sys/it/authn_test.go)
- [Login Alias integration helpers](../../../../../pkg/sys/it/impl_signupin_test.go)
- [workspace deactivation integration-test patterns](../../../../../pkg/sys/it/impl_deactivateworkspace_test.go)
- [profile-scoped child-workspace fixtures and resolution](../../../../../pkg/vit/utils.go)

## Functional design

- [x] update: [prod/auth/authn.feature](../../../../specs/prod/auth/authn.feature)
  - add: scenario proving that a deactivated Login identifier can become another active Login's Login Alias and resolves only that active Login's Profile Workspace and child-workspace data

## Construction

- [x] update: [sys/it/authn_test.go](../../../../../pkg/sys/it/authn_test.go)
  - add: integration subtest for "Deactivated Login identifier can become another Login Alias without exposing profile data", preserving the scenario name and verbatim step comments
  - arrange: two User Logins with same-named child Workspaces containing distinct values, then deactivate one Profile Workspace and await completion
  - act: assign the deactivated Login identifier as the active Login's Login Alias, await projection completion, and sign in through the alias
  - verify: the issued Principal Token identifies the active Login and Profile Workspace, and alias-authorized child-workspace reads return only the active profile's value
