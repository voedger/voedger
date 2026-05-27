---
change_id: 2605271401-derive-auth-specs
type: docs
---
# Change request: Derive authn specifications

## Why

The authentication area needs source-derived architecture, feature, and technical design specifications so reviewers and implementers can reason from documented behavior instead of scattered code knowledge. Capturing these artifacts will make future authn changes easier to evaluate for correctness, security impact, and integration boundaries.

## What

This documentation change gives readers an authn-focused specification set derived from the current codebase:

- Authn architecture is described in uspecs terms, including responsibilities, boundaries, and interactions with related domains.
- User and device login creation behavior is captured, including registry login creation, generated device credentials, and profile workspace readiness.
- Principal token issuance, refresh behavior, TTL behavior, and authn identity payload fields are documented as existing authn contracts.
- Token format is documented as a cross-cutting architecture dependency referenced by authn, not as an authn-owned feature.
- Authn feature behavior is captured so externally observable flows and constraints can be reviewed without reading implementation code first.
- Authn technical designs document the current mechanisms, contracts, and integration points needed to maintain or extend the area.

## How

Decisions:

- Derive the authn specification set from current source behavior and existing trace links instead of introducing new runtime behavior.
- Place auth artifacts under the existing production auth specification area so architecture, feature behavior, and technical designs are reviewed together.
- Treat user login creation, device login creation, authentication contracts, authn identity payload fields, principal token issuance, token refresh, TTL behavior, token format as a cross-cutting architecture dependency, and profile workspace readiness as the initial source scope for derivation.

Out of scope:

- Changing authentication behavior, token payload/format contracts, or API responses.
- Authorization behavior, including role evaluation, workspace/device authorization, ACL decisions, global/enriched role semantics, and invite authorization flows.
- Designing new auth flows such as MFA, external identity providers, or session management.
- Refactoring implementation code while deriving the specifications.

References:

- [authentication contract](../../../../../pkg/iauthnz/authn-interface.go)
- [authentication request and principal types](../../../../../pkg/iauthnz/authn-types.go)
- [principal token payload types](../../../../../pkg/itokens-payloads/types.go)
- [user and device API handlers](../../../../../pkg/router/impl_apiv2.go)
- [registry login creation command](../../../../../pkg/registry/impl_createlogin.go)
- [principal token issuance query](../../../../../pkg/registry/impl_issueprincipaltoken.go)
- [login token issuance handler](../../../../../pkg/processors/query2/impl_auth_login_handler.go)
- [token refresh handler](../../../../../pkg/processors/query2/impl_auth_refresh_handler.go)
- [signup and device integration coverage](../../../../../pkg/sys/it/impl_signupin_test.go)

## Domain specifications

- [x] update: [prod/domain.md](../../../../specs/prod/domain.md)
  - add: authn concepts for Subject, User, Device, Login, Credential, Principal, Principal Token, Verified Value Token, and Profile Workspace
  - group: concepts into Platform and Authentication subsections

## Functional design

- [x] create: [auth/authn.feature](../../../../specs/prod/auth/authn.feature)
  - Feature specification with scenarios for user login creation, device login creation, profile workspace readiness, sign-in, existing authn token response and identity payload contracts, token issue, token refresh, and documented error outcomes

## Technical design

- [x] create: [auth/arch.md](../../../../specs/prod/auth/arch.md)
  - Context Architecture: authn responsibilities, boundaries, token dependency, registry/profile workspace interactions, and relationship to authorization as an out-of-scope downstream consumer

- [x] create: [auth/authn--td.md](../../../../specs/prod/auth/authn--td.md)
  - Feature Technical Design: user and device login creation, verified value token usage, profile workspace readiness, principal token issue/refresh, TTL behavior, authn identity payload fields, and error mapping
