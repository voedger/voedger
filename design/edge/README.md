# Edge Computing Analysis & Design

Motivation

- [dev.untill.com: Local Box](https://dev.untill.com/projects/#!625849)
- [heeus.io/launchpad: Edge Computing](https://dev.heeus.io/launchpad/#!25305)
- [heeus.io/launchpad: cmd: edger](https://dev.heeus.io/launchpad/#!25307)

Concepts

- [Edge Node](#edge-node)
- [Edge Nodes Registry and Status](#edge-nodes-registry-and-status)
- [Edge Node Lifecycle](#edge-node-lifecycle)
- [Asymmetrical asynchronous replication](#asymmetrical-asynchronous-replication)
- [Edge Authentication](#edge-authentication)
- [sys_WorkspaceDescriptor_cdoc](#sys_workspacedescriptor_cdoc)
- [External Projectors](#external-projectors)
- [Synced Events](#synced-events)

Processes

- [Check-out Workspace](#check-out-workspace)
- [Replicate Edge Workspace](#replicate-edge-workspace)
- [Initial Sync](#initial-sync)
- [Process commands](#process-commands)

# Concepts

## Edge Node

```mermaid
    flowchart TB

    %% Entities ====================

    Node:::G
    Cloud:::G

    subgraph Node["Edge Node"]

        
        edger[edger]:::S
        DockerStack[Docker stack]:::S
        CE[CE]:::S
        EdgeWorkspace[(Edge Workspace)]:::H

        edger --- |controls| DockerStack
        DockerStack --- |runs| CE

        CE --x |works with| EdgeWorkspace        
    end    

    subgraph Cloud
        CheckedOutWorkspace[(Checked-out Workspace)]:::H
    end

    EdgeWorkspace  ---> |replicated to| CheckedOutWorkspace

    classDef B fill:#FFFFB5
    classDef S fill:#B5FFFF
    classDef H fill:#C9E7B7
    classDef G fill:#FFFFFF,stroke:#000000, stroke-width:1px, stroke-dasharray: 5 5
``` 
- Edge Workspace: Перефирийная рабочая область
- Checked-out Workspace: Выписанная рабочая область

## Edge Nodes Registry and Status


```mermaid
    flowchart TB

    %% Entities ====================

    Cloud:::G
    subgraph Cloud
        direction TB
        Federation:::H
        Cluster:::H
        CloudApp:::S
        edgeregWS[(edgereg.EdgeNodesRegistry)]:::H
        edgestatWS[(edgestat.EdgeNodesState)]:::H
        edgereg[[edgereg]]:::S

        Federation --x Cluster
        Cluster ---x CloudApp
        
        CloudApp --- |has workspace| edgeregWS
            edgeregWS --- edgereg
                edgereg --- ChangeDesiredNodeState>"ChangeDesiredNodeState()"]:::S
                edgereg --- RegisterNodeInCloud>"RegisterEdgeNode()"]:::S
                edgereg --- UnregisterNodeInCloud>"UnregisterEdgeNode()"]:::S
        CloudApp --x |has workspaces| edgestatWS
            edgestatWS --- edgestatp
                edgestatp[[edgestat]]:::S
                    edgestatp --- UploadState>"UploadEdgeNodeState()"]:::S
                    edgestatp --- ViewState>"ViewEdgeNodesState()"]:::S   
        edgeregWS -.->|partially replicated to| edgestatWS

    end
    EdgeNode:::H
    UploadState --- |used by| EdgeNode:::H
    edgeregWS --x EdgeNode

    classDef B fill:#FFFFB5
    classDef S fill:#B5FFFF
    classDef H fill:#C9E7B7
    classDef G fill:#FFFFFF,stroke:#000000, stroke-width:1px, stroke-dasharray: 5 5
``` 

ViewEdgeNodesState()

- Sorted by CPU usage as a percentage
- Sorted by Memory usage as a percentage
- Sorted by Error state (ones with errors first)
- // TODO: Merge results from all Edge State workspaces

```mermaid
    flowchart TD

    %% Entities ====================

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


## Edge Node Lifecycle

```mermaid
    flowchart TB

    %% Entities ====================

    ConfigureNode --> RegisterNodeInCloud
    RegisterNodeInCloud --> NodeUsage

    NodeUsage:::G


    subgraph NodeUsage

        ChangeDesiredNodeState
        ViewNodeState
        NodeAlerting


        EdgeWorkspace:::G

        subgraph EdgeWorkspace[Edge Workspace]
            direction TB
            CheckOutWorkspace[Check-out Workspace]

            CheckOutWorkspace --> WorkWithEdgeWorkspace
            WorkWithEdgeWorkspace --> CheckInWorkspace

            WorkWithEdgeWorkspace:::G

            subgraph WorkWithEdgeWorkspace
                CQ
                ReplicateEdgeWorkspace
            end

            WorkWithEdgeWorkspace --- ReplicateEdgeWorkspace:::S
        end
    end 

    NodeUsage --> RemoveNodeFromCloud

    classDef default fill:#FFFFB5
    classDef S fill:#B5FFFF
    classDef G fill:#FFFFFF,stroke:#000000, stroke-width:1px, stroke-dasharray: 5 5    
```    

- **Check out Workspace**: Выписывание Рабочей Области
- **Check in Workspace**: Возврат Рабочей Области
- **Change Desired Node State** (software version, etc.)
- **View Node State**: Software version, etc.
- **Node Alerting**: Alerts about CPU, memory  exhaustion


## Asymmetrical asynchronous replication

- Data is replicated one direction
- Data is replicated asynchronously

```mermaid
flowchart LR

    Node:::G
    Cloud:::G

    subgraph Node["Edge Node"]
        EdgeBLOBs[(BLOBStorage)]:::H
        EdgeWorkspace[(Edge Workspace)]:::H
        Replicator["Projector[A, Replicator]"]:::S

        EdgeBLOBs -.-> Replicator
        EdgeWorkspace -.->  Replicator
    end

    subgraph Cloud
        CloudBLOBs[(BLOBStorage)]:::H
        CheckedOutWorkspace[(Checked-out Workspace)]:::H
    end

    Replicator -.-> CheckedOutWorkspace
    Replicator -.-> CloudBLOBs

classDef default fill:#FFFFB5
classDef S fill:#B5FFFF
classDef G fill:#FFFFFF,stroke:#000000, stroke-width:1px, stroke-dasharray: 5 5
classDef H fill:#C9E7B7
```

## Edge Authentication

```mermaid
flowchart TD

    Node:::G
    Cloud:::G

    subgraph Node["Edge Node"]
        EdgeWorkspace[(Edge Workspace)]:::H
        NodeEdgeWorkspaceEcnryptionKey[EdgeEcnryptionKey]:::H
        
        EdgeWorkspace --- NodeEdgeWorkspaceEcnryptionKey:::H
        EdgeApplication[Application]:::S
        NodeEdgeWorkspaceEcnryptionKey -.-> EdgeApplication

    end

    subgraph Cloud
        CheckedOutWorkspace[(Checked-out Workspace)]:::H
        ClusterEncryptionKey:::H
        CloudEdgeWorkspaceEcnryptionKey[EdgeEcnryptionKey]:::H
        CheckedOutWorkspace --- CloudEdgeWorkspaceEcnryptionKey
        
        IssueAccessToken:::S
        ClusterEncryptionKey -.-> IssueAccessToken
        IssueAccessToken -.-> AccessToken:::H

        AccessToken -.-> IssueEdgeAccessToken:::S
        ClusterEncryptionKey -.-> IssueEdgeAccessToken
        
        CloudEdgeWorkspaceEcnryptionKey -.-> IssueEdgeAccessToken
    end

    IssueEdgeAccessToken -.-> EdgeAccessToken:::H
    EdgeAccessToken -.-> EdgeAuthentication:::V

    EdgeAuthentication -.- EdgeApplication

    NodeEdgeWorkspaceEcnryptionKey -.- CloudEdgeWorkspaceEcnryptionKey


classDef default fill:#FFFFB5
classDef S fill:#B5FFFF
classDef G fill:#FFFFFF,stroke:#000000, stroke-width:1px, stroke-dasharray: 5 5
classDef H fill:#C9E7B7
```

## sys_WorkspaceDescriptor_cdoc

```mermaid
erDiagram

  sys_WorkspaceDescriptor_cdoc{
    EdgeStatus int "0, 1(CheckedOut), 2(Edge)"
    EdgeEncryptionKey string "Base64-encoded string"
    EdgeReplicationOffset int64 "WLog offset Edge Workspace should be replicated from"
  }
```

## Synced Events

```go
type IAbsractEvent struct {
    Synced()  bool
}
```

# Processes

## Check-out Workspace

### Get tokens

```mermaid
sequenceDiagram
	actor a as Edge Node User (ENU)
    participant ui as Some UI
    participant c as Cloud
	participant en as EdgeNode

    a --> ui: 
    ui ->> c: q.sys.IssueAccessToken()
    c -->> ui: CloudAccessToken

    ui ->> en: q.sys.IssueAccessToken()
    en -->> ui: EdgeAccessToken    

    ui->>en: q.sys.InitUserWorkspace(EdgeAccessToken, CloudAccessToken)

```    

### Check-out

```mermaid
sequenceDiagram
	participant enup as EdgeNode/app/ENUProfile/c,q
    participant wsp as EdgeNode/app/NewWS/Projectors
    participant cow as Сloud/app/CheckedOutWS/c,q
    participant wsc as EdgeNode/app/NewWS/c,q

    Note over enup: q.sys.InitUserWorkspace(EdgeAccessToken, CloudAccessToken)

    enup --)wsp: Projectors<A, InitializeWorkspace>
    loop InitialSync: until WLog delta is empty
        wsp ->> cow: q.sys.ReadWLog(CloudAccessToken)
        cow -->> wsp: WLog delta
        wsp ->> wsc: Apply WLog delta???
        wsp --> wsp: wait for some projectors???
    end
    wsp ->> cow: c.sys.CheckOut()
    cow ->> cow: sys_WorkspaceDescriptor_cdoc.EdgeStatus = 1
    cow ->> cow: sys_WorkspaceDescriptor_cdoc.EdgeEncryptionKey = ...
    cow -->> wsp: 
    Note over cow: Commands are not accepted anymore (but CheckIn)

    wsp ->> cow: q.sys.GetEdgeEncryptionKey()
    cow -->> wsp: EdgeEncryptionKey

    wsp ->> wsc: q.sys.GetWLogOffset()
    wsc -->> wsp: EdgeReplicationOffset

    wsp ->> wsc: c.sys.CUD(EdgeStatus = 2, EdgeEncryptionKey, EdgeReplicationOffset)
```

## Actualize External Projection

```mermaid
erDiagram
  ProjectorDef {
    External bool
  }
```
- Like Mailer

Requirements
- Skip Synced Events in External Projectors
  - if Event.Synced() == true


## Replicate Edge Workspace

Requirements
- Resulting Events in Checked-out Workspace must be Synced Events

## Initial Sync

Requirements
- Resulting Events in Edge Workspace must be Synced Events

## Process commands

Requirements
- Error if EdgeStatus == CheckedOut

# ???

- User access Edge using browser which do not have a token yet
- Edge User wants to link New device to the workspace