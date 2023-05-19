# Deactivate Workspace

## Motivation

- [Deactivate Workspace](https://github.com/voedger/voedger/issues/53)


## Principles

- Workspace with WorkspaceDescriptor.Status != Active accepts only System token. 403 forbidden otherwise
- Workspace is (consistently) inactive if:
  - Workspace/WorkspaceDescriptor.Status == Inactive
  - There is no any active JoinedWorkspace record which refers to the Workspace
  - Note that Workspace.Subject records are still active
  - AppWorkspace/WorkspaceID[Workspace].IsActive == false
- The following case is possible: `cdoc.sys.WorkspaceID.IsActive == true` but it is impossible to work there because `cdoc.sys.WorkspaceDescriptor.Status` != Active already. Consistency is gauranteed within a single partition only, here there are 2 different partitions

## c.sys.InitiateDeactivateWorkspace()

- AuthZ: role.sys.WorkspaceOwner ???
- Params: none

```mermaid
    sequenceDiagram

    actor owner as WorkspaceOwner
    participant ws as Workspace
    participant appws as currentApp/ApplicationWS
    participant profile as ProfileWS

    owner ->> ws: c.sys.InitiateDeactivateWorkspace()
    opt WorkspaceDescriptor.Status != Active
      note over ws: error "Workspace Status is not Active"
    end

    ws ->> ws: cdoc.sys.WorkspaceDescriptor.Status = ToBeDeactivated

    note over ws: ap.sys.ApplyDeactivateWorkspace()
    opt foreach cdos.sys.Subject
      ws ->> profile: c.sys.OnJoinedWorkspaceDeactivated(currentWSID)
      opt JoinedWorkspace.IsActive
        profile ->> profile: JoinedWorkspace.IsActive = false
      end
    end

    ws ->> appws: sys.OnWorkspaceDeactivated(ownerWSID, wsName)
    appws ->> appws: read IDOfCDocWorkspaceID from view.sys.WorkspaceIDIdx
    opt exists && !WorkspaceID.IsActive
      appws ->> appws: WorkspaceID.IsActive = false
    end

    ws ->> ws: c.sys.CUD: cdoc.sys.WorkspaceDescriptor.Status = Inactive

```

## c.sys.DeactivateWorkspace()

- AuthNZ: System


