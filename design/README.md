## Heeus concepts

- [Federation](#federation)
- [Cluster](#cluster)
- [AppPartition](#apppartition)
- [Event Sourcing & CQRS](#event-sourcing--cqrs)
- [Repository & Application Schema](#repository--application-schema)
- [Package Schema](#package-schema)
- [Application Image](#application-image)
- [Extensions](#extensions)
- [Bus](#bus)
- [Edge Computing](#edge-computing)


## Heeus products

- [Editions](#editions)
- [Community Edition (CE)](#community-edition-ce)
- [Standart Edition (SE)](se/README.md)
- [Enterprise Edition (EE)](#enterprise-edition-ee)

## DevOps

- [Building](building)

## Detailed design

Features
- [Orchestration](orchestration/README.md)
- [Workspaces](workspaces/README.md)
- [HTTPS+ACME](https-acme/README.md)
- [Edge Computing](edge/README.md)

Non-Functional Reqiurements (Quality Attributes, Quality Requirements, Qualities)
- [Consistency](consistency)
- Security
  - [Authentication and Authorization (AuthNZ)](authnz)
  - Encryption: [HTTPS + ACME](https-acme)
- Maintainability, Perfomance, Portability, Usability ([ISO 25010](https://iso25000.com/index.php/en/iso-25000-standards/iso-25010), System and software quality models)

System Design

- [Bus](https://github.com/heeus/core/tree/main/ibus)
- [State](state/README.md)
- [Command Processor](commandprocessor/README.md)
- [Query Processor](queryprocessor/README.md)
- [Projectors](projectors/README.md)
- [Storage](storage/README.md)

## Notation

Notation based on:
- [ArchiMate](https://en.wikipedia.org/wiki/ArchiMate) (/ˈɑːrkɪmeɪt/ AR-ki-mayt; originally from Architecture-Animate), open and independent enterprise architecture modeling language
- [Entity–relationship model](https://en.wikipedia.org/wiki/Entity%E2%80%93relationship_model), describes interrelated things of interest in a specific domain of knowledge

```mermaid
flowchart TD

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
  
  User:::B
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
  BusinessProcess --- |assigned to| User["Human actor (e.g. User)"]

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

## Federation

```mermaid
    erDiagram
    Federation ||--|| MainCluster : has
    MainCluster  ||..||  Cluster: "is"
    MainCluster  ||--||  sys_registry: "has deployed Application"

    Federation ||--o{ WorkerCluster : has
    WorkerCluster  ||..||  Cluster: "is"
    Cluster ||--o{ Application : "has deployed"
```

## Cluster

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

    VVM ||--o{ AppPartition : "executes q/c/a for"

```

## AppPartition

> AppPartition is a scheduling unit

```mermaid
    erDiagram

    AppPartition ||--|| AppPartitionStorage : "has"
    AppPartition ||--|| CommandProcessor : "has"
    AppPartition ||--|{ Actualizer : "has"
    Actualizer ||--|{ Projector : "manages"


    AppPartitionStorage ||--|| AppStorage : "to work with"
    AppStorage ||--|| AppPartitionCache : "through"


    AppPartitionStorage ||--|{  QueryProcessor: "used by"
    AppPartitionStorage ||--||  CommandProcessor: "used by"
    AppPartitionStorage ||--|{  Projector: "used by"

    AppPartition ||--|{ QueryProcessor : "has"

    AppStorage ||--|{ Workspace: has
    Workspace ||--|{ InternalProjection: "keeps"
    AppStorage ||--|| PLog: has
    PLog ||--|{ PLogPartition : has

    PLogPartition ||--|{ Actualizer: "is read by"

    Projector ||--|{ Projection: "prepares write intents for"
    Projection ||--|| InternalProjection: "can be"
    Projection ||--|| ExternalProjection: "can be"

    ExternalProjection ||--|| Email: like


    InternalProjection ||--|| WLog: "can be"
    InternalProjection ||--|| Table: "can be"
    InternalProjection ||--|| View: "can be"


    CommandProcessor ||--|| PLogPartition: "writes Event to"

    QueryProcessor ||--|{ QueryFunction: "reads from"
    QueryFunction ||--|| Projection: "can be"
```

## Repository & Application Schema

```mermaid
    erDiagram
    Repository ||--|{ Application: defines
    Application ||--|| ApplicationSchema: "defines"
    ApplicationSchema ||--o{ PackageSchema: "has"
    Application ||--o{ Package: "has"
    Package ||--|| PackageSchema : "defines"
    Package ||--|{ SchemaFile : "has *.heeus"
    PackageSchema ||--|{ SchemaFile: "defined by"
```

## Package Schema

```mermaid
    erDiagram
    PackageSchema ||--o{ Def: "has"
    Def ||--|| TableDef: "can be"
    Def ||--|| ViewDef: "can be"
    Def ||--|| ExtensionDef: "can be"
    ExtensionDef ||--|| FunctionDef : "can be"
    FunctionDef ||--|| CommandFunctionDef: "can be"
    FunctionDef ||--|| QueryFunctionDef: "can be"
    ExtensionDef ||--|| ValidatorDef: "can be"
    ExtensionDef |{--|| ExtEngineKind: "has"
    ExtEngineKind ||..|| ExtEngineKind_WASM: "can be"
    ExtEngineKind ||..|| ExtEngineKind_BuiltIn: "can be"
```

## Application Image

```mermaid
    erDiagram
    ApplicationImage ||--|| ApplicationSchema: "has"
    ApplicationImage ||--o{ Resource: "has"
    Resource ||--|| Image: "can be"
    Resource ||--|| ExtensionsPackage: "can be"
    ExtensionsPackage ||--|| ExtEngineKind: "has a property"
```

## Bus

### Bus principles

- Limit number of concurrent requests: maxNumOfConcurrentRequests
  - Example: million of http connections but 1000 concurrent requests
  - "ibus.ErrBusUnavailable" (503) is returned if the number of concurrent requests is exceeded
- Sender and Receiver both respect timeouts: readWriteTimeout
  - E.g. 5 seconds, by (weak) analogy with FoundationDB, Long-running read/write

### Bus Nodes
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

### Some known Bus Nodes
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
### See also
- [Bus detailed design](https://github.com/heeus/core/tree/main/ibus)


## Event Sourcing & CQRS

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
```

## Extensions

### Principles

- Extensions extend Core functionality
  - Расширения расширяют функциональность ядра
- Extensions can be loaded/updated/unloaded dynamically
  - But BuiltIn Extensions

### Extensions Site
- Extensions Site: Сайт расширений

```mermaid
    erDiagram
    ExtensionsSite ||--|{ ExtensionPoint : "has"
    ExtensionPoint ||--|{ Extension : "has"
```
Extension Site examples:
```mermaid
    erDiagram
    ExtensionsSite ||--|| CommandProcessor : "can be e.g."
    CommandProcessor ||--|| CommandFunctions : "has"
    CommandProcessor ||--|| Validators : "has"
    ExtensionsSite ||--|| QueryProcessor : "can be e.g."
    QueryProcessor ||--|| QueryFunctions : "has"
    CommandFunctions ||--|| ExtensionPoint : "is"
    QueryFunctions ||--|| ExtensionPoint : "is"
    Validators ||--|| ExtensionPoint : "is"
```

### Extension Engines
- Extension Engine: Механизм расширения

```mermaid
    erDiagram
    ExtensionsSite ||--|{ ExtensionPoint : "has"
    ExtensionsSite ||--|{ ExtensionEngine : "has"

    ExtensionEngine ||--|| Limits : "has"
    ExtensionEngine ||..|| ExtensionEngineFactory : "created by"
    ExtensionEngine ||..|{ Extension: "executes"
    ExtensionEngine |{..|| ExtensionEngineFactory: "created by"
    ExtensionEngine ||--|| ExtEngineKind: has

    Limits ||..|| MemoryLimit: "can be"
    Limits ||..|| GasLimit: "can be"

    ExtensionEngineFactory ||..|| ExtEngineKind : "one per"
    ExtensionPoint ||--|{ Extension : "has"

    Extension |{..|| ExtEngineKind: has

    ExtEngineKind ||..|| ExtEngineKind_WASM: "can be"
    ExtEngineKind ||..|| ExtEngineKind_BuiltIn: "can be"
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

## Edge Computing

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
### Detailed design

- [edge/README.md](edge/README.md)

