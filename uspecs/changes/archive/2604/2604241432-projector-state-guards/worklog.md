# Worklog

## 2026-04-24

### 2026-04-24 17:27 segregation of concerns

Problem: commands and projectors both write Invite state -> races, TOCTOU, guards,
validated commands -- all patches on a fundamentally flawed design.

Solution: single projector is the sole writer of Invite State. Commands validate
and may write data fields (but never State). Projector processes events in PLog
order -- serialized, no races.

Commands do pre-validation (UX -- immediate 400 for invalid requests).
Projector re-validates (source of truth -- sees actual state after all prior events).

```text
Command:  pre-validate state -> create event (no CUD on Invite)
Projector: process event -> re-check actual state -> apply or skip
```

#### Constraints

Some commands must still write to the Invite CDoc because they carry data that
is only available at command execution time (from auth token, user input):

- `InitiateInvitationByEMail`: creates the Invite CDoc (new record) or updates
  Roles, ExpireDatetime on re-invite. Command must do this -- projector can't
  create CDocs
- `InitiateJoinWorkspace`: writes InviteeProfileWSID, SubjectKind, ActualLogin
  from the auth token (RequestSubject). Projector only sees the event, not the
  token

These commands still write their data fields but do NOT write State. The
projector is the sole writer of State.

Commands that only validate + set ToBe state today (no extra data fields):

- `InitiateUpdateInviteRoles`: only sets State=ToUpdateRoles. Remove CUD
- `InitiateCancelAcceptedInvite`: only sets State=ToBeCancelled. Remove CUD
- `InitiateLeaveWorkspace`: sets State=ToBeLeft and IsActive=false. Remove CUD.
  IsActive=false moves to projector
- `CancelSentInvite`: sets State=Cancelled. Remove CUD

#### Compatibility

- State enum constants keep their numeric values (iota-based). ToBe states stay
  in the enum as dead values -- projector never writes them, but old data with
  these values remains readable
- Projector must handle old ToBe states from pre-migration data. When reading
  invite state, treat ToBe states as equivalent to their source state:
  - ToBeInvited -> same as no email sent yet, apply invitation
  - ToBeJoined -> same as Invited, apply join
  - ToUpdateRoles -> same as Joined, apply role update
  - ToBeCancelled -> same as Joined, apply cancel
  - ToBeLeft -> same as Joined, apply leave
- `CancelSentInvite` keeps its API name (renaming to `InitiateCancelSentInvite`
  would break clients). Implementation changes: remove CUD, keep pre-validation
- Command pre-validation valid states must include ToBe states for recovery
  from old data (same as today)
- No command returns data -- clients only get 200/400. State observation requires
  a separate query, so eventual consistency is transparent

#### What goes away

- `c.sys.CompleteInvitation` and `c.sys.CompleteJoinWorkspace` commands
- Projector guards and TOCTOU concerns
- 5 separate projectors -> 1 projector (`ap.sys.ApplyInviteEvents`)
- Commands no longer write State to Invite CDoc

#### What stays

- State enum constants (all values preserved, ToBe states unused by new code)
- Federation calls to invitee's profile workspace (different WSID):
  - `c.sys.CreateJoinedWorkspace`
  - `c.sys.UpdateJoinedWorkspaceRoles`
  - `c.sys.DeactivateJoinedWorkspace`
- Command pre-validation (reject invalid state transitions synchronously)
- All API endpoint names unchanged
- `InitiateInvitationByEMail` still creates/updates Invite CDoc (data fields only, not State)
- `InitiateJoinWorkspace` still writes InviteeProfileWSID, SubjectKind, ActualLogin (data fields only, not State)

#### State diagram

New projector writes these states: Invited, Joined, Cancelled, Left.
ToBe states are dead (only in old data).

