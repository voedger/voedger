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
- Deactivating a previously created workspaces is possible but nothing will be made on `c.sys.OnJoinedWorkspaceDeactivated` beacuse:
  - there was no `sp.sys.WorkspaceIDIdx`
  - there was no field `view.sys.WorkspaceIDIdx.InvitingWorkspaceWSID`

## c.sys.InitiateDeactivateWorkspace()

- AuthZ: role.sys.WorkspaceOwner
- Params: none
- `cdoc.sys.WorkspaceID` existence in appWS is checked by `view.sys.WorkspaceIDIdx` but there was no this view before. So need to check the existence of the link to `cdoc.sys.WorkspaceID` before checking `cdoc.sys.WorkspaceID.IsActive`

```mermaid
    sequenceDiagram

    actor owner as WorkspaceOwner
    participant ws as Workspace
    participant appws as currentApp/ApplicationWS
    participant profile as ProfileWS
    participant ownerWS as OwnerApp/OwnerWS

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
    opt cdoc.sys.WorkspaceID could be found && !cdoc.sys.WorkspaceID.IsActive
      appws ->> appws: WorkspaceID.IsActive = false
    end

    ws ->> ownerWS: c.sys.OnChildWorkspaceDeactivated(ownerID)
    opt cdocs[ownerID].IsActive
      ownerWS ->> ownerWS: cdocs[ownerID].IsActive = false
    end

    ws ->> ws: c.sys.CUD: cdoc.sys.WorkspaceDescriptor.Status = Inactive
```
