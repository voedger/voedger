---
registered_at: 2026-04-24T10:00:36Z
change_id: 2604241000-projector-state-guards
baseline: f91d1438f6fbf47ffd247bb8d06e47cfdff55e6e
issue_url: https://untill.atlassian.net/browse/AIR-3704
pre_cleanup_head: a917956fcaeeeb1ae5acdacabad2f83a316da9dc
pre_pr_head: 469ba0d4bcb10345ea6c19f4f2a95c607d09631d
archived_at: 2026-04-24T14:32:29Z
---

# Change request: Refactor invite flow - single projector and roles validation

## Why

Invitations can get stuck in transitional states (`State_ToBeInvited`, `State_ToBeJoined`) when async projectors fail. The `ApplyJoinWorkspace` projector had an early-return bug that skipped updating invite state when an inactive Subject existed, leaving invites permanently stuck in `State_ToBeJoined`. Even after fixing the projector (change [2604221416-fix-reinvite-after-removal](../2604221416-fix-reinvite-after-removal/change.md)), already-stuck invites could not be recovered because commands `InitiateInvitationByEMail` and `CancelSentInvite` did not accept these transitional states.

PR [#4511](https://github.com/voedger/voedger/pull/4511) restored recovery by letting those commands accept `ToBe*` states. Reviewers then identified a race between commands and async projectors `ApplyInvitation` / `ApplyJoinWorkspace`: stale projector events could overwrite recovered state (e.g., flip `Cancelled` back to `Invited`). The initial plan added projector state guards plus dedicated commands `c.sys.CompleteInvitation` / `c.sys.CompleteJoinWorkspace`, but analysis showed this preserved the underlying problem - commands and projectors both mutating Invite state. A simpler design segregates concerns: commands accept input, pre-validate, and persist data fields only; a single projector applies state transitions and side effects atomically.

A second concern surfaced during this work: invite Roles were never validated, so any workspace owner could grant arbitrary, malformed, or `sys.*` roles.

## What

Invite flow refactor (segregation of concerns):

- Replace 5 separate invite projectors with a single `ap.sys.ApplyInviteEvents` projector reacting to all invite commands
- Move all invite state transitions and side effects out of commands into the single projector
- Keep commands responsible for data writes and pre-validation only; remove non-initial `State` writes
- Remove `c.sys.CompleteInvitation` and `c.sys.CompleteJoinWorkspace` commands and their params
- Remove projector test hooks (`OnBeforeApply*`, `OnAfterGuard*`) and the projector-state-guards machinery
- Preserve all `State_*` numeric values for backward compatibility; keep `ToBeInvited` written by `InitiateInvitationByEMail` as the only legitimate `ToBe*` write
- Keep legacy `ToBe*` states accepted by command pre-validation so old stuck records remain recoverable

Roles validation:

- Add `validateInviteRoles` helper that rejects malformed QNames, system roles (`sys.*`), and roles not declared in the target workspace
- Call `validateInviteRoles` in `InitiateInvitationByEMail` and `InitiateUpdateInviteRoles`
- Convert `iauthnz.IsSystemRole` to a package-prefix check (`role.Pkg() == appdef.SysPackage`) instead of a static slice
- Rename test role `app1pkg.WorkspaceSubject` to `app1pkg.InviteTestRole` for clarity

Specifications:

- Update `uspecs/specs/prod/prod--domain.md`: expand auth context with workspace membership
- Update `uspecs/specs/prod/auth/invites--td.md`: replace state diagram and projector design with single-projector model, mark legacy `ToBe*` states
