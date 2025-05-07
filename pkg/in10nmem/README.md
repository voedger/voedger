# in10nmem: implementation of in10n interface

## Concepts

```mermaid
erDiagram

    Broker ||--o{ Channel : has
    Broker ||--o{ "Projection" : has
    Channel ||--|| "Watching routine" : has

    Channel ||..|{ Projection : "subscribed to few"

    Projection ||--|| "offset" : has

    "Watching routine" ||..|| "offset" : "is notified about changes of"
```

## Architecture

```mermaid
graph TD

%% Entities =================================
BrokerSystem:::G
subgraph BrokerSystem[Broker system]
  Broker["Broker"]:::S
  BrokerUpdateFn("Update()"):::S
  NotifierGoroutine["go notifier()"]:::S
  EventsChannel(["events chan event{}"]):::H
  
  ChannelComp["channel"]:::S
  WatchChannel["WatchChannel()"]:::S
  CChanStruct(["cchan chan struct{}"]):::H
  
  ProjectionComp["projection"]:::S
  OffsetField(["offset"]):::H
  
  Subscription["subscription"]:::S
end

%% Relations =================================
Broker --x |has many| ChannelComp
Broker --x |has many| ProjectionComp
Broker --- |has| BrokerUpdateFn
Broker --- |has| NotifierGoroutine
Broker --- |has| EventsChannel

ProjectionComp -.-x |has few subscribed| Subscription
ProjectionComp --- |has| OffsetField


ChannelComp --- |has| CChanStruct
ChannelComp --- |has| WatchChannel
ChannelComp --x |has many| Subscription

CChanStruct -.-> WatchChannel

BrokerUpdateFn -.-> |changes| OffsetField
BrokerUpdateFn -.-> EventsChannel

NotifierGoroutine -.-> CChanStruct
EventsChannel -.->   NotifierGoroutine

classDef B fill:#FFFFB5,color:#333
classDef S fill:#B5FFFF,color:#333
classDef H fill:#C9E7B7,color:#333
classDef G fill:#ffffff15, stroke:#999, stroke-width:2px, stroke-dasharray: 5 5
```

## History

- [Before removing the inv, v1 folders](https://github.com/voedger/voedger/blob/e79b37d3644a626f9cef03e17a5904c638e293b5/pkg/in10nmem/README.md)
