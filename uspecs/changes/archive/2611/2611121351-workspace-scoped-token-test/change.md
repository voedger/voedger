---
change_id: 2611121320-workspace-scoped-token-test
type: test
issue_url: https://untill.atlassian.net/browse/AIR-5246
domains: [prod]
scope: [auth]
---

# Change request: Workspace-scoped token integration coverage

Refs:

- [AIR-5246: Reseller access, voedger: implement a VIT integration test showing how to issue a token scoped to one Workspace](./issue-AIR-5246.md)

## Why

Reseller diagnostic access requires granting a role in one foreign Workspace without extending access to related Workspaces. A VIT integration test should demonstrate token issuance and verify that the authentication and authorization boundary enforces that scope through real requests.

## What

- VIT integration coverage demonstrates that a caller cannot access a newly created foreign Workspace before receiving a role grant scoped to that Workspace
- VIT integration coverage demonstrates issuing a Principal Token with a RoleGrant scoped to the foreign Workspace and successfully accessing that Workspace with the token
- VIT integration coverage verifies that the same token does not grant access to sibling Workspaces or to the Workspace that owns them

## How

Decisions:

- Keep exact-Workspace authorization in a focused VIT test, separate from the existing Subject membership, enrichment, and API-token lifecycle scenarios, while reusing their Workspace setup and token validation infrastructure
- Issue a regular Principal Token through the application's trusted token service, preserving the signed-in foreign caller's identity and adding one exact-Workspace role grant without switching to API-token semantics or creating Subject membership
- Use the foreign caller's explicit bearer token for baseline requests and the newly issued token for subsequent requests against the same protected operation, so access decisions cannot depend on VIT's default Workspace owner credentials
- Treat the existing authentication changes as a prerequisite: regular Principal Tokens must honor exact-Workspace grants alongside owned-by grants while retaining their existing identity, Subject, global-role, and descriptor processing
- Reconcile the authorization and token compatibility specifications with this newly supported token-kind and scope combination before treating the cumulative change as complete; retain unit coverage of scope matching separately from VIT coverage of HTTP access decisions

Assumptions:

- Deployment of regular Principal Tokens with exact-Workspace grants occurs only after all nodes that validate, refresh, or enrich those tokens preserve and honor explicit scopes

Out of scope:

- Adding a public token issuance endpoint or changing token enrichment delegation policy

References:

- [Foreign Workspace access and token lifecycle coverage](../../../../../pkg/sys/it/impl_childworkspace_test.go)
- [Regular and API-token principal composition](../../../../../pkg/iauthnzimpl/impl.go)
- [Scope matching regression coverage](../../../../../pkg/iauthnzimpl/impl_test.go)
- [VIT request authorization defaults](../../../../../pkg/vit/impl.go)
- [Authorization scope matching contract](../../../../specs/prod/auth/arch-authz.md)
- [Token issuance and mixed-version compatibility contract](../../../../specs/prod/auth/arch-tokens.md)

## Domain design

- [x] update: [prod/auth/context.md](../../../../specs/prod/auth/context.md)
  - update: the PrincipalToken scope-combination invariant to allow regular User and Device tokens to carry both exact-Workspace and owned-by RoleGrants, retaining API-token restricted composition
  - preserve: RoleScope meanings, RoleGrant equality, and legacy scope inference from the original token kind

## Technical design

- [x] update: [prod/auth/arch-authz.md](../../../../specs/prod/auth/arch-authz.md)
  - update: token-carried role composition, the scope matching table, and the authentication sequence to include exact-Workspace grants for regular User and Device tokens alongside owned-by grants
  - clarify: regular-token grants remain processed after the existing System short-circuits and Workspace-derived roles, require an existing Workspace descriptor, and emit deduplicated role principals in the request Workspace
  - preserve: API-token restricted composition and existing identity, Subject, global-role, and owned-by behavior, referencing the auth model for RoleScope meanings

- [x] update: [prod/auth/arch-tokens.md](../../../../specs/prod/auth/arch-tokens.md)
  - add: trusted application-bound issuance of a regular Principal Token carrying an exact-Workspace grant while preserving the caller's identity and leaving API-token composition disabled
  - update: compatibility guidance to distinguish unchanged legacy scope inference from the new regular-token exact-Workspace combination, requiring scope-aware validators and scope-preserving refresh and enrichment nodes before issuing that combination
  - preserve: existing refresh claim preservation, API-token conversion, and enrichment capture policy, referencing the authorization architecture for effective role matching

## Construction

- [x] update: [sys/it/impl_childworkspace_test.go](../../../../../pkg/sys/it/impl_childworkspace_test.go)
  - refine: the existing TestRoleScopedToOneForeignWorkspace using the shared VIT application, a signed-in foreign User, and three initialized child Workspaces with one selected target
  - fix: baseline protected-query requests to explicitly use the foreign User's original token and verify HTTP 403 for the target, siblings, and owning Workspace
  - verify: trusted application-bound issuance preserves the foreign User's identity, carries one WorkspaceOwner grant with RoleScopeWorkspace for the target, and leaves IsAPIToken false without Subject membership or global roles
  - verify: the issued token receives HTTP 200 for the same protected query in the target and HTTP 403 in both siblings and the owning Workspace, using explicit bearer tokens for every authorization assertion
  - reuse: existing token validation and Workspace lifecycle helpers, keeping one root assertion instance and brief comments explaining the authorization boundaries without adding new mocks or helper abstractions

- [x] update: [pkg/iauthnzimpl/impl_test.go](../../../../../pkg/iauthnzimpl/impl_test.go)
  - retain: the existing TestAuthenticate cases proving regular-token exact grants apply in the target but not siblings, the parent, or descendants, and mixed grants retain owned-by access in siblings
  - verify: assertions compare principals produced by Authenticate, including retained identity and Host principals, while distinguishing pre-existing ownership from roles granted by the token
  - preserve: existing API-token restrictions, descriptor requirements, System short-circuits, and owned-by regression cases without duplicating their setup or assertions

- [x] update: [pkg/iauthnzimpl/impl.go](../../../../../pkg/iauthnzimpl/impl.go)
  - retain: the existing regular-token scope switch matching RoleScopeWorkspace against RequestWSID and RoleScopeWorkspacesOwnedBy against the immediate OwnerWSID
  - preserve: the existing descriptor guard, processing order, principal deduplication, identity and role sources, and separate API-token composition path

- [x] verify: scoped implementation and regression checks
  - run: `gofmt -w pkg/iauthnzimpl/impl.go pkg/iauthnzimpl/impl_test.go pkg/sys/it/impl_childworkspace_test.go`
  - run: `go test ./pkg/iauthnzimpl ./pkg/itokens-payloads ./pkg/sys/authnz`
  - run: `go test ./pkg/sys/it -run '^(TestRoleScopedToOneForeignWorkspace|TestForeignAuthorization)$' -count=1`
  - run: `golangci-lint run ./pkg/iauthnzimpl/... ./pkg/itokens-payloads/... ./pkg/sys/authnz/... ./pkg/sys/it/...`
  - run: `git diff --check`
  - review: the cumulative implementation for duplicated setup, scenario overlap, readable naming, and assertions that would fail if role matching were removed or broadened
  - result: scoped unit tests and both focused VIT tests pass; changed-line lint reports zero issues, while full scoped lint reports the existing revive add-constant finding in unchanged pkg/sys/it/test_utils.go:131
  - result: working-tree and changed-code whitespace checks pass; the staged verbatim Jira snapshot retains trailing whitespace and a blank line at EOF from the issue body
