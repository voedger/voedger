# iextsee

see: State Storage Extension Engines

- [server/storage-extensions.md](https://github.com/voedger/voedger-internals/blob/main/server/storage-extensions.md)

## Use cases

- Handle communication using proprietary SCADA protocol
- Handle communication using proprietary SCADA protocol

## Key components

```mermaid
erDiagram
    %% Entities

    %% Relationships

    VVM ||--|{ DeployedApplication : "1+"
    VVM ||--|{ SSEInstance : "1+"
    SSEInstance ||--||SSEType : ""
    SSEType ||--|| "iextsee.ISSEVvmFactory" : ""
    "iextsee.ISSEVvmFactory" ||--|{ "iextsee.ISSEAppVerFactory" : "creates"

    DeployedApplication ||--|{ Version : "1+"
    Version ||--|| "iextsee.ISSEAppVerFactory": ""


    "iextsee.ISSEVvmFactory" ||--|| "iextsee.Config" : "has"
    "iextsee.Config" ||--|| "iextsee.ISSELogger" : "has"

    "iextsee.ISSEAppVerFactory" ||--|{ "iextsee.IPartitionSSEFactory" : "creates"
    "iextsee.IPartitionSSEFactory" ||--|{ "iextsee.ISSE" : "creates"


    DeployedApplication ||--|{ AppPartition : "1+"

    AppPartition ||--|{ "iextsee.IPartitionSSEFactory" : "1 active"

    AppPartition ||--o{ Workspace : "0+"

    Workspace ||--o{ "State" : "0+"

    State ||--|| "iextsee.ISSE" : "1"

    QueryState ||--|| State : "is"
    CommandState  ||--|| State : "is"
```