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

- Workspace can be AppWorkspace, ProfileWorkspace or ChildWorkspace
- ProfileWorkspace keeps Subject data, including list of ChildWorkspace-s
  - `sys.UserProfile`, `sys.DeviceProfile`
- ChildWorkspace: `air.RestaurantWS` etc.
- Workspace has the OwningDocument
- OwningDocument: a document whose fields {WSID, wsError} will be updated when workspace will be ready
- Currently, OwningDocument kinds: `cdoc.sys.Login`, `cdoc.sys.ChildWorkspace`
- // TODO: Clearing the owner.error causes the workspace to be regenerated
- OwningDocument.error must NOT be published to CUD function (only System can update)

### Workspace Kinds

| English     | Russian     |
| ----------- | ----------- |
| Workspace| Рабочая область       |
| AppWorkspace   |Рабочая область приложения|
| ProfileWorkspace   | Профиль        |
| ChildWorkspace   |Дочерняя рабочая область|

```mermaid
erDiagram
  Workspace||--|| AppWorkspace: "can be"
  Workspace||--|| ProfileWorkspace: "can be"
  Workspace||--|| ChildWorkspace: "can be"

  AppWorkspace ||--|{ cdoc_sys_Login: "e.g. can keep"
  AppWorkspace ||--|{ cdoc_sys_WorkspaceID: "e.g. can keep"

  ProfileWorkspace ||--|{ UserProfile: "can be"
  ProfileWorkspace ||--|{ DeviceProfile: "can be"

  ChildWorkspace ||--|{ air_Restaurant: "e.g. can be"
  ChildWorkspace ||--|{ slack_Organization: "e.g. can be"
```

### Owning Document

> "Doc from App owns Workspace" means that
Workspace.Docs[sys.WorkspaceDescriptor].OwnerID = Doc.ID AND Workspace.Docs[sys.WorkspaceDescriptor].App = Doc.App

```mermaid
erDiagram


  app_sys_registry ||--|{ app_sys_registry_AppWorkspace: "has (10)"

  app_sys_registry_AppWorkspace ||--|| AppWorkspace: "is"

  app_sys_registry_AppWorkspace ||--|{ cdoc_sys_Login: "has"
  cdoc_sys_Login ||--|| ProfileWorkspace: "owns"
  UserApp ||--|| Cluster: "running one per"
  UserApp ||--|{ ProfileWorkspace: "has"

  ProfileWorkspace ||--|{ cdoc_ChildWorkspace: "has"
  cdoc_ChildWorkspace ||--|| ChildWorkspace: "owns"
  AppWorkspace||--|| Workspace: "is"
  ProfileWorkspace||--|| Workspace: "is"
  ChildWorkspace||--|| Workspace: "is"

  Workspace||--|| cdoc_sys_WorkspaceDescriptor: "has"

  cdoc_sys_WorkspaceDescriptor{
    OwnerApp AppQName
    OwnerID RecordID "Owner fields {WSID, WorkspaceError} will be updated by workspace creation proc"
    OwnerWSID WSID
    OwnerQName QName
    CreatedAtMs  int64
    InitStartedAtMs int64
    InitCompletedAtMs int64 "will be updated by workspace creation proc"
    CreateError string "e.g. wrong workspace init data"
    InitError string "will be updated by workspace creation proc"
    WSName string
    WSKind  QName
    WSKindInitializationData string "json"
    TemplateName string
    TemplateParams string
    WSID WSID "current WSID"
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
  AppWorkspace ||--|{ cdoc_sys_WorkspaceID: "has"


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
