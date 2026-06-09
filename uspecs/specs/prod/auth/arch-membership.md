# Context subsystem architecture: prod/auth/membership

Workspace membership subsystem architecture covering the invite lifecycle, the subjects doc, joined-workspace records, role updates, and member removal. Context-level overview and shared concepts: [arch.md](./arch.md). Consumer of the subjects doc on every request: [arch-authz.md](./arch-authz.md). Profile workspace creation that is the prerequisite for any user joining a workspace: see the `apps` context [../apps/arch.md](../apps/arch.md).

## External actors

Roles:

- `@WorkspaceOwner`
  - Caller acting in the inviting workspace; sends and cancels invites, updates roles, removes members.

- `@Invitee`
  - Caller named in an invite (by email at invite time, by login after first sign-in); joins or leaves the workspace.

## Scenarios overview

- **`Send invite by email`**
  - `@WorkspaceOwner` calls `[c.sys.InitiateInvitationByEMail]`; an `[(cdoc.sys.Invite)]` is created in `State_ToBeInvited`, the async invite-events projector enqueues an email with a verification code via `[Send invite email]`, and the invite transitions to `State_Invited`.

- **`Join workspace`**
  - `@Invitee` calls `[c.sys.InitiateJoinWorkspace]` with the verification code from the email; `[ap.sys.ApplyInviteEvents]` persists `[(cdoc.sys.Subject)]` in the inviting workspace with the granted roles, calls `[c.sys.CreateJoinedWorkspace]` in the invitee's profile (creates `[(cdoc.sys.JoinedWorkspace)]`), and writes `State_Joined` on `[(cdoc.sys.Invite)]`.

- **`Update roles`**
  - `@WorkspaceOwner` calls `[c.sys.InitiateUpdateInviteRoles]`; `[ap.sys.ApplyInviteEvents]` rewrites the role list on `[(cdoc.sys.Subject)]` and on the invitee's `[(cdoc.sys.JoinedWorkspace)]` via `[c.sys.UpdateJoinedWorkspaceRoles]`. `[(cdoc.sys.Invite)]` remains in `State_Joined`.

- **`Cancel sent invite`**
  - `@WorkspaceOwner` calls `[c.sys.CancelSentInvite]`; `[ap.sys.ApplyInviteEvents]` writes `State_Cancelled` on `[(cdoc.sys.Invite)]`. No `[(cdoc.sys.Subject)]` exists yet, so no further cleanup is required.

- **`Cancel accepted invite (remove member)`**
  - `@WorkspaceOwner` calls `[c.sys.InitiateCancelAcceptedInvite]`; `[ap.sys.ApplyInviteEvents]` deactivates `[(cdoc.sys.Subject)]` in the inviting workspace, calls `[c.sys.DeactivateJoinedWorkspace]` in the invitee's profile, and writes `State_Cancelled` on `[(cdoc.sys.Invite)]`.

- **`Leave workspace`**
  - `@Invitee` calls `[c.sys.InitiateLeaveWorkspace]` from its own profile; `[ap.sys.ApplyInviteEvents]` deactivates `[(cdoc.sys.Subject)]` and `[(cdoc.sys.JoinedWorkspace)]` the same way the cancel flow does, and writes `State_Left` on `[(cdoc.sys.Invite)]`.

## Components

### Layers

```text
External actors
    |
    +-- @WorkspaceOwner
    +-- @Invitee
    |
    v
Invite lifecycle commands
    |
    +-- [c.sys.InitiateInvitationByEMail]
    +-- [c.sys.InitiateJoinWorkspace]
    +-- [c.sys.InitiateUpdateInviteRoles]
    +-- [c.sys.InitiateCancelAcceptedInvite]
    +-- [c.sys.InitiateLeaveWorkspace]
    +-- [c.sys.CancelSentInvite]
    |
    v
Membership write commands (projector-emitted)
    |
    +-- [c.sys.CreateJoinedWorkspace]
    +-- [c.sys.UpdateJoinedWorkspaceRoles]
    +-- [c.sys.DeactivateJoinedWorkspace]
    |
    v
Async invite-events projector
    |
    +-- [ap.sys.ApplyInviteEvents]
    +-- [Send invite email]
    |
    v
Records and indexes
    |
    +-- [(cdoc.sys.Invite)]
    +-- [(cdoc.sys.JoinedWorkspace)]
    +-- [(cdoc.sys.Subject)]
    +-- [(view.sys.InviteIndexView)]
    +-- [(view.sys.JoinedWorkspaceIndexView)]
    +-- [(view.sys.ViewSubjectsIdx)]
```

### Invite lifecycle commands

