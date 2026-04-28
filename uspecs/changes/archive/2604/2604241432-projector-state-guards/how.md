# How: Refactor invite flow - single projector and roles validation

## Approach

- Segregate concerns between command and projector
  - Commands persist input data and pre-validate; they do not write transitional `ToBe*` states (except initial `ToBeInvited` written by `InitiateInvitationByEMail`)
  - A single async projector `ap.sys.ApplyInviteEvents` reacts to all invite commands and performs state transitions plus federation calls atomically
  - Projector re-validates the actual state of the Invite record at execution time and skips silently when the state has already moved on (idempotency, replay safety)
- Replace 5 dedicated projectors with one VSQL declaration in `sys.vsql`: `PROJECTOR ApplyInviteEvents AFTER EXECUTE ON (InitiateInvitationByEMail, InitiateJoinWorkspace, InitiateUpdateInviteRoles, InitiateCancelAcceptedInvite, InitiateLeaveWorkspace, CancelSentInvite)`
- Drop the `c.sys.CompleteInvitation` / `c.sys.CompleteJoinWorkspace` commands and the projector test hooks - they are not needed under segregation of concerns
- Preserve numeric values of all `State_*` constants in `consts.go` and accept legacy `ToBe*` states in command pre-validation so old stuck records remain recoverable
- Validate Roles strings in invite commands using `validateInviteRoles` in `utils.go` (split on comma, parse QName, reject `sys.*` package, reject roles not in workspace types)
- Detect system roles by package prefix in `iauthnz/utils.go` (`role.Pkg() == appdef.SysPackage`) rather than maintaining a static `SysRoles` slice

References:

- [pkg/sys/invite/consts.go](../../../../../pkg/sys/invite/consts.go)
- [pkg/sys/invite/provide.go](../../../../../pkg/sys/invite/provide.go)
- [pkg/sys/invite/impl_applyinviteevents.go](../../../../../pkg/sys/invite/impl_applyinviteevents.go)
- [pkg/sys/invite/utils.go](../../../../../pkg/sys/invite/utils.go)
- [pkg/sys/sys.vsql](../../../../../pkg/sys/sys.vsql)
- [pkg/iauthnz/utils.go](../../../../../pkg/iauthnz/utils.go)
- [pkg/sys/it/impl_invite_test.go](../../../../../pkg/sys/it/impl_invite_test.go)
- [uspecs/specs/prod/auth/invites--td.md](../../../../specs/prod/auth/invites--td.md)
- [uspecs/specs/prod/prod--domain.md](../../../../specs/prod/prod--domain.md)
