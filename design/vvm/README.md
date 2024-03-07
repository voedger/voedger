# Voedger Virtual Machine

## Motivation

- [VVM: Draft design](https://dev.heeus.io/launchpad/#!26720)
- [VVM: Design & Plan](https://dev.heeus.io/launchpad/#!26771)

## VVM Relations

```mermaid
flowchart TD

  %% Entities =================================


  Cluster{{Cluster}}:::H
  VVMNode:::H
  Router:::S

  app.sys.cluster[app.sys.cluster]:::S
  ws.cluster.Сluster[(ws.cluster.Сluster)]:::H
  ws.cluster.VirtualMachine[(ws.cluster.VirtualMachine)]:::H

  VVM:::S


  %% Relations =================================

  Cluster --- app.sys.cluster
  Cluster --x Router
  Cluster --x VVMNode
  VVMNode --x |1..8| VVM

  Router x-.-x VVM

  app.sys.cluster --- ws.cluster.Сluster
  ws.cluster.Сluster --x ws.cluster.VirtualMachine

  ws.cluster.VirtualMachine -.- VVM

  classDef G fill:#FFFFFF,stroke:#000000, stroke-width:1px, stroke-dasharray: 5 5
  classDef B fill:#FFFFB5,color:#333
  classDef S fill:#B5FFFF
  classDef H fill:#C9E7B7
```


## VVM Dataflow

```mermaid
flowchart TD

  %% Entities =================================

  Router:::S

  ws.cluster.Сluster[(ws.cluster.Сluster)]:::H
  ws.cluster.VirtualMachine[(ws.cluster.VirtualMachine)]:::H

  VVM:::S


  %% Relations =================================

  Router <-.->|Request-Response| VVM
  
  ws.cluster.VirtualMachine <-.-> |AppPartition SP - PV| VVM

  ws.cluster.Сluster -.-> |AppImage, Schema| VVM

  classDef G fill:#FFFFFF,stroke:#000000, stroke-width:1px, stroke-dasharray: 5 5
  classDef B fill:#FFFFB5,color:#333
  classDef S fill:#B5FFFF
  classDef H fill:#C9E7B7
```

## apppartsctl.ControlLoop

```mermaid
flowchart TD

  %% Entities =================================

  ws.cluster.VirtualMachine[(ws.cluster.VirtualMachine)]:::H
  ws.Cluster[(ws.Cluster)]:::H
  cdoc.cluster.AppPartition:::H

  VVM:::G
  subgraph VVM
    ControlLoop_AppPartition:::G
    subgraph ControlLoop_AppPartition["apppartsctl.ControlLoop"]
      AppPartition_SPReader:::S
      DownloadImage:::S
      ParseAppSchemaFiles:::S
      DownloadImage --> ParseAppSchemaFiles
    end
    AppSchemaFiles[(folder.AppSchemaFiles)]:::H
    AppDef[(AppDef)]:::S
    AppPartitions[(AppPartitions)]:::S
    AppPartition:::S
      Cache([Cache]):::S
      AppSchema(["AppDef"]):::S
      Version([Version]):::S
    ExtEngine:::S
      
    IAppStructsProvider:::S
    Processors[[Processors]]:::S


    ExtensionEngineFactories["iextengine.IExtensionEngineFactories"]:::S

  end


  %% Relations =================================

  ws.cluster.VirtualMachine --- cdoc.cluster.AppPartition
  cdoc.cluster.AppPartition -.-> AppPartition_SPReader
  AppPartition_SPReader --> DownloadImage

  ws.Cluster -.-> |BLOBs| DownloadImage
  DownloadImage -.-> |*.sql etc.| AppSchemaFiles
  AppSchemaFiles -.-> ParseAppSchemaFiles
  ParseAppSchemaFiles -.-> AppDef

  AppPartitions --x AppPartition
  AppPartition --- Cache
  AppPartition --- AppSchema
  AppPartition --- Version
  AppPartition --x ExtEngine

  AppPartitions -.-> IAppStructsProvider
  IAppStructsProvider -.-> Processors
  ExtensionEngineFactories -.-> AppPartitions
  AppDef -.-> AppPartitions
  

  classDef G fill:#FFFFFF,stroke:#000000, stroke-width:1px, stroke-dasharray: 5 5
  classDef B fill:#FFFFB5,color:#333
  classDef S fill:#B5FFFF
  classDef H fill:#C9E7B7
```
