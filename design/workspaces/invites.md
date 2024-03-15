# Invites

Invite Users/Devices to Workspaces

## Motivation

- [Air: Reseller Portal: Invite unTill Payments Users](https://dev.untill.com/projects/#!625718)
- [A&D: Invites](https://dev.heeus.io/launchpad/#!25709)
  - [invites.md](https://github.com/heeus/heeus-design/blob/d9b14d105ef443a2f70cc6fc8530ab42f36a6f5d/workspaces/invites.md)
- [A&D: Invites: Review](https://dev.heeus.io/launchpad/#!25847)

## Contents

- [Concepts](#concepts)
- [Invite State Diagram](#invite-state-diagram)
  - [Main sequence](#main-sequence)
  - [Extra](#extra)

Commands
- [c.sys.InitiateInvitationByEMail()](#csysinitiateinvitationbyemail)
- [c.sys.InitiateJoinWorkspace()](#csysinitiatejoinworkspace)
- [c.sys.InitiateUpdateInviteRoles()](#csysinitiateupdateinviteroles)
- [c.sys.InitiateCancelAcceptedInvite()](#csysinitiatecancelacceptedinvite)
- [c.sys.InitiateLeaveWorkspace()](#csysinitiateleaveworkspace)
- [c.sys.CancelSentInvite()](#csyscancelsentinvite)

Docs
- [cdoc.sys.Invite](#cdocsysinvite)
- [cdoc.sys.Subject](#cdocsyssubject)

## Concepts

```mermaid
    flowchart TD

    WorkspaceOwner["role.sys.WorkspaceOwner"]:::B
    WorkspaceAdmin["role.sys.WorkspaceAdmin"]:::B
    SubjectRole["role.sys.Subject"]:::B
    Inviter["Inviter"]:::B
    Invitee["Invitee"]:::B


    registry[(registry)]:::H
        Login["cdoc.Login"]:::H

    InviteeProfile[(InviteeProfile)]:::H
    JoinedWorkspace["cdoc.sys.JoinedWorkspace"]:::H


    InvitingWorkspace[(InvitingWorkspace)]:::H
        InvitingWorkspace --x Invite["cdoc.sys.Invite"]:::H
        Invite --- State(["State"]):::H
        Invite --- InviteRoles(["Roles"]):::H
        InvitingWorkspace --x Subject["cdoc.sys.Subject"]:::H

    InvitesService([InvitesService]):::S

    Subject -.- Invite
    Subject -.- |gives| SubjectRole

    InviteeProfile --- JoinedWorkspace

    InvitesService -.- |creates| Subject
    InvitesService -.- |reads| Invite
    InvitesService -.- |can create| Login
    InvitesService -.- |creates| JoinedWorkspace

    Inviter -.- |creates, updates| Invite
    Inviter -.- |must be| WorkspaceAdmin

    WorkspaceOwner -.- |is| WorkspaceAdmin

    registry --x Login

    Invitee x-.- |joins WS using| Invite
    Invitee --- InviteeProfile

    JoinedWorkspace -.-x InvitingWorkspace


    classDef G fill:#FFFFFF,color:#333,stroke:#000000, stroke-width:1px, stroke-dasharray: 5 5
    classDef B fill:#FFFFB5,color:#333
    classDef S fill:#B5FFFF,color:#333
    classDef H fill:#C9E7B7,color:#333

```
## Invite State Diagram

### Main sequence
```mermaid
stateDiagram-v2

    [*] --> ToBeInvited : c.sys.InitiateInvitationByEMail() by Inviter
    ToBeInvited --> Invited: ap.sys.ApplyInvitation() // send EMail

    Invited --> ToBeInvited: c.sys.InitiateInvitationByEMail() by Inviter
    Invited --> ToBeJoined: c.sys.InitiateJoinWorkspace() by Invitee
    Invited --> Cancelled: c.sys.CancelSentInvite() by Inviter

    ToBeJoined --> Joined: ap.sys.ApplyJoinWorkspace()

    Joined --> ToBeCancelled: c.sys.InitiateCancelAcceptedInvite() by Inviter
    Joined --> ToBeLeft: c.sys.InitiateLeaveWorkspace() by Invitee
    Joined --> ToUpdateRoles: c.sys.InitiateUpdateInviteRoles() by Inviter

    ToUpdateRoles --> Joined: ap.sys.ApplyUpdateInviteRoles()

    ToBeLeft --> Left: ap.sys.ApplyLeaveWorkspace()

    ToBeCancelled --> Cancelled: ap.sys.ApplyCancelAcceptedInvite()
```

### Extra

```mermaid
stateDiagram-v2
    Cancelled --> ToBeInvited: c.sys.InitiateInvitationByEMail() by Inviter
    Left --> ToBeInvited: c.sys.InitiateInvitationByEMail() by Inviter
```

## c.sys.InitiateInvitationByEMail()

- AuthZ: role.sys.WorkspaceAdmin
- Params
    - Email
    - Roles
    - ExpireDatetime
    - EmailTemplate // Must be prefixed with 'text:' or 'resource:'
        - Placeholders:
            - ${VerificationCode}
            - ${InviteID}
            - ${WSID} - Inviting Workspace ID
            - ${WSName} - Inviting Workspace Name
            - ${Email} - Invitee Email
    - EmailSubject
- Errors
    - State not in (None, Cancelled, Left, Invited)
    - invalid argument EmailTemplate
- //TODO: EMail => Login must be implemented, currently it is supposed that EMail == Login
```mermaid
    sequenceDiagram

    actor Inviter
    participant workspace as InvitingWorkspace
    actor user as Invitee

    Inviter ->> workspace: c.sys.InitiateInvitationByEMail()
    activate workspace
        opt cdoc.sys.Subject exists by login && cdoc.sys.Invite.State is !Cancelled & !Left (allow re-invite otherwise)
            workspace -->> Inviter: subject already exists
        end
        workspace ->> workspace: Create/Update cdoc.Invite, State=ToBeInvited, Login=args.Email, Email, Roles, ExpireDatetime
        note over workspace: Update if exists cdoc.Invite where Login == args.EMail
        workspace -->> Inviter: OK
    deactivate workspace

    Inviter ->> Inviter: Wait for State = Invited

    note over workspace: ap.sys.ApplyInvitation()
    activate workspace
        workspace ->> workspace: Prepare Email
        workspace -->> user: Send invitation Email
        workspace ->> workspace: Update cdoc.Invite State=Invited
    deactivate workspace
```

## cdoc.sys.Invite

- ID
- SubjectKind ([User/Device](https://github.com/heeus/core-istructs/blob/b95ff00ea97f3731f58b8d95f71914f29786e6bf/types.go#L81))
- Login // actually `c.sys.InitiateInvitationByEmail.EMail`
- Email // actually `c.sys.InitiateInvitationByEmail.EMail`
- Roles (comma-separated)
- ExpireDatetime (unix-timestamp)
- VerificationCode
- State
- Created (unix-timestamp) ???
- Updated (unix-timestamp) ???
- SubjectID (Subject.ID) // by ap.sys.ApplyJoinWorkspace
- InviteeProfileWSID     // by ap.sys.ApplyJoinWorkspace
- ActualLogin            // `token.Login`, by ap.sys.ApplyJoinWorkspace

## c.sys.InitiateJoinWorkspace()

- AuthZ: PrincipalToken + VerificationCode
- Params
  - InviteID
  - VerificationCode
- Errors
  - Invite state is not in (Invited)
  - Invite does not exist
  - Invite expired
  - wrong Verification Code

```mermaid
    sequenceDiagram

    actor Invitee
    participant front as Frontend
    participant workspace as InvitingWorkspace
    participant profilews as InviteeProfile

    Invitee ->> front: Confirmation URL
    front ->> front: Sign in or CreateLogin

    front ->> workspace: sys.c.InitiateJoinWorkspace()
    activate workspace
        workspace ->> workspace: Verify VerificationCode
        workspace ->> workspace: Read InviteeProfileWSID, SubjectKind from State
        workspace ->> workspace: Update cdoc.Invite: State=ToBeJoined, InviteeProfileWSID, SubjectKind
        workspace -->> front: OK
    deactivate workspace

    note over workspace: ap.sys.ApplyJoinWorkspace()
    activate workspace
        note over workspace: cdoc.sys.SubjectIdx exists by Invite.Login or by Invite.ActualLogin -> do nothing
        workspace ->> profilews: sys.CreateJoinedWorkspace()
        profilews ->> profilews: create cdoc.sys.JoinedWorkspace if not exists

        workspace ->> workspace: Create cdoc.sys.Subject If not exists
        workspace ->> workspace: Update cdoc.Invite
        workspace ->> workspace: ...State=Joined, SubjectID, Login

    deactivate workspace

    front -->> workspace: Wait for  cdoc.sys.Invite.State == Joined

```

## cdoc.sys.Subject

- Login // old stored records -> `Invite.Login` that is actually `c.sys.InitiateInvitationByEMail.Email`, new records (starting from https://github.com/voedger/voedger/issues/1107) - `Invite.ActualLogin` that is login from token
- SubjectKind ([User/Device](https://github.com/heeus/core-istructs/blob/b95ff00ea97f3731f58b8d95f71914f29786e6bf/types.go#L81))
- Roles (comma-separated list)

## c.sys.InitiateUpdateInviteRoles()

- AuthZ: role.sys.WorkspaceAdmin
- Params
    - InviteID
    - Roles
    - EmailTemplate // Must be prefixed with 'text:' or 'resource:'
    - EmailSubject
- Errors
    - State not in (Joined)
    - invalid argument EmailTemplate


```mermaid
    sequenceDiagram

    actor inviter as Inviter
    participant workspace as TargetWorkspace
    participant profilews as InviteeProfile
    actor Invitee

    inviter ->> workspace: c.sys.InitiateUpdateInviteRoles()
    activate workspace
        workspace ->> workspace: Update cdoc.sys.Invite State=ToUpdateRoles
        workspace -->> inviter: OK
    deactivate workspace

    note over workspace: ap.sys.ApplyUpdateInviteRoles()
    activate workspace
        workspace ->> workspace: Update cdoc.sys.Subject.Roles
        workspace ->> profilews: sys.UpdateJoinedWorkspaceRoles(TargetWorkspaceWSID, Roles)
        profilews ->> profilews: Update cdoc.sys.JoinedWorkspace.Roles
        profilews -->> workspace: OK
        workspace -->> Invitee: "Your roles updated" Email
        workspace ->> workspace: Update cdoc.sys.Invite.State=Joined, Roles
    deactivate workspace

    inviter -->> workspace: Wait for  cdoc.sys.Invite.State == Joined

```

## c.sys.InitiateCancelAcceptedInvite()

- AuthZ: role.sys.WorkspaceAdmin
- Params
    - InviteID
- Errors
    - State not in (Joined)

```mermaid
sequenceDiagram

    actor inviter as Inviter
    participant workspace as InvitingWorkspace
    participant profilews as InviteeProfile

    inviter ->> workspace: c.sys.InitiateCancelAcceptedInvite()
    activate workspace
        workspace ->> workspace: Update cdoc.sys.Invite State=ToBeCancelled
        workspace -->> inviter: OK
    deactivate workspace

    note over workspace: ap.sys.ApplyCancelAcceptedInvite()
    activate workspace
        workspace ->> workspace: Update cdoc.sys.Subject IsActive=0
        workspace ->> profilews: sys.DeactivateJoinedWorkspace(InvitingWorkspaceWSID)
        profilews ->> profilews: Update cdoc.sys.JoinedWorkspace IsActive=0
        profilews -->> workspace: OK
        workspace ->> workspace:: Update cdoc.sys.Invite State=Cancelled
    deactivate workspace

    inviter -->> workspace: Wait for  cdoc.sys.Subject.State == Cancelled

```

## c.sys.InitiateLeaveWorkspace()

- AuthZ: role.sys.Subject
- Params
    - Token.Login is used to find Invite
- Errors
    - Invite not found
    - State not in (Joined)

```mermaid
sequenceDiagram

    actor Invitee
    participant workspace as TargetWorkspace
    participant profilews as InviteeProfile

    Invitee ->> workspace: c.sys.InitiateLeaveWorkspace()
    activate workspace
        workspace ->> workspace: Update cdoc.sys.Invite State=ToBeLeft
        workspace -->> Invitee: OK
    deactivate workspace

    note over workspace: ap.sys.ApplyLeaveWorkspace()
    activate workspace
        workspace ->> workspace: Update cdoc.sys.Subject IsActive=0
        workspace ->> profilews: sys.DeactivateJoinedWorkspace(InvitingWorkspaceWSID)
        profilews -->> workspace: OK
        workspace ->> workspace: Update cdoc.sys.Invite State=Left
    deactivate workspace

    Invitee -->> profilews: Wait for cdoc.JoinedWorkspace deactivation
```

## c.sys.CancelSentInvite()

- AuthZ: role.sys.WorkspaceAdmin
- Params
    - InviteID
- Errors
    - State not in (Invited)

```mermaid
    sequenceDiagram

    actor inviter as Inviter
    participant workspace as TargetWorkspace

    inviter ->> workspace: c.sys.CancelSentInvite()
    activate workspace
        workspace ->> workspace: Update cdoc.sys.Invite State=Cancelled
        workspace -->> inviter: OK
    deactivate workspace
```
