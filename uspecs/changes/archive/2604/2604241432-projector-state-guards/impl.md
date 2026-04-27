# Implementation plan: Refactor invite flow - single projector and roles validation

## Functional design

- [x] update: [prod--domain.md](../../../../specs/prod/prod--domain.md)
  - update: Expand auth context description to include workspace membership (invites, subjects, role assignments)

## Technical design

- [x] update: [invites--td.md](../../../../specs/prod/auth/invites--td.md)
  - replace: state diagram - mark legacy `ToBe*` states, single-projector transitions
  - replace: projector guards section with single-projector design
  - remove: CompleteInvitation, CompleteJoinWorkspace from commands list
  - update: Decisions section

## Construction

### Segregation of concerns

- [x] update: [sys.vsql](../../../../../pkg/sys/sys.vsql)
  - remove: CompleteInvitationParams, CompleteJoinWorkspaceParams types
  - remove: CompleteInvitation, CompleteJoinWorkspace commands
  - add: ApplyInviteEvents projector declaration `AFTER EXECUTE ON (InitiateInvitationByEMail, InitiateJoinWorkspace, InitiateUpdateInviteRoles, InitiateCancelAcceptedInvite, InitiateLeaveWorkspace, CancelSentInvite)`
- [x] update: [consts.go](../../../../../pkg/sys/invite/consts.go)
  - keep: all `State_*` constants (numeric values preserved)
  - remove: QNames for CompleteInvitation, CompleteJoinWorkspace
  - remove: 5 separate projector QNames -> 1 (`qNameAPApplyInviteEvents`)
  - remove: test hooks (`OnBeforeApply*`, `OnAfterGuard*`)
  - update: `inviteValidStates` (keep `ToBe*` for old data recovery)
- [x] add: [impl_applyinviteevents.go](../../../../../pkg/sys/invite/impl_applyinviteevents.go)
  - add: `ap.sys.ApplyInviteEvents` projector handling all invite events
  - add: per-command handlers re-validate actual state and apply transition + side effects, skip stale events
- [x] remove: pkg/sys/invite/impl_applyinvitation.go
- [x] remove: pkg/sys/invite/impl_applyjoinworkspace.go
- [x] remove: pkg/sys/invite/impl_applyupdateinviteroles.go
- [x] remove: pkg/sys/invite/impl_applycancelacceptedinvite.go
- [x] remove: pkg/sys/invite/impl_applyleaveworkspace.go
- [x] remove: pkg/sys/invite/impl_completeinvitation.go
- [x] remove: pkg/sys/invite/impl_completejoinworkspace.go
- [x] update: [provide.go](../../../../../pkg/sys/invite/provide.go)
  - register: ApplyInviteEvents projector
  - remove: old projector and Complete* command registrations
- [x] update: [impl_initiateinvitationbyemail.go](../../../../../pkg/sys/invite/impl_initiateinvitationbyemail.go)
  - keep: Invite CDoc create/update with data fields and `State=ToBeInvited`
- [x] update: [impl_initiatejoinworkspace.go](../../../../../pkg/sys/invite/impl_initiatejoinworkspace.go)
  - keep: InviteeProfileWSID/SubjectKind/ActualLogin writes
  - remove: State write
- [x] update: [impl_initiateupdateinviteroles.go](../../../../../pkg/sys/invite/impl_initiateupdateinviteroles.go)
  - remove: all CUD on Invite (projector writes Roles)
- [x] update: [impl_initiatecancelacceptedinvite.go](../../../../../pkg/sys/invite/impl_initiatecancelacceptedinvite.go)
  - remove: all CUD on Invite
- [x] update: [impl_initiateleaveworkspace.go](../../../../../pkg/sys/invite/impl_initiateleaveworkspace.go)
  - keep: no-op CUD (carries InviteID to projector via event.CUDs)
  - remove: State/IsActive/Updated writes
- [x] update: [impl_cancelsentinvite.go](../../../../../pkg/sys/invite/impl_cancelsentinvite.go)
  - remove: all CUD on Invite

### Roles validation

- [x] update: [iauthnz/utils.go](../../../../../pkg/iauthnz/utils.go)
  - update: `IsSystemRole` uses package prefix check (`role.Pkg() == appdef.SysPackage`)
  - remove: `slices` import and `SysRoles` usage from `IsSystemRole`
- [x] update: [iauthnzimpl/impl_test.go](../../../../../pkg/iauthnzimpl/impl_test.go)
  - update: IssueAPIToken test allowed case uses non-sys QName (e.g. `appdef.NewQName("test", "role")`)
- [x] add: [invite/utils.go](../../../../../pkg/sys/invite/utils.go)
  - add: `validateInviteRoles(rolesStr string, ws appdef.IWorkspace) error`
  - logic: split by comma, trim, parse QName, reject `sys.*`, reject roles not declared in workspace
  - return: `coreutils.NewHTTPError(http.StatusBadRequest, ...)`
- [x] add: [invite/utils_test.go](../../../../../pkg/sys/invite/utils_test.go)
  - add: extensive unit tests for `validateInviteRoles`
- [x] update: [impl_initiateinvitationbyemail.go](../../../../../pkg/sys/invite/impl_initiateinvitationbyemail.go)
  - add: call `validateInviteRoles` early
- [x] update: [impl_initiateupdateinviteroles.go](../../../../../pkg/sys/invite/impl_initiateupdateinviteroles.go)
  - add: call `validateInviteRoles` early

### Tests

- [x] update: [impl_invite_test.go](../../../../../pkg/sys/it/impl_invite_test.go)
  - remove: `TestCompleteInvitation`, `TestCompleteJoinWorkspace`
  - remove: `TestProjectorStateGuards`
  - update: `TestInvite_BasicUsage` (states go directly to Invited/Joined, no `ToBe*`)
  - add: projector idempotency test (stale events skipped)
  - add: old `ToBe*` state handling test (migration compatibility)
  - add: `TestInvite_RolesValidation` cases:
    - malformed QName -> HTTP 400
    - `sys.WorkspaceOwner` -> HTTP 400
    - non-existent role (`app1pkg.NonExistentRole`) -> HTTP 400
    - whitespace-only / leading comma / duplicate role -> HTTP 400
- [x] update: [schemaTestApp1.vsql](../../../../../pkg/vit/schemaTestApp1.vsql)
  - rename: `app1pkg.WorkspaceSubject` -> `app1pkg.InviteTestRole`
- [x] update: [impl_deactivateworkspace_test.go](../../../../../pkg/sys/it/impl_deactivateworkspace_test.go)
  - update: references to renamed test role

- [x] Review
