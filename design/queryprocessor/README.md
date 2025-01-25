# Query Processor
A component used to query data and return it to client

# Context
```mermaid
flowchart
bus(Bus):::S
bus2(Bus):::S
qpMessage>QP Message]:::G
qp[/Query Processor/]:::S
vvm[[vvm]]:::S
appPartition[[App Partition]]:::S
rp[Rows Processor]:::S
objects[Object]:::S
queryData[Query Data]:::G
state[State]:::H

%% Relations ======================
vvm --x |has many|qp
vvm --x |has many|appPartition
bus --x |provides|qpMessage
qpMessage -.-> |handled by|qp
appPartition -.-> |borrowed by|qp
qp --> |builds|rp
qp --> |builds|state
state --> |used to|queryData
qp -.- |invokes|queryData
queryData -.-x |outputs|objects
objects -.-> |pushed to|rp
rp -.-> |"sends result(s) to"|bus2

%% Styles ====================
classDef B fill:#FFFFB5,color:#333
classDef S fill:#B5FFFF,color:#333
classDef H fill:#C9E7B7,color:#333
classDef G fill:#ffffff15, stroke:#999, stroke-width:2px, stroke-dasharray: 5 5
```

# Components
## Query Processor v1 (/api/)
### Principles
- Only reads from query functions
- handled with HTTP POST

### QP Message & Query data
```mermaid
flowchart
queryData[Query Data]:::G
appPartition[[App Partition]]:::S
state[State]:::H
qpMessage>QP Message]:::S
rp[Rows Processor]:::S

subgraph queryData
  query[Query extension]:::S
  intents["intent sys.Result"]:::S
  objects[Object]:::S
end

queryData --> |use borrowed|appPartition
appPartition --> |to invoke|query
state --> |used to invoke|query
qpMessage --> |has|body:::S
qpMessage --> |has|QName:::S
QName --> |used to invoke|query
body -.-> |parsed to|queryParams[QueryParams]:::S
body -.-> |parsed to|queryArg[ArgumentObject]:::S
queryArg -.-> |used to build|state
queryParams -.-> |used to build|rp
objects -.-> |pushed to|rp
query -.-x intents:::S
intents -.-> |used to build|objects


%% Styles ====================
classDef B fill:#FFFFB5,color:#333
classDef S fill:#B5FFFF,color:#333
classDef H fill:#C9E7B7,color:#333
classDef G fill:#ffffff15, stroke:#999, stroke-width:2px, stroke-dasharray: 5 5
```

### Rows processor
```mermaid
flowchart
rp[Rows Processor]:::S
asyncPipeline:::G
subgraph asyncPipeline[Async Pipeline]
    resultFields:::S
    enrichment:::S
    filter:::S
    order:::S
    counter:::S
    send:::S
end

rp --> |is a|asyncPipeline
resultFields --> enrichment
enrichment --> filter
filter --> order
order --> counter
counter --> send
%% Styles ====================
classDef B fill:#FFFFB5,color:#333
classDef S fill:#B5FFFF,color:#333
classDef H fill:#C9E7B7,color:#333
classDef G fill:#ffffff15, stroke:#999, stroke-width:2px, stroke-dasharray: 5 5
```

## Query Processor v2 (/api/v2/)
### Principles
- Reads from query functions, documents, views
- handled with HTTP GET
- supports query constraints in ParseAPI syntax
- see also: https://github.com/voedger/voedger/issues/1162 

### QP Message & Query data
```mermaid
flowchart
qpMessage>QP Message]:::S
rp[Rows Processor]:::S
queryData[Query Data]:::G
subgraph queryData
  entity[Entity]:::A
  doc[Document]:::S
  views[View]:::S
  extension[Query Extension]:::S
  intents["intent sys.Result"]:::S
  entity --> |is either|extension
  entity --> |is either|doc
  entity -.-> |is either|views
  extension -.-x |generates|intents:::S
  objects[Object]:::S
end

%% Relations ======================
qpMessage --> |has|QName:::S
qpMessage --> |has|Type:::S
qpMessage --> |may have|DocID:::S
qpMessage --> |may have|QueryParams:::S
objects -.-> |pushed to|rp

QueryParams --> |may have|Constraints:::S
QueryParams --> |may have|Argument:::S

intents -.-> |used to build|objects
views -.-> |QP reads rows and <br>converts to|objects
doc -.-> |QP reads and <br>converts to|objects

QName -.-> |used to find|entity
Type -.-> |used to find|entity
DocID -.-> |used to find|doc
Argument -..-> |used by|extension
Constraints -.-> |used to build|rp

%% Styles ====================
classDef B fill:#FFFFB5,color:#333
classDef S fill:#B5FFFF,color:#333
classDef H fill:#C9E7B7,color:#333
classDef G fill:#ffffff15, stroke:#999, stroke-width:2px, stroke-dasharray: 5 5
classDef A fill:transparent, stroke-width:1px, stroke-dasharray: 4 3
```
### Rows Processor
```mermaid
flowchart
rp[Rows Processor]:::S
asyncPipeline:::G
subgraph asyncPipeline[Async Pipeline]
    select:::S
    include:::S
    filter:::S
    order:::S
    limit:::S
    send:::S
end

rp --> |is a|asyncPipeline
select --> include
include --> filter
filter --> order
order --> limit
limit --> send
%% Styles ====================
classDef B fill:#FFFFB5,color:#333
classDef S fill:#B5FFFF,color:#333
classDef H fill:#C9E7B7,color:#333
classDef G fill:#ffffff15, stroke:#999, stroke-width:2px, stroke-dasharray: 5 5
```

