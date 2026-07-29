---
change_id: 2607291112-accept-alias-invitations
type: fix
issue_url: https://untill.atlassian.net/browse/AIR-4612
domains: [prod]
scope: [auth]
---

# Change request: Workspace invitations addressed to login aliases

Refs:

- [AIR-4612: Voedger: accept workspace invitations addressed to login aliases](./issue-AIR-4612.md)

## Why

Authenticated users must be able to accept workspace invitations sent to their active `Login Alias` without weakening canonical `Login` identity or membership integrity. The current validation rejects such invitations even though the `Principal Token` carries both identifiers.

## What

Symptom: An authenticated user receives HTTP 400 when accepting a workspace invitation addressed to the user's active login alias.

```text
@Invitee submits the verification code for an alias-addressed invitation
      |
      v
[c.sys.InitiateJoinWorkspace] loads [(cdoc.sys.Invite)] and the request subject
      |
      v
execCmdInitiateJoinWorkspace compares Invite.Email only with RequestSubject.Name   <-- fault: ignores the authenticated token Alias
      |
      v
[c.sys.InitiateJoinWorkspace] returns HTTP 400   (symptom)
```

Corrected behavior: `InitiateJoinWorkspace` accepts an invitation when `Invite.Email` matches the authenticated canonical `Login` or token `Alias`, keeps `ActualLogin` and `Subject.Login` canonical, avoids duplicate `Subject` records for existing members, and rejects invitations for other identities.

Because workspace invitations previously had no cohesive functional specification, this change also derives the existing user-facing invitation lifecycle and aligns its integration tests with that contract. Projector recovery, replay versioning, and other implementation-only behavior remain outside the functional feature.

## How

Decisions:

- Decode the authenticated principal token at the join-command boundary through the invite subsystem's existing token-service dependency instead of extending the generic request-subject storage or principal model.
- Use the `PrincipalPayload.Alias` snapshot carried by the request token without re-resolving the alias in the registry, preserving the platform's token snapshot semantics.
- Limit the alias to synchronous invitation-recipient matching; do not persist it into the invite or propagate it into the asynchronous membership projection.
- Verify the change through the existing invite integration boundary with a real alias-bearing principal token so command validation and canonical asynchronous membership behavior are exercised together.
- Derive the invitation feature from public commands, authorization rules, technical design, and externally observable integration behavior without promoting projector or persistence mechanics into the functional contract.

Assumptions:

- None

References:

- [invite subsystem dependency wiring](../../../../../pkg/sys/invite/provide.go)
- [existing request-token decoding pattern](../../../../../pkg/sys/authnz/impl_refreshprincipaltoken.go)
- [canonical login and alias payload contract](../../../../../pkg/itokens-payloads/types.go)
- [principal token snapshot semantics](../../../../../uspecs/specs/prod/auth/arch-tokens.md)
- [canonical membership projection and subject reuse](../../../../../pkg/sys/invite/impl_applyinviteevents.go)
- [invite integration boundary](../../../../../pkg/sys/it/impl_invite_test.go)

## Functional design

- [x] create: [auth/invites.feature](../../../../specs/prod/auth/invites.feature)
  - Feature specification for the user-facing workspace invitation lifecycle, including canonical-login and active-alias recipients

## Construction

- [x] create: [sys/it/impl_invites_feature_test.go](../../../../../pkg/sys/it/impl_invites_feature_test.go)
  - Feature-traceable integration suite for [auth/invites.feature](../../../../specs/prod/auth/invites.feature)
  - `TestInvites` scenario subtests with exact Scenario Outline rows and verbatim step comments
  - Coverage for sending, resending, accepting, role updates, membership ending and restoration, recipient aliases, unusable invitations, and input validation

- [x] update: [sys/it/impl_invite_test.go](../../../../../pkg/sys/it/impl_invite_test.go)
  - move: user-facing invitation lifecycle coverage to `impl_invites_feature_test.go`
  - retain: projector recovery, version-marker, inactive-subject authorization, and other implementation-only tests without feature scenario markers

- [x] update: [sys/invite/provide.go](../../../../../pkg/sys/invite/provide.go)
  - pass the existing token-service dependency to the `InitiateJoinWorkspace` command registration

- [x] update: [sys/invite/impl_initiatejoinworkspace.go](../../../../../pkg/sys/invite/impl_initiatejoinworkspace.go)
  - decode the authenticated request principal token and read its canonical `Login` and snapshotted `Alias`
  - accept the invitation recipient when it matches either authenticated identifier and preserve rejection for every other identity
  - continue writing only the canonical login to `ActualLogin` for asynchronous membership processing
