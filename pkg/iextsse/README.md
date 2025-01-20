# iextsee

Status: Frozen

see: State Storage Extension Engines

- [server/storage-extensions.md](https://github.com/voedger/voedger-internals/blob/main/server/storage-extensions.md)

## Motivation

- https://github.com/voedger/voedger/issues/2653

## Key components

```mermaid
erDiagram
    %% Entities


    AppPartition["apparts.appPartitionRT"]
    ISSEVvmFactory["iextsee.ISSEVvmFactory"]

    %% It is here for better layout
    ISSEVvmFactory ||--|| "iextsee.Config" : "receives"

    ISSEAppVerFactory["iextsee.ISSEAppVerFactory"]
    ISSEPartitionFactory["iextsee.ISSEPartitionFactory"]
    ISSEStateStorage["iextsee.ISSEStateStorage"]

    App["apparts.appRT"]

    "iextsee.Config" {
        Logger ISSELogger
    }
    
    ProjectorState

    %% Relationships

    VVM ||--|{ App : "1+"
    VVM ||--|{ SSEInstance : "1+"
    SSEInstance ||--||SSEType : "one per"
    SSEType ||--|| ISSEVvmFactory: ""

    ISSEVvmFactory ||--|{ ISSEAppVerFactory : "creates"
    ISSEAppVerFactory ||--|{ ISSEPartitionFactory : "creates"


    App ||--|| AppVersion : "1 current"
    App ||--o{ AppVersion : "0+ active"
    App ||--o{ AppPartition : ""

    AppPartition ||--o{ AppPartitionVersion : "0+ active"
    AppPartition ||--|| AppPartitionVersion : "1 current"

    AppVersion ||--|| ISSEAppVerFactory: ""


    ISSEPartitionFactory ||--o{ ISSEStateStorage : "creates"

    AppPartition ||--o{ Workspace : "0+"

    Workspace ||--o{ "State" : ""

    State ||--|| ISSEStateStorage : "1"

    QueryState ||--|| State : "is"
    CommandState  ||--|| State : "is"
    AppPartitionVersion ||--|| ISSEPartitionFactory : ""
    ProjectorState ||--|| State : is

```