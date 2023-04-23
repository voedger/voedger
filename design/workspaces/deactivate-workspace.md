# Deactivate Workspace

## Motivation

- [Deactivate Workspace](https://github.com/voedger/voedger/issues/53)



## Principles

- If Workspace is not active it accepts only System??? token


## c.sys.DeactivateWorkspace()

- AuthZ: role.sys.WorkspaceOwner ???


```mermaid
    sequenceDiagram
    
    actor owner as WorkspaceOwner
    participant ws as Workspace
    participant parent as OwnerApp/ParentWS
    participant profile as ProfileWS
    participant registry as regisrty

    owner ->> ws: c.sys.DeactivateWorkspace()
    opt Workspace is active
        ws ->> ws: cdoc.sys.WorkspaceDescriptor.IsActive = false
    end

    note over ws: ap.sys.DeactivateWorkspaceReferences()
    ws ->> ws: Read cdoc.sys.WorkspaceDescriptor{OwnerApp, OwnerDoc, ParentWSID???}

    ws ->> parent: c.sys.ChildWorkspaceDeactivated(OwnerDoc)

    opt Docs[OwnerDoc].IsActive
      parent ->> parent: Docs[OwnerDoc].IsActive = false
    end

    opt Foreach cdos.sys.Subject    
        registry -->> ws : ProfileWSID by Subject.Login
        ws ->> profile: c.sys.JoinedWorkspaceDeactivated()
        opt JoinedWorkspace.IsActive
          profile ->> profile: JoinedWorkspace.IsActive = false
        end
    end



    


```
