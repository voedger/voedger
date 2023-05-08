## Auth


### Devices

- Owner calls CreateWorkspaceDevice(name)
  - 6 digit code is generated
- Device calls JoinWorkspace(6 digit code)
  - Token is generated and returned
  - Can be called once

### Devices


- Device calls CreateWorkspaceDevice(token,)
- EdgeNode calls CreateWorkspaceLogin

```mermaid
    flowchart TD




    Workspace:::H
    WorkspaceKind:::S

    Workspace --- WorkspaceKind

    WorkspaceKind  --- registry.Registry:::S


       


    EdgeNode:::G
    subgraph EdgeNode
      Login([Login]):::H
      PrincipalToken([PrincipalToken]):::H
      Login -.- PrincipalToken
    end

    Cloud:::G

    subgraph Cloud
      CloudApp:::S
        CloudApp --- Profile[("Profile[Login]")]:::H
        CloudApp --- EdgeNodesRegistry[(edgereg.EdgeNodesRegistry)]:::H
        EdgeNodesRegistry  --- sys.LinkedProfile["sys.LinkedProfile[Profile.WSID]"]:::H
        sys.LinkedProfile
          sys.LinkedProfile --- Role(["Roles['edgereg.EdgeNode']"]):::H
    end

    Profile -.- Login
    sys.LinkedProfile -.- Profile
    
    classDef B fill:#FFFFB5
    classDef S fill:#B5FFFF
    classDef H fill:#C9E7B7
    classDef G fill:#FFFFFF,stroke:#000000, stroke-width:1px, stroke-dasharray: 5 5
```    