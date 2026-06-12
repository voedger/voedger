---
registered_at: 2026-04-24T08:20:45Z
change_id: 2604240820-fix-stuck-invite-states
baseline: f91d1438f6fbf47ffd247bb8d06e47cfdff55e6e
archived_at: 2026-04-24T08:54:40Z
---

# Change request: Fix stuck invite states recovery

## Why

Invitations can get stuck in transitional states (`State_ToBeInvited`, `State_ToBeJoined`) when async projectors fail. Previously, the `ApplyJoinWorkspace` projector had an early-return bug that skipped updating invite state when an inactive Subject existed, leaving invites permanently stuck in `State_ToBeJoined`. Even after fixing the projector (change 2604221416), already-stuck invites cannot be recovered because commands `InitiateInvitationByEMail` and `CancelSentInvite` don't accept these transitional states.

## What

Allow recovery from stuck transitional invite states:

- Add `State_ToBeInvited` and `State_ToBeJoined` to valid states for `InitiateInvitationByEMail`
- Add `State_ToBeInvited` and `State_ToBeJoined` to valid states for `CancelSentInvite`
- Add `State_ToBeJoined` to `reInviteAllowedForState`

## How

Update `inviteValidStates` and `reInviteAllowedForState` maps in `pkg/sys/invite/consts.go`.

Add integration tests:

- Test re-invite from `State_ToBeInvited` (email send failed scenario)
- Test re-invite from `State_ToBeJoined` (join projector failed scenario)
- Test cancel from `State_ToBeInvited`
- Test cancel from `State_ToBeJoined`

References:

- [pkg/sys/invite/consts.go](../../../../../pkg/sys/invite/consts.go)
- [pkg/sys/it/impl_invite_test.go](../../../../../pkg/sys/it/impl_invite_test.go)
