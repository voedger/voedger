### Concepts

- [Principles](#principles)
- [Workspace Kinds](#workspace-kinds)
- [Ownership](#ownership)
- [Workspace-related Tables](#workspace-related-tables)
- [Invites](invites.md)
- [Child Workspaces](child-workspaces.md)

### Detailed design

- [Create Workspace v2](create-workspace-v2.md)
- [Deactivate Workspace](deactivate-workspace.md)

### Principles

- Workspace can be App Workspace, ProfileWorkspace or UserWorkspace
- ProfileWorkspace keeps Subject data, including list of UserWorkspace-s
  - "sys.UserProfile", "sys.DeviceProfile"
- UserWorkspace: "air.Restaurant" etc.
- Workspace has OwningDocument
- OwningDocument: a document whose fields {WSID, wsError} will be updated when workspace is ready
- Currently, OwningDocument kinds: CDoc[Login], CDoc[UserWorkspace]
- // TODO: Clearing the owner.error causes the workspace to be regenerated
- OwningDocument.error must NOT be published to CUD function (only System can update)

### Workspace Kinds

| English     | Russian     |
| ----------- | ----------- |
| Workspace| Рабочая область       |
| AppWorkspace   |Рабочая область приложения|
| ProfileWorkspace   | Профиль        |
| UserWorkspace   |Пользовательская рабочая область|

```mermaid
erDiagram
  Workspace||--|| AppWorkspace: "can be"
  Workspace||--|| ProfileWorkspace: "can be"
  Workspace||--|| UserWorkspace: "can be"
  
  AppWorkspace ||--|{ cdoc_sys_Login: "e.g. can keep"
  AppWorkspace ||--|{ cdoc_sys_WorkspaceID: "e.g. can keep"

  ProfileWorkspace ||--|{ UserProfile: "can be"
  ProfileWorkspace ||--|{ DeviceProfile: "can be"

  UserWorkspace ||--|{ air_Restaurant: "e.g. can be"
  UserWorkspace ||--|{ slack_Organization: "e.g. can be"
```  

### Owninig Document

> "Doc from App owns Workspace" means that Workspace.Docs[sys.WorkspaceDescriptor].Owner = Doc.ID AND Workspace.Docs[sys.WorkspaceDescriptor].App = Record.App

```mermaid
erDiagram


  app_sys_registry ||--|{ app_sys_registry_AppWorkspace: "has (10)"

  app_sys_registry_AppWorkspace ||--|| AppWorkspace: "is"

  app_sys_registry_AppWorkspace ||--|{ cdoc_sys_Login: "has"
  cdoc_sys_Login ||--|| ProfileWorkspace: "owns"
  UserApp ||--|| Cluster: "running one per"
  UserApp ||--|{ ProfileWorkspace: "has"

  ProfileWorkspace ||--|{ cdoc_ChildWorkspace: "has"
  cdoc_ChildWorkspace ||--|| UserWorkspace: "owns"
  AppWorkspace||--|| Workspace: "is"
  ProfileWorkspace||--|| Workspace: "is"
  UserWorkspace||--|| Workspace: "is"

  Workspace||--|| cdoc_sys_WorkspaceDescriptor: "has"

  cdoc_sys_WorkspaceDescriptor{
    OwnerApp AppQName
    OwnerDoc ID "Owner fields {WSID, WorkspaceError} will be updated by workspace creation proc"
    InitStartedAtMs  int64
    InitCompletedAtMs int64
    wsKind  QName
    ParentWSID WSID
  }
```

## Workspace-related Tables

```mermaid
erDiagram
 

  ProfileWorkspace ||--o{ cdoc_sys_JoinedWorkspace: "has"
  ProfileWorkspace || -- || Workspace: is

  cdoc_sys_JoinedWorkspace ||--|| cdoc_sys_WorkspaceDescriptor: "refers to another WS"

  AppWorkspace || -- || Workspace: is
  %% ??? one-to-one
  AppWorkspace ||--|| cdoc_sys_WorkspaceID: "has"
  AppWorkspace ||--|{ cdoc_sys_Workspace: "has"
  
  
  Workspace || -- || AppWorkspaceDescriptorCDoc: has
  Workspace || -- || cdoc_sys_WorkspaceDescriptor: has
  Workspace || -- |{ cdoc_sys_Subject: has
  Workspace || -- |{ cdoc_sys_ChildWorkspace: has  



  AppWorkspaceDescriptorCDoc ||--|| cdoc_sys_UserProfile: "can be"
  AppWorkspaceDescriptorCDoc ||--|| cdoc_sys_DeviceProfile: "can be"
  AppWorkspaceDescriptorCDoc ||--|| AnyCustomCDoc: "can be"
  
  cdoc_sys_WorkspaceDescriptor ||--|| cdoc_sys_ChildWorkspace: "refers to parent WS"

  Workspace || -- || WSID: "addressed by"

  WSID || -- || cdoc_sys_WorkspaceID: "Normally taken from of WSID field of"
  WSID || -- || istructs_consts: "for AppWorkspace taken from"

```

## See Also
- [Invites](./invites.md)