```text
[*] -> Invited:   InitiateInvitationByEMail (projector sends email, sets State=Invited)
Invited -> Invited:   InitiateInvitationByEMail (re-invite, projector sends new email)
Invited -> Joined:    InitiateJoinWorkspace (projector creates Subject, JoinedWorkspace)
Invited -> Cancelled: CancelSentInvite (projector sets State=Cancelled)
Joined -> Cancelled:  InitiateCancelAcceptedInvite (projector deactivates Subject, JoinedWorkspace)
Joined -> Left:       InitiateLeaveWorkspace (projector deactivates Subject, JoinedWorkspace, sets IsActive=false)
Joined -> Joined:     InitiateUpdateInviteRoles (projector updates Subject, JoinedWorkspace, sends email)
Cancelled -> Invited: InitiateInvitationByEMail (re-invite)
Left -> Invited:      InitiateInvitationByEMail (re-invite)
```

#### Plan

- [ ] update: invites--td.md
  - replace: state diagram (ToBe states marked as legacy)
  - replace: projector guards section with single-projector design
  - remove: CompleteInvitation, CompleteJoinWorkspace from commands list
  - update: Decisions section

- [ ] update: sys.vsql
  - remove: CompleteInvitationParams, CompleteJoinWorkspaceParams types
  - remove: CompleteInvitation, CompleteJoinWorkspace commands
  - add: ApplyInviteEvents projector declaration

- review  

- [ ] update: consts.go
  - keep: all State_ constants (preserve numeric values)
  - remove: QNames for CompleteInvitation, CompleteJoinWorkspace
  - remove: 5 separate projector QNames -> 1: `qNameAPApplyInviteEvents`
  - remove: test hooks (OnBeforeApply*, OnAfterGuard*)
  - update: inviteValidStates (keep ToBe states for old data recovery)

- [ ] add: `ap.sys.ApplyInviteEvents` projector implementation
  - VSQL: `PROJECTOR ApplyInviteEvents AFTER EXECUTE ON (InitiateInvitationByEMail, InitiateJoinWorkspace, InitiateUpdateInviteRoles, InitiateCancelAcceptedInvite, InitiateLeaveWorkspace, CancelSentInvite)`
  - event handler: switch on event command QName
  - each handler: re-validate actual state (including old ToBe states), apply transition + side effects, or skip

- [ ] add: validation helper `validateInviteCmd(state, args, cmdQName) error`
  - load invite by ID, check exists, check state via inviteValidStates
  - used by: InitiateUpdateInviteRoles, InitiateCancelAcceptedInvite, InitiateLeaveWorkspace, CancelSentInvite
  - InitiateInvitationByEMail and InitiateJoinWorkspace have extra logic, use helper partially

- [ ] update: commands (remove State CUD, keep pre-validation and data fields)
  - update: InitiateInvitationByEMail -- keep Invite CDoc create/update (data fields), remove State write
  - update: InitiateJoinWorkspace -- keep InviteeProfileWSID/SubjectKind/ActualLogin writes, remove State write
  - update: InitiateUpdateInviteRoles -- remove all CUD on Invite
  - update: InitiateCancelAcceptedInvite -- remove all CUD on Invite
  - update: InitiateLeaveWorkspace -- remove all CUD on Invite (IsActive=false moves to projector)
  - update: CancelSentInvite -- remove all CUD on Invite

- [ ] remove: impl_completeinvitation.go
- [ ] remove: impl_completejoinworkspace.go
- [ ] remove: 5 separate projector files -> 1 file

- [ ] update: provide.go
  - register ApplyInviteEvents projector
  - remove old projector registrations
  - remove CompleteInvitation, CompleteJoinWorkspace registrations

- [ ] update: tests
  - remove: TestCompleteInvitation, TestCompleteJoinWorkspace
  - remove: TestProjectorStateGuards (guards no longer exist)
  - update: TestInvite_BasicUsage (states go directly to Invited/Joined, no ToBe)
  - add: test projector idempotency (stale events skipped)
  - add: test old ToBe state handling (migration compatibility)
