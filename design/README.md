## Top-level sections

- [Operations Concepts](#operations-concepts)
- [Development Concepts](#development-concepts)
- [Editions](#editions)
- [Detailed design](#detailed-design)


## Notation

Notation based on:
- [ArchiMate](https://en.wikipedia.org/wiki/ArchiMate) (/ˈɑːrkɪmeɪt/ AR-ki-mayt; originally from Architecture-Animate), open and independent enterprise architecture modeling language
- [Entity–relationship model](https://en.wikipedia.org/wiki/Entity%E2%80%93relationship_model), describes interrelated things of interest in a specific domain of knowledge

```mermaid
graph TD

  %% Entities =================================

  Infrastructure{{Infrastructure}}:::H
  Database[(Database)]:::H
  Table:::H
  Field([Field]):::H
  Data:::H
  DataField1([Field1]):::H
  DataField2([Field2]):::H

  ProductLine[[Product Line]]:::S
    ProductLine --- Product1[Product 1]:::S
    ProductLine --- Product2[Product 2]:::S

  SoftwareComponents:::G
  subgraph SoftwareComponents[Group of elements]
    SoftwareComponent[Software Component 1]:::S
    SoftwareComponent2[Software Component 2]:::S
  end
  
  SoftwareService([Software Service]):::S  
  
  User["Human actor (e.g. User)"]:::B
  Company{{"Non-human actor (e.g. Company)"}}:::B
  BusinessProcess(Business Process):::B


  %% Relations =================================

  Infrastructure ---|runs| SoftwareComponents

  SoftwareComponent --- |provides| SoftwareService

  Infrastructure --x Database
  Database ---x|has few| Table
  Table ---x|has few| Field

  SoftwareService -.->|generates| Data

  SoftwareService --- |used by| BusinessProcess
  BusinessProcess --- |assigned to| User

  Data -.->|used by| SoftwareComponent2

  Data --- DataField1
  Data --- DataField2

  Product2 --- |used by| Company
  Company --- |has| BusinessProcess

  classDef G fill:#FFFFFF,stroke:#000000, stroke-width:1px, stroke-dasharray: 5 5
  classDef B fill:#FFFFB5,color:#333
  classDef S fill:#B5FFFF
  classDef H fill:#C9E7B7
```

## Roles & Services

```mermaid
  graph TD

  %% Entities ====================

  Admin[Admin]:::B
  Developer[Developer]:::B
  User[User]:::B

  mod.exttinygo:::S
  cmd.vpm:::S
  cmd.voedger:::S
  cmd.ctool:::S  
  grafana[extsoft.grafana]:::G
  alertmanager[extsoft.alertmanager]:::G
  scylla[extsoft.scylla]:::G

  DBMS(DBMS):::S
  Development(Development):::S

  ClusterMgmt(Cluster Management):::S
  AppMgmt(Application Management):::S
  Monitoring(Monitoring):::S
  Alerting(Alerting):::S
  RunApp(Application Execution):::S
  APIGateway(API Gateway):::S

  %% Relations ====================

  scylla --> DBMS
  DBMS --> cmd.voedger

  mod.exttinygo --> Development
  cmd.vpm --> Development
  Development --> Developer

  cmd.ctool --> ClusterMgmt
  ClusterMgmt --> Admin


  grafana --> Monitoring
  alertmanager --> Alerting
  Monitoring --> Admin
  Alerting --> Admin

  cmd.voedger --> AppMgmt
  AppMgmt --> Admin

  cmd.voedger --> RunApp
  RunApp -.- APIGateway

  cmd.voedger --> APIGateway
  APIGateway --> User


  classDef B fill:#FFFFB5
  classDef S fill:#B5FFFF
  classDef H fill:#C9E7B7
  classDef G fill:#FFFFFF,stroke:#000000, stroke-width:1px, stroke-dasharray: 5 5
```


## Operations Concepts

### Federation

```mermaid
    erDiagram
    Federation ||--|| MainCluster : has
    MainCluster  ||..||  Cluster: "is"
    MainCluster  ||--||  sys_registry: "has deployed Application"

    Federation ||--o{ WorkerCluster : has
    WorkerCluster  ||..||  Cluster: "is"
    Cluster ||--o{ Application : "has deployed"
```

### Cluster

```mermaid
    erDiagram
    Cluster ||--|{ Router : has
    Cluster ||--|{ VVM : has
    Cluster ||--o{ Application : "has deployed"
    Cluster ||--|| ClusterStorage : has
    ClusterStorage ||--|{ AppStorage : has

    Application ||--|{ AppPartition : "partitioned into"
    Application ||--|| AppStorage : "uses"
    AppPartition ||--|{ Workspace : "uses"
    AppPartition ||--|| PLogPartition : "uses"

    AppStorage ||--|{ Workspace: has
    AppStorage ||--|| PLog: has

    AppStorage ||--|| istorage: "has API"
    istorage ||--|| istoragecas : "implemented e.g. by"
    istorage ||--|| istoragemem : "implemented e.g. by"

    Workspace ||--|{ InternalProjection: keeps
    PLog ||--|{ PLogPartition : has

    VVM ||--o{ AppPartition : "assigned by Scheduler to"
```
### VVM: Execute assigned AppPartition

| Old term      | New term|
| ----------- | ----------- |
| IAppStructsProvider      | IAppPartitions       |
| IAppStructs   | IAppPartition      |

#### Processors



```mermaid
    erDiagram
    
    %% Entities

    %% Relations

    VVM ||--|{ Processor : "has"

    Processor ||--|| CommandProcessor : "can be"
    Processor ||--|| QueryProcessor : "can be"
    Processor ||--|| Actualizer : "can be"   

    CommandProcessor ||..|| "Invoke(extName, extIO)": "executes Extension using"
    QueryProcessor ||..|| "Invoke(extName, extIO)": "executes Extension using"
    Actualizer ||..|| "Invoke(extName, extIO)": "executes Extension using"

    "Invoke(extName, extIO)" ||..|| IAppPartition: "method of"


    IAppPartition ||..|| IAppPartitions: "borrowed from"
```


#### Borrow IAppPartition

```go
type IAppPartitions interface {
    ...
    Borrow(qpp AppQName, part PartitionID, procKind ProcessorKind) (IAppPartition, error)
    ...
}
```

```mermaid
    erDiagram

    IAppPartitions ||--|{ appRT : "has"

    appRT ||--|{ appPartitionRT : "has"

    appRT ||--|| latestVersion : "has"
    appPartitionRT ||--|| partitionCache : "has"

    latestVersion ||--|| AppDef : "has"
    latestVersion  ||--|{ commandsExEnginePool : "has one per EngineKind"
    latestVersion  ||--|{ queryExEnginePool : "has one per EngineKind"
    latestVersion  ||--|{ projectionExEnginePool : "has one per EngineKind"


    AppDef ||--|{ appdef_IPackage : "has"
    appdef_IPackage ||..|{ appdef_IEngine: "extensions instantiated by"

    appdef_IEngine ||..|| "IAppPartitions_Borrow()": "copied by ref by"
    
    commandsExEnginePool ||..|{ "appdef_IEngine": "provides pool of"
    queryExEnginePool ||..|{ "appdef_IEngine": "provides pool of"
    projectionExEnginePool ||..|{ "appdef_IEngine": "provides pool of"
    partitionCache ||..|| "IAppPartitions_Borrow()": "copied by ref by"

    "IAppPartitions_Borrow()" ||..|| "IAppPartition": "returns"

    IAppPartition ||--|{ "Invoke()" : "may invoke extensions with method"
```

#### Construct Pools of Extension Engines
interfaces:
```go
// Builtin engines factory
type IExtensionEngineFactories interface {
  QueryFactory(appdef.ExtensionEngineKind) IExtensionEngineFactory
}

IExtensionEngineFactory {
  // LocalPath is a path package data can be got from
  // packageNameToLocalPath is not used for ExtensionEngineKind_BuiltIn
  New(packageNameToLocalPath map[string]string, numEngines int) []IExtensionEngine
}

type IExtensionEngineFactories map[appdef.ExtensionEngineKind]IExtensionEngineFactory

type ExtQName struct {
    PackageName string // Fully qualified package name
    ExtName string
}

type IExtensionEngine interface {
    Invoke(ctx context.Context, extName ExtQName, io IExtensionIO) (err error)
}

type BuiltInExtFunc func(io ExtensionIO) error
type BuiltInExtFuncs map[ExtQName]BuiltInExtFunc // Provided to construct factory of engines
```

An instance of IExtensionEngineFactories is provided to apppartsctl.New().

#### Execute Extentions
```go
type IAppPartition interface {
    // Processor constructs and provides ExtensionIO
    Invoke(extensionName QName, io IExtensionIO) (err error)
}
```

### Event Sourcing & CQRS

**Event Sourcing**

- Event Sourcing is a design pattern where all changes to the application state are stored as a sequence of events

> Event Sourcing ensures that all changes to application state are stored as a sequence of events.
>
> [Martin Fowler: Event Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html)
> <img src="https://martinfowler.com/mf.jpg" alt="drawing" width="60"/>

- Storing a log of all events provides an "natural" **audit trail** (журнал аудита, контрольный журнал) ([link](https://arkwright.github.io/event-sourcing.html#audit-trail))
- Partitioning PLog into PLogPartition provides horizontal **scalability**

**CQRS**

- CQRS (Command and Query Responsibility Segregation) is a design pattern where different data models are used for writes (by Commands) and reads (by Queries)
- Implementing CQRS in your application can maximize its **performance, scalability, and security** ([CQRS pattern, learn.microsoft.com](https://learn.microsoft.com/en-us/azure/architecture/patterns/cqrs))

```mermaid
    erDiagram
    Client ||--|| CommandProcessor : "1. sends Command through HTTPProcessor to"
    Client ||--|| QueryProcessor : "2. sends Query through HTTPProcessor to"
    CommandProcessor ||--|| WriteModel : "writes Event to"
    WriteModel ||--|| PLogPartition : "implemented by"
    Actualizer ||--|{ Projector : "manages"
    PLogPartition ||--|{ Actualizer: "is read by"
    Projector ||--|{ Projection: "prepares write intents for"
    QueryProcessor ||--|{ ReadModel: "reads from"
    ReadModel ||--|| Projection: "implemented by"

    Projection ||--|| InternalProjection: "can be"
    Projection ||--|| ExternalProjection: "can be"

    ExternalProjection ||--|| Email: like


    InternalProjection ||--|| WLog: "can be"
    InternalProjection ||--|| Table: "can be"
    InternalProjection ||--|| View: "can be"    
    
    Workspace ||--|{ InternalProjection: keeps    
```

### Extensions

#### Principles

- Extensions extend Core functionality
  - Расширения расширяют функциональность ядра
- Extensions can be loaded/updated/unloaded dynamically
  - But BuiltIn Extensions

#### Extension Engines
- Extension Engine: Движок расширения
- ??? Does DockerExtensionEngine need memory?

```mermaid
    erDiagram
    ExtensionEngine {
      Memory MB
      Invoke() func()
      Kind ExtEngineKind  "e.g. WASM, BuiltIn, Container"
    }
    ExtensionEngine ||..|| ExtensionEngineFactory : "created by"
    ExtensionEngine ||--|| Kind: has
    ExtensionEngineFactory ||..|| Kind : "one per"

```

### Bus

#### Bus principles

- Limit number of concurrent requests: maxNumOfConcurrentRequests
  - Example: million of http connections but 1000 concurrent requests
  - "ibus.ErrBusUnavailable" (503) is returned if the number of concurrent requests is exceeded
- Sender and Receiver both respect timeouts: readWriteTimeout
  - E.g. 5 seconds, by (weak) analogy with FoundationDB, Long-running read/write

#### Bus Nodes
```mermaid
    erDiagram
    Bus ||--o{ BusNode : "connects"
    BusNode ||--o| Sender : "can be"
    BusNode ||--o| Receiver : "can be"
    Receiver ||--|| Address : "has"
    Address{
        owner string
        app string
        partition int
        part string "e.g. 'q' or 'c'"
    }
```

#### Some known Bus Nodes
```mermaid
    erDiagram
    AppPartitionController||--o{ AppPartition : "sends to"
    HTTPProcessorController||--|| HTTPProcessor : "sends to"
    HTTPProcessor ||--|{ QueryProcessor : "sends to <owner>/<app>/<partition>/q"
    HTTPProcessor ||--|{ CommandProcessor : "sends to <owner>/<app>/<partition>/c"
    HTTPProcessor ||--|| FederationGateway : "sends to"
    QueryProcessor ||--|{ Gateway : "sends to"
    Actualizer ||--|{  Gateway : "sends to"

    Gateway ||--|| FederationGateway: "can be"
    Gateway ||--|| HTTPGateway: "can be"
    Gateway ||--|| MailerGateway: "can be"
```
#### See also
- [Bus detailed design](https://github.com/heeus/core/tree/main/ibus)
- [Bus](#bus)

### Edge Computing

[redhat.com An Architect's guide to edge computing essentials](https://www.redhat.com/architect/edge-computing-essentials)

- Edge computing (периферийные вычисления, граничные вычисления) is a distributed computing pattern (модель распределенных вычислений). Computing assets on a very wide network are organized so that certain computational and storage devices that are essential to a particular task are positioned close to the physical location where a task is being executed
- Edge computing is definitely a thing in today's technical landscape. The market size for edge computing products and services has more than doubled since 2017. And, according to the statistics site, Statista, it's projected to explode by 2025. (See Figure 1, below)

[tadviser.ru: https://www.tadviser.ru/](https://www.tadviser.ru/index.php/%D0%A1%D1%82%D0%B0%D1%82%D1%8C%D1%8F:%D0%9F%D0%B5%D1%80%D0%B8%D1%84%D0%B5%D1%80%D0%B8%D0%B9%D0%BD%D1%8B%D0%B5_%D0%B2%D1%8B%D1%87%D0%B8%D1%81%D0%BB%D0%B5%D0%BD%D0%B8%D1%8F_(Edge_computing%))

- Выступая на конференции Open Networking Summit в Бельгии в сентябре 2019 года руководитель сетевых проектов Linux Foundation Арпит Джошипура (Arpit Joshipura) заявил, что периферийные вычисления станут важнее облачных к 2025 году.

```mermaid
    flowchart TB

    %% Entities ====================

    Node:::G
    Cloud:::G

    subgraph Node["Edge Node"]


        edger[edger]:::S
        DockerStack[Docker stack]:::S
        CE[CE]:::S
        CheckedOutWorkspace[(Checked-out Workspace)]:::H

        edger --- |controls| DockerStack
        DockerStack --- |runs| CE

        CE --x |works with| CheckedOutWorkspace
    end

    subgraph Cloud
        OriginalWorskpace[(Original Worskpace)]:::H
    end

    CheckedOutWorkspace  ---> |replicated to| OriginalWorskpace

    classDef B fill:#FFFFB5
    classDef S fill:#B5FFFF
    classDef H fill:#C9E7B7
    classDef G fill:#FFFFFF,stroke:#000000, stroke-width:1px, stroke-dasharray: 5 5
```
- [Detailed design](edge/README.md)


## Development Concepts

### Schemas

```mermaid
  erDiagram
  
  %% Entities

  Repository
  Folder
  PackageSchema 
  SchemaFile
  ApplicationSchema
  ApplicationStmt

  %% Relationships
  Repository ||--|{ Folder : "has"
  Folder ||--|{ SchemaFile : "has"
  Folder ||..|| PackageSchema :"corresponds to exactly one"
  SchemaFile |{..|| PackageSchema : "used to build"
  PackageSchema ||--o| ApplicationStmt : "can have"
  PackageSchema |{..|| ApplicationSchema : "used to build"
  ApplicationSchema ||..|| ApplicationStmt : "defined by exactly one"
```

### Example of ApplicationStmt
- **Package name**: last part of the package path or alias from the IMPORT statement

```sql
IMPORT SCHEMA 'github.com/untillpro/untill' AS air;
IMPORT SCHEMA 'github.com/untillpro/airsbp';

-- Only one APPLICATION statement allowed per package and per application
APPLICATION bp3 (
  -- "sys" is always used in any application
  USE air;  -- name or alias. This actually identifies package in QNames of the app
  USE airsbp; 
)
```

## Editions

|             | CE          |SE          |Enterprise  |
| ----------- | ----------- |----------- |----------- |
| Federation  | Yes         |Yes         |Yes         |
| Router      | 1           |1           |Many        |
| VM          | 1           |1           |Many        |
| HA          | No          |Yes         |Yes         |
| Scalability | No          |No          |Yes         |

### Community Edition (CE)

Principles

- Node can run other software, unlike the SE (all nodes must be clean)
- Docker
  - So we won't support FreeBSD as a host OS
  - Reason: We beleive (paa) that FreeBSD is for things like routers
- Scylla as a ClusterStorage  
  - Reason: We do not want to learn how to operate bbolt
- Scylla is also containerized
  - Reason: [The cost of containerization is within 10%](https://scylladb.medium.com/the-cost-of-containerization-for-your-scylla-a24559d17d01), so ok

```mermaid
    erDiagram
    CECluster ||--|| Node : "always has one"
    Node ||--|| CEDockerStack : "runs"
    CEDockerStack ||--|| voedger : "contains"
    CEDockerStack ||--|| prometheus : "contains"
    CEDockerStack ||--|| graphana : "contains"
    CEDockerStack ||--|| scylla : "contains"
    voedger ||..|| scylla : "uses as ClusterStorage"
```

### Standart Edition (SE)

ref. [se/README.md](se/README.md)

### Enterprise Edition (EE)

```mermaid
    erDiagram
    EECluster ||--|{ RouterNode : "has 2+ nodes with role Router"
    EECluster ||--|{ VVMNode : "has 2+ nodes with role VVM"
    EECluster ||--|{ DBNode : "has 3+ nodes with role DBNode"
    EECluster ||--|| ClusterStorage : "has"
    ClusterStorage ||--|| DBMS: has

    RouterNode ||--|| Node : "is"
    VVMNode ||--|| Node : "is"
    DBNode ||--|| Node : "is"
    DBNode |{--|| ClusterStorage : "used by"
    DBNode ||--|| DBMS : "runs"

    Node ||--|| agent_exe : "has process"

    VVMNode ||--|| VVM_exe : "has process"
    RouterNode ||--|| router_exe : "has process"

    agent_exe ||--|| router_exe : "controls"
    agent_exe ||--|| VVM_exe : "controls"
    agent_exe ||--|| DBMS : "controls"

    DBMS ||--|| Cassandra : "can be"
    DBMS ||--|| Scylla : "can be"
    DBMS ||--|| FoundationDB : "can be"
```

## Detailed design

Functional Design

- [Orchestration](orchestration/README.md)
- [Workspaces](workspaces/README.md)
- [Edge Computing](edge/README.md)

Non-Functional Reqiurements, aka Quality Attributes, Quality Requirements, Qualities

- [Consistency](consistency)
- Security
  - Encryption: [HTTPS + ACME](https-acme)
  - [Authentication and Authorization (AuthNZ)](authnz)
- TBD: Maintainability, Perfomance, Portability, Usability ([ISO 25010](https://iso25000.com/index.php/en/iso-25000-standards/iso-25010), System and software quality models)

Technical Design

- [Bus](https://github.com/heeus/core/tree/main/ibus)
- [State](state/README.md)
- [Command Processor](commandprocessor/README.md)
- [Query Processor](queryprocessor/README.md)
- [Projectors](projectors/README.md)
- [Storage](storage/README.md)

## Misc

DevOps

- [Building](building)

Previous incompatible versions

- [Prior 2023-09-13](https://github.com/voedger/voedger/blob/7f9ff095d66e390028abe9037806dcd28bde5d9e/design/README.md)
