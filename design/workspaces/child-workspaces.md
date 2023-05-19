# Motivation

- Air: [Reseller Portal: Invite unTill Payments Users](https://dev.untill.com/projects/#!625718)
- launchpad: [Child Workspaces](https://dev.heeus.io/launchpad/#!25679)


# Concepts

```mermaid
    flowchart TD

    %% Entities ===============================================================

    registry[(sys.registry)]:::H
    ParentWorkspace[(ParentWorkspace)]:::H
    ParentWorkspaceDescriptor[single.sys.WorkspaceDescriptor]:::H
    ParentSubject[cdoc.sys.Subject]:::H

    IAuthenticator["IAuthenticator"]:::S

    cdoc_ChildWorkspace[cdoc.sys.ChildWorkspace]:::H
    ChildWorkspace[(ChildWorkspace)]:::H
    ChildWorkspaceDescriptor[single.sys.WorkspaceDescriptor]:::H
    SomeFunction["SomeFunction()"]:::S
    OwnerWSID([OwnerWSID]):::H
    aproj.sys.CreateWorkspace:::H

    PrincipalToken[PrincipalToken]:::H
    EnrichedPrincipalToken[Enriched PrincipalToken]:::H

    q.EnrichPrincipalToken["q.sys.EnrichPrincipalToken()"]:::S

    %% Relations ===============================================================



    ParentWorkspace --x cdoc_ChildWorkspace
    ParentWorkspace --x ParentSubject
    ParentWorkspace --- ParentWorkspaceDescriptor

    cdoc_ChildWorkspace -.- OwnerWSID

    ChildWorkspace --- ChildWorkspaceDescriptor
    ChildWorkspaceDescriptor --- OwnerWSID
    ChildWorkspace -.- |provides| SomeFunction
    ChildWorkspace --- |created by| aproj.sys.CreateWorkspace
    aproj.sys.CreateWorkspace --- |eventually triggered by| cdoc_ChildWorkspace

    ParentWorkspaceDescriptor -.-> IAuthenticator
    ParentSubject -.-> IAuthenticator

    IAuthenticator -.-> |Subject workspace principals| q.EnrichPrincipalToken

    PrincipalToken -.-> q.EnrichPrincipalToken
    %% PrincipalToken -.- |taken from| registry
    registry -.- |used to issue| PrincipalToken


    q.EnrichPrincipalToken -.-> EnrichedPrincipalToken

    EnrichedPrincipalToken -.-> SomeFunction


    classDef G fill:#FFFFFF,color:#333,stroke:#000000, stroke-width:1px, stroke-dasharray: 5 5
    classDef B fill:#FFFFB5,color:#333
    classDef S fill:#B5FFFF,color:#333
    classDef H fill:#C9E7B7,color:#333

```

## c.sys.EnrichPrincipalToken()

- AuthZ: role.sys.Subject
- Params
    - PrincipalToken
- Returns
    - PrincipalToken enriched with subject workspace principals

## Unclear
- Implicit roles like Owner?
