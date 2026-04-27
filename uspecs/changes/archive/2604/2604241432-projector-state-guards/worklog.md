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

### 2026-04-27 12:10 validate roles

- Currently roles are not validated

#### sys.JoinedWorkspace scheme

- TABLE JoinedWorkspace INHERITS sys.CDoc
  - Roles varchar(1024) NOT NULL
  - InvitingWorkspaceWSID int64 NOT NULL
  - WSName varchar NOT NULL

- Related types:
  - CreateJoinedWorkspaceParams (Roles text, InvitingWorkspaceWSID int64, WSName text)
  - UpdateJoinedWorkspaceRolesParams (Roles text, InvitingWorkspaceWSID int64)
  - DeactivateJoinedWorkspaceParams (InvitingWorkspaceWSID int64)

- Related view:
  - JoinedWorkspaceIndexView (Dummy int32, InvitingWorkspaceWSID int64, JoinedWorkspaceID ref) PK((Dummy), InvitingWorkspaceWSID)

- Commands: CreateJoinedWorkspace, UpdateJoinedWorkspaceRoles, DeactivateJoinedWorkspace, OnJoinedWorkspaceDeactivated
- Projector: SYNC ProjectorJoinedWorkspaceIndex AFTER EXECUTE ON (CreateJoinedWorkspace)

#### Field_Roles validation status

- Field_Roles is never validated anywhere in the invite flow
- It is read via AsString and stored directly into CDoc records (Invite, JoinedWorkspace, Subject)
- No check that roles string contains valid/existing role QNames
- No check that caller is authorized to grant specific roles
- Only email is validated (coreutils.ValidateEMail) in InitiateInvitationByEMail

#### ACL for granting roles

- No ACL or authorization check validates which roles can be granted
- Invite commands are tagged with WorkspaceOwnerFuncTag -- only workspace owner can execute them
- But any workspace owner can assign any arbitrary string as Roles
- The ACL engine (pkg/appdef/acl) handles operation-level permissions, not role-assignment permissions

#### Roles per workspace

- sys.Workspace (root, all workspaces inherit these):
  - sys.Everyone -- assigned regardless of auth token
  - sys.Anonymous -- assigned if no token
  - sys.AuthenticatedUser -- assigned if valid token
  - sys.System -- everything allowed, ACL skipped
  - sys.ProfileOwner -- user/device works in its own profile
  - sys.WorkspaceDevice -- device MAY work in a workspace owned by its profile
  - sys.RoleWorkspaceOwner -- deprecated, use WorkspaceOwner
  - sys.WorkspaceOwner -- user works in a workspace owned by their profile
  - sys.ClusterAdmin -- not used yet
  - sys.WorkspaceAdmin
  - sys.BLOBUploader

- Role inheritance (GRANTs):
  - WorkspaceOwner -> ProfileOwner
  - WorkspaceOwner -> WorkspaceDevice
  - WorkspaceOwner -> RoleWorkspaceOwner (backward compat)
  - Everyone -> Anonymous
  - Everyone -> AuthenticatedUser
  - BLOBUploader -> WorkspaceOwner

- sys.AppWorkspaceWS: adds ClusterAdmin (via ALTER in pkg/cluster/appws.vsql)
- sys.ProfileWS, sys.UserProfileWS, sys.DeviceProfileWS: no additional roles

- Roles can be listed at runtime via appdef.Roles(workspace.Types()) or filtering workspace.Types() by TypeKind_Role

#### bp3 app-level roles (air package)

- air.BeneficiaryWS (abstract, inherits sys.Workspace):
  - air.UntillPaymentsUser
  - air.UntillPaymentsTerminal
  - air.UntillPaymentsReseller (GRANT sys.WorkspaceAdmin TO UntillPaymentsReseller)
  - air.SubscriptionReseller
  - air.AirReseller -- deprecated, use SubscriptionReseller (GRANT SubscriptionReseller TO AirReseller)

- air.RestaurantWS (inherits untill.unTillWS):
  - air.BOReader
  - air.SimpleRestaurantApi (PUBLISHED)

- air.UntillPaymentsWS (inherits BeneficiaryWS): no additional roles

- air.ResellerWS (inherits BeneficiaryWS): no additional roles

- air.ResellersWS:
  - air.ResellerPortalDashboardViewer
  - air.ResellersAdmin (GRANT sys.WorkspaceAdmin TO ResellersAdmin)

- ALTER sys.UserProfileWS (in air package):
  - air.UntillChargebeeAgent
  - air.UntillPaymentsManager

#### Roles validation plan

- [x] update pkg/iauthnz/utils.go: IsSystemRole to use prefix check (role.Pkg() == appdef.SysPackage) instead of slices.Contains(SysRoles)
  - remove "slices" import, remove SysRoles usage from IsSystemRole
  - SysRoles slice in authn-types.go stays (used by IssueAPIToken test)

- [x] update pkg/iauthnzimpl/impl_test.go: IssueAPIToken test at line 448 uses appdef.NewQName(appdef.SysPackage, "test") as "allowed" case -- change to non-sys QName (e.g. appdef.NewQName("test", "role"))
  - the sys role rejection loop at line 440 stays as-is

- [x] add func ValidateInviteRoles in pkg/sys/invite/utils.go (or new file impl_validateroles.go):
  - signature: func validateInviteRoles(rolesStr string, ws appdef.IWorkspace) error
  - logic:
    - strings.Split(rolesStr, ",") then strings.TrimSpace each
    - appdef.ParseQName(role) -- reject malformed QNames
    - iauthnz.IsSystemRole(qname) -- reject sys.* roles
    - appdef.Role(ws.Type, qname) == nil -- reject roles not found in workspace
  - return coreutils.NewHTTPError(http.StatusBadRequest, ...) with descriptive message

- [x] add a very extensive unit test for validateInviteRoles in pkg/sys/invite/utils_test.go

- [x] rename test role `app1pkg.WorkspaceSubject` -> `app1pkg.InviteTestRole` in pkg/vit/schemaTestApp1.vsql and pkg/sys/it/impl_deactivateworkspace_test.go (clearer name for the non-system role used as substitute for sys.WorkspaceOwner in invite tests)

- [x] update pkg/sys/invite/impl_initiateinvitationbyemail.go: call validateInviteRoles(args.ArgumentObject.AsString(Field_Roles), args.Workspace) early in execCmdInitiateInvitationByEMail
  - args.Workspace is appdef.IWorkspace, set by command processor in buildCommandArgs

- [x] update pkg/sys/invite/impl_initiateupdateinviteroles.go: call validateInviteRoles(args.ArgumentObject.AsString(Field_Roles), args.Workspace) early in execCmdInitiateUpdateInviteRoles

- [x] add integration tests `TestInvite_RolesValidation` in pkg/sys/it/impl_invite_test.go:
  - malformed QName in Roles -> HTTP 400 (`not-a-qname`)
  - sys.WorkspaceOwner in Roles -> HTTP 400
  - non-existent role -> HTTP 400 (`app1pkg.NonExistentRole`)
  - whitespace-only roles, leading comma, duplicate role -> HTTP 400
  - valid app role still passes via existing tests (`TestInvite_BasicUsage` uses `app1pkg.LimitedAccessRole`)

- [x] verify after rename: `go test -run "TestValidateInviteRoles" ./pkg/sys/invite/...`, `TestInvite_RolesValidation`, `TestInvite_BasicUsage`, `TestBasicUsage_InitiateDeactivateWorkspace`, `TestDeactivateJoinedWorkspace` all pass
