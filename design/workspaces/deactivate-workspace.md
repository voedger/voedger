# Deactivate Workspace

## Motivation

- [Deactivate Workspace](https://github.com/voedger/voedger/issues/53)


## Principles

- Workspace with WorkspaceDescriptor.IsActive accepts only System token
- Workspace is (consistently) inactive if:
  - Workspace/WorkspaceDescriptor.IsActive == false
  - There is no active JoinedWorkspace which refers to the Workspace
    - Subject's are still active
  - AppWorkspace/WorkspaceID[Workspace].IsActive == true

## c.sys.DeactivateWorkspace()

- AuthZ: role.sys.WorkspaceOwner ???

```mermaid
    sequenceDiagram

    actor owner as WorkspaceOwner
    participant ws as Workspace
    participant appws as ApplicationWS
    participant profile as ProfileWS
    participant registry as regisrty

    owner ->> ws: c.sys.DeactivateWorkspace()
    opt Workspace is active
        ws ->> ws: cdoc.sys.WorkspaceDescriptor.IsActive = false
    end

  note over ws: ap.sys.OnDeactivateWorkspace()

```


```mermaid
    sequenceDiagram

    actor owner as WorkspaceOwner
    participant ws as Workspace
    participant parent as OwnerApp/ParentWS
    participant profile as ProfileWS (cdoc.sys.Subject.ProfileWSID)

    owner ->> ws: c.sys.DeactivateWorkspace()
    opt Workspace is active
        ws ->> ws: cdoc.sys.WorkspaceDescriptor.IsActive = false
    end

    note over ws: ap.sys.DeactivateWorkspaceReferences()
    ws ->> ws: Read cdoc.sys.WorkspaceDescriptor{OwnerApp, OwnerDocID, ParentWSID???}

    ws ->> parent: c.sys.OnChildWorkspaceDeactivated(OwnerDocID)

    opt Docs[OwnerDocID].IsActive
      parent ->> parent: Docs[OwnerDocID].IsActive = false
    end

    opt Foreach cdoc.sys.Subject
        ws ->> profile: c.sys.OnJoinedWorkspaceDeactivated()
        opt JoinedWorkspace.IsActive
          profile ->> profile: JoinedWorkspace.IsActive = false
        end
    end
```

## c.sys.DeactivateWorkspace()

- AuthNZ: System


