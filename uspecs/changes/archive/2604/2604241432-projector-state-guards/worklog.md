# Worklog

## 2026-04-24

### 2026-04-24 17:27 segregation of concerns

Design is now in [invites--td.md](../../../../specs/prod/auth/invites--td.md).

#### What goes away

- `c.sys.CompleteInvitation` and `c.sys.CompleteJoinWorkspace` commands
- `CompleteInvitationParams`, `CompleteJoinWorkspaceParams` types in sys.vsql
- Projector guards and TOCTOU concerns
- 5 separate projectors -> 1 projector (`ap.sys.ApplyInviteEvents`)
- Test hooks: OnBeforeApply*, OnAfterGuard*

#### What stays

- State enum constants (all numeric values preserved, ToBeInvited still active)
- Federation calls to invitee's profile workspace:
  `CreateJoinedWorkspace`, `UpdateJoinedWorkspaceRoles`, `DeactivateJoinedWorkspace`
- Command pre-validation (reject invalid state transitions synchronously)
- All API endpoint names unchanged

#### Command CUD changes

- `InitiateInvitationByEMail`: keep create/update (data fields + State=ToBeInvited)
- `InitiateJoinWorkspace`: keep InviteeProfileWSID/SubjectKind/ActualLogin, remove State write
- `InitiateUpdateInviteRoles`: remove all CUD. Projector writes Roles to Invite CDoc
- `InitiateCancelAcceptedInvite`: remove all CUD
- `InitiateLeaveWorkspace`: keep no-op CUD (no InviteID param, no auth token in
  projector -- CUD is the only way to pass InviteID to projector via event.CUDs)
- `CancelSentInvite`: remove all CUD

#### Compatibility

- ToBe states keep numeric values (iota). ToBeInvited still written by command.
  Other ToBe states become dead values -- only in old data
- Command pre-validation valid states must include ToBe states for old data recovery
- No command returns data -- eventual consistency is transparent

#### Plan

- [x] update: invites--td.md
  - replace: state diagram (ToBe states marked as legacy)
  - replace: projector guards section with single-projector design
  - remove: CompleteInvitation, CompleteJoinWorkspace from commands list
  - update: Decisions section

- [x] update: sys.vsql
  - remove: CompleteInvitationParams, CompleteJoinWorkspaceParams types
  - remove: CompleteInvitation, CompleteJoinWorkspace commands
  - add: ApplyInviteEvents projector declaration

- [x] review

- [x] update: consts.go
  - keep: all State_ constants (preserve numeric values)
  - remove: QNames for CompleteInvitation, CompleteJoinWorkspace
  - remove: 5 separate projector QNames -> 1: `qNameAPApplyInviteEvents`
  - remove: test hooks (OnBeforeApply*, OnAfterGuard*)
  - update: inviteValidStates (keep ToBe states for old data recovery)

- [x] add: `ap.sys.ApplyInviteEvents` projector implementation
  - VSQL: `PROJECTOR ApplyInviteEvents AFTER EXECUTE ON (InitiateInvitationByEMail, InitiateJoinWorkspace, InitiateUpdateInviteRoles, InitiateCancelAcceptedInvite, InitiateLeaveWorkspace, CancelSentInvite)`
  - event handler: switch on event command QName
  - each handler: re-validate actual state (including old ToBe states), apply transition + side effects, or skip

- [-] add: validation helper `validateInviteCmd(state, args, cmdQName) error`
  - not needed: commands already share `isValidInviteState`; remaining load+check is ~10 lines with unique additional validation per command (KISS)

- [x] update: commands (remove final State CUD, keep pre-validation and data fields)
  - update: InitiateInvitationByEMail -- keep Invite CDoc create/update (data fields + State=ToBeInvited)
  - update: InitiateJoinWorkspace -- keep InviteeProfileWSID/SubjectKind/ActualLogin writes, remove State write
  - update: InitiateUpdateInviteRoles -- remove all CUD on Invite
  - update: InitiateCancelAcceptedInvite -- remove all CUD on Invite
  - update: InitiateLeaveWorkspace -- keep no-op CUD (passes InviteID to projector), remove State/IsActive/Updated writes
  - update: CancelSentInvite -- remove all CUD on Invite

- [x] remove: impl_completeinvitation.go
- [x] remove: impl_completejoinworkspace.go
- [x] remove: 5 separate projector files -> 1 file

- [x] update: provide.go
  - register ApplyInviteEvents projector
  - remove old projector registrations
  - remove CompleteInvitation, CompleteJoinWorkspace registrations

- [x] update: tests
  - remove: TestCompleteInvitation, TestCompleteJoinWorkspace
  - remove: TestProjectorStateGuards (guards no longer exist)
  - update: TestInvite_BasicUsage (states go directly to Invited/Joined, no ToBe)
  - add: test projector idempotency (stale events skipped)
  - add: test old ToBe state handling (migration compatibility)

### 2026-04-24 19:11 review

- Why this file exists https://github.com/voedger/voedger/blob/main/pkg/sys/it/testdata/apps/test2.app1/image/pkg/sys/sys.vsql?
- Can it differ from the main sys.vsql? If not, should we remove it to avoid confusion?
- //TODO Denis how to get WS by login? I want to check sys.JoinedWorkspace
