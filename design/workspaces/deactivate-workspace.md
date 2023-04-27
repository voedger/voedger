# Deactivate Workspace

## Motivation

- [Deactivate Workspace](https://github.com/voedger/voedger/issues/53)


## Principles

- Workspace with WorkspaceDescriptor.Status != Active accepts only System token
- Workspace is (consistently) inactive if:
  - Workspace/WorkspaceDescriptor.Status == Inactive
  - There is no any active JoinedWorkspace record which refers to the Workspace
  - Note that Workspace.Subject records are still active
  - AppWorkspace/WorkspaceID[Workspace].IsActive == false

## c.sys.InitiateDeactivateWorkspace()

???: Add ProfileWSD to Subject?

- AuthZ: role.sys.WorkspaceOwner ???
- Params: none

```mermaid
    sequenceDiagram

    actor owner as WorkspaceOwner
    participant ws as Workspace
    participant appws as ApplicationWS
    participant profile as ProfileWS
    participant registry as regisrty

    owner ->> ws: c.sys.InitiateDeactivateWorkspace()
    opt WorkspaceDescriptor.Status != Active
      note over ws: error "Workspace Status is not Active"
    end

    ws ->> ws: cdoc.sys.WorkspaceDescriptor.Status = ToBeDeactivated

    note over ws: ap.sys.ApplyDeactivateWorkspace()
    opt foreach cdos.sys.Subject
      registry -->> ws : ProfileWSIDByLogin()
      ws ->> profile: c.sys.OnJoinedWorkspaceDeactivated()
      opt JoinedWorkspace.IsActive
        profile ->> profile: JoinedWorkspace.IsActive = false
      end
    end

    ws ->> appws: sys.OnWorkspaceDeactivated()
    opt ! WorkspaceID.IsActive
      appws ->> appws: WorkspaceID[ID(WSID)].IsActive = false
    end

    ws ->> ws: cdoc.sys.WorkspaceDescriptor.Status = Inactive

```

## c.sys.DeactivateWorkspace()

- AuthNZ: System


