# Implementation plan: Fix stuck invite states recovery

## Construction

- [x] update: [pkg/sys/invite/consts.go](../../../../../pkg/sys/invite/consts.go)
  - add: `State_ToBeInvited` and `State_ToBeJoined` to `inviteValidStates[InitiateInvitationByEMail]`
  - add: `State_ToBeInvited` and `State_ToBeJoined` to `inviteValidStates[CancelSentInvite]`
  - add: `State_ToBeJoined` to `reInviteAllowedForState`
- [x] update: [pkg/sys/it/impl_invite_test.go](../../../../../pkg/sys/it/impl_invite_test.go)
  - add: `TestRecoverFromStuckInviteStates` with subtests for re-invite and cancel from stuck states
  - add: `setInviteState` helper function for testing