- `[c.sys.InitiateInvitationByEMail]`, `[c.sys.InitiateJoinWorkspace]`, `[c.sys.InitiateUpdateInviteRoles]`, `[c.sys.InitiateCancelAcceptedInvite]`, `[c.sys.InitiateLeaveWorkspace]`, `[c.sys.CancelSentInvite]`
  - Owner-side and invitee-side state-transition commands on `[(cdoc.sys.Invite)]`. Allowed transitions are gated by `inviteValidStates` (per command) and `reInviteAllowedForState` (re-invite on cancelled/left/etc). Records stuck in `ToBe*` legacy states are tolerated to allow recovery on old data.
  - impl: [pkg/sys/invite/provide.go](../../../../pkg/sys/invite/provide.go), [pkg/sys/invite/consts.go#inviteValidStates](../../../../pkg/sys/invite/consts.go)

### Membership write commands

- `[c.sys.CreateJoinedWorkspace]`, `[c.sys.UpdateJoinedWorkspaceRoles]`, `[c.sys.DeactivateJoinedWorkspace]`
  - Emitted by `[ap.sys.ApplyInviteEvents]` running in the invitee's profile workspace. Maintain the invitee-side view of which workspaces they belong to.
  - impl: [pkg/sys/invite/provide.go](../../../../pkg/sys/invite/provide.go)

### Async invite-events projector

- `[ap.sys.ApplyInviteEvents]`
  - Reacts to invite state transitions and: (1) sends the invitation email via `[Send invite email]` when entering `State_ToBeInvited`, (2) writes `[(cdoc.sys.Subject)]` in the inviting workspace on join, (3) calls the invitee-profile `Create/Update/Deactivate JoinedWorkspace` commands on join/role-update/leave/cancel.
  - impl: [pkg/sys/invite/impl_applyinviteevents.go](../../../../pkg/sys/invite/impl_applyinviteevents.go)

- `[Send invite email]`
  - SMTP-backed side effect performed by `[ap.sys.ApplyInviteEvents]`; renders the email with `EmailTemplatePlaceholder_*` placeholders defined in `pkg/sys/invite/consts.go`.

### Records and indexes

- `[(cdoc.sys.Invite)]`
  - Per-invitation record in the inviting workspace; carries `State`, `Email`, `Login`, `Roles`, `InviteeProfileWSID`, `SubjectID`, `JoinedWorkspaceID`, `Updated`, `Created`, `ExpireDatetime`.
  - decl: [pkg/sys/invite/consts.go#QNameCDocInvite](../../../../pkg/sys/invite/consts.go)

- `[(cdoc.sys.JoinedWorkspace)]`
  - Per-membership record in the invitee's profile workspace; carries the WSID of the inviting workspace, the granted roles, and an active flag.
  - decl: [pkg/sys/invite/consts.go#QNameCDocJoinedWorkspace](../../../../pkg/sys/invite/consts.go)

- `[(cdoc.sys.Subject)]`
  - Shared concept; see [arch.md#shared-concepts](./arch.md#shared-concepts). Workspace-scoped role-grant record consumed on every request by `[Subjects reader]` in [arch-authz.md](./arch-authz.md).

- `[(view.sys.InviteIndexView)]`, `[(view.sys.JoinedWorkspaceIndexView)]`, `[(view.sys.ViewSubjectsIdx)]`
  - Sync-projector-maintained lookup views (by email, by inviting WSID, by login) used by the invite lifecycle commands and by `[Subjects reader]`.
  - decl: [pkg/sys/invite/consts.go](../../../../pkg/sys/invite/consts.go)

## Scenarios

### Send invite, join, update, leave

```text
@WorkspaceOwner -> [c.sys.InitiateInvitationByEMail] -> [(cdoc.sys.Invite)] State_ToBeInvited
  -> [ap.sys.ApplyInviteEvents] -> [Send invite email] -> State_Invited
@Invitee -> [c.sys.InitiateJoinWorkspace](code)
  -> [ap.sys.ApplyInviteEvents]
       -> @inviting WSID: persist [(cdoc.sys.Subject)] with granted roles
       -> @invitee profile: [c.sys.CreateJoinedWorkspace] -> [(cdoc.sys.JoinedWorkspace)] active
       -> [(cdoc.sys.Invite)] State_Joined
@WorkspaceOwner -> [c.sys.InitiateUpdateInviteRoles]
  -> [ap.sys.ApplyInviteEvents]
       -> @inviting WSID: rewrite roles on [(cdoc.sys.Subject)]
       -> @invitee profile: [c.sys.UpdateJoinedWorkspaceRoles] -> rewrite roles on [(cdoc.sys.JoinedWorkspace)]
       -> [(cdoc.sys.Invite)] State_Joined (unchanged)
@Invitee -> [c.sys.InitiateLeaveWorkspace]
  -> [ap.sys.ApplyInviteEvents]
       -> @inviting WSID: deactivate [(cdoc.sys.Subject)]
       -> @invitee profile: [c.sys.DeactivateJoinedWorkspace]
       -> [(cdoc.sys.Invite)] State_Left
```

### Cancel sent or accepted invite

```text
@WorkspaceOwner -> [c.sys.CancelSentInvite]
  -> [ap.sys.ApplyInviteEvents] -> [(cdoc.sys.Invite)] State_Cancelled (no Subject exists)
@WorkspaceOwner -> [c.sys.InitiateCancelAcceptedInvite]
  -> [ap.sys.ApplyInviteEvents]
       -> @inviting WSID: deactivate [(cdoc.sys.Subject)]
       -> @invitee profile: [c.sys.DeactivateJoinedWorkspace]
       -> [(cdoc.sys.Invite)] State_Cancelled
```

## Notes

`[(cdoc.sys.Subject)]` and `[(cdoc.sys.JoinedWorkspace)]` are written by separate flows in different workspaces; the authoritative source of "member role at request time" consumed by [arch-authz.md](./arch-authz.md) is `[(cdoc.sys.Subject)]` in the request's `RequestWSID`, not `[(cdoc.sys.JoinedWorkspace)]`. The joined-workspace records exist for the invitee's own listing UI and for deactivation symmetry; ACL evaluation does not read them.
