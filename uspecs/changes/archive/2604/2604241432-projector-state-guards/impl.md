# Implementation plan: Add state guards to invite projectors

## Functional design

- [x] update: [prod--domain.md](../../../../specs/prod/prod--domain.md)
  - update: Expand auth context description to include workspace membership (invites, subjects, role assignments)

## Technical design

- [x] update: [invites--td.md](../../../../specs/prod/auth/invites--td.md)
  - update: Extra state diagram - add recovery transitions from ToBeInvited and ToBeJoined (PR 4511)
  - add: Projector guards subsection - describe state check behavior (skip if state changed)
  - add: New commands for state transitions (CompleteInvitation, CompleteJoinWorkspace)

## Construction

- [x] Review

### Layer 1: Projector state guards

Prevents projector from doing any work when the invite state has already changed.
Covers the case when recovery action completes before projector starts.

- [x] update: [impl_applyinvitation.go](../../../../../pkg/sys/invite/impl_applyinvitation.go)
  - add: Check `State == ToBeInvited` after loading invite, return nil if not
- [x] update: [impl_applyjoinworkspace.go](../../../../../pkg/sys/invite/impl_applyjoinworkspace.go)
  - add: Check `State == ToBeJoined` after loading invite, return nil if not

### Layer 2: Command-based state transitions

Layer 1 alone has a TOCTOU gap: state can change between the guard check and the
CUD that updates state. Layer 2 replaces direct CUD with dedicated commands that
validate the expected state at execution time.

If command returns error (state changed between guard and command), projector fails
and is reapplied. On reapply, Layer 1 guard sees the actual state and skips.

- [x] add: [impl_completeinvitation.go](../../../../../pkg/sys/invite/impl_completeinvitation.go)
  - `c.sys.CompleteInvitation` command (ToBeInvited -> Invited)
  - validate: `State == ToBeInvited`, error otherwise
  - set: State, VerificationCode, Updated
- [x] add: [impl_completejoinworkspace.go](../../../../../pkg/sys/invite/impl_completejoinworkspace.go)
  - `c.sys.CompleteJoinWorkspace` command (ToBeJoined -> Joined)
  - validate: `State == ToBeJoined`, error otherwise
  - set: State, SubjectID, Updated
- [x] add: VSQL declarations in [sys.vsql](../../../../../pkg/sys/sys.vsql)
  - add: `CompleteInvitationParams` type
  - add: `CompleteJoinWorkspaceParams` type
  - add: `COMMAND CompleteInvitation` declaration
  - add: `COMMAND CompleteJoinWorkspace` declaration
- [x] update: [consts.go](../../../../../pkg/sys/invite/consts.go)
  - add: QName vars for new commands
- [x] update: [provide.go](../../../../../pkg/sys/invite/provide.go)
  - add: register new commands
- [x] update: [impl_applyinvitation.go](../../../../../pkg/sys/invite/impl_applyinvitation.go)
  - replace: `federation.Func("c.sys.CUD")` with `federation.Func("c.sys.CompleteInvitation")`
- [x] update: [impl_applyjoinworkspace.go](../../../../../pkg/sys/invite/impl_applyjoinworkspace.go)
  - replace: direct CUD state update with `federation.Func("c.sys.CompleteJoinWorkspace")`

### Test hooks and tests

Two hooks per projector: before guard (Layer 1 tests) and after guard (Layer 2 tests).
Hooks are nil in production. Tests set hook, block via channel, change state, unblock.
Layer 2 tests use `reached` channel to ensure projector is blocked before state change.

- [x] add: global vars in [consts.go](../../../../../pkg/sys/invite/consts.go)
  - `OnBeforeApplyInvitation func()` - before invite load and guard
  - `OnAfterGuardApplyInvitation func()` - after guard passes, before side effects
  - `OnBeforeApplyJoinWorkspace func()` - before invite load and guard
  - `OnAfterGuardApplyJoinWorkspace func()` - after guard passes, before side effects
- [x] update: [impl_applyinvitation.go](../../../../../pkg/sys/invite/impl_applyinvitation.go)
  - add: call hooks at appropriate positions
- [x] update: [impl_applyjoinworkspace.go](../../../../../pkg/sys/invite/impl_applyjoinworkspace.go)
  - add: call hooks at appropriate positions
- [x] update: [impl_invite_test.go](../../../../../pkg/sys/it/impl_invite_test.go)
  - add: Layer 1 - block before guard, cancel from ToBeInvited, verify state stays Cancelled
  - add: Layer 1 - block before guard, re-invite from ToBeJoined, verify state stays Invited
- [x] add: TOCTOU - block ApplyInvitation after guard, cancel from ToBeInvited, verify state stays Cancelled
- [x] add: TOCTOU - block ApplyJoinWorkspace after guard, cancel from ToBeJoined, verify state stays Cancelled
- [x] run all TestProjectorStateGuards tests

### Dedicated command tests

- [x] add: `TestCompleteInvitation` in [impl_invite_test.go](../../../../../pkg/sys/it/impl_invite_test.go)
  - invite not exists -> 400 (referential integrity)
  - wrong state (Invited) -> 409
- [x] add: `TestCompleteJoinWorkspace` in [impl_invite_test.go](../../../../../pkg/sys/it/impl_invite_test.go)
  - invite not exists -> 400 (referential integrity)
  - wrong state (Invited) -> 409
