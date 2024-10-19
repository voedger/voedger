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
    VVM ||--|{ SEEInstance : "1+"
    SEEInstance ||--||SEEType : ""
    SEEType ||--|| "iextsee.ISEEVvmFactory" : ""
    "iextsee.ISEEVvmFactory" ||--|{ "iextsee.ISEEAppVerFactory" : "creates"

    DeployedApplication ||--|{ Version : "1+"
    Version ||--|| "iextsee.ISEEAppVerFactory": ""


    "iextsee.ISEEVvmFactory" ||--|| "iextsee.Config" : "has"
    "iextsee.Config" ||--|| "iextsee.ISEELogger" : "has"

    "iextsee.ISEEAppVerFactory" ||--|{ "iextsee.IPartitionSEEFactory" : "creates"
    "iextsee.IPartitionSEEFactory" ||--|{ "iextsee.ISEE" : "creates"


    DeployedApplication ||--|{ AppPartition : "1+"

    AppPartition ||--|{ "iextsee.IPartitionSEEFactory" : "1 active"

    AppPartition ||--o{ Workspace : "0+"

    Workspace ||--o{ "State" : "0+"

    State ||--|| "iextsee.ISEE" : "1"

    QueryState ||--|| State : "is"
    CommandState  ||--|| State : "is"
```