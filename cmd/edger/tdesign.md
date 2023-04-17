# Technical Design

- [Node State](#node-state)
- [Goroutines](#goroutines)

# Node State

```mermaid
erDiagram
    Cloud ||--|{ EdgeNodeState : "has"

    EdgeNodeState ||--|| DesiredState: "has"
    EdgeNodeState ||--|| ActualState: "has"

    DesiredState ||--|{ DesiredStateAttribute: "map[AttributeID]"
    DesiredStateAttribute{
        Kind AttributeKind "[DockerStack, Edge, Command]"
        Offset uint ""
        ScheduleTime timestamp "Sheduled date and time"
        Value string "Version for Docker and Edger or Name for command"
        Args strings "Enviroments for Docker Stack or Args for command"
        Abort bool "for command only"
    }    

    ActualState ||--|{ ActualStateAttribute: "map[AttributeID]"
    ActualStateAttribute{
        Kind AttributeKind "[DockerStack, Edge, Command]"
        Offset uint ""
        Time timestamp "Date and time of actualization"
        AttemptNum int "Attempt number"
        Status enum "[Pending, InProgress, Finished]"
        Error text "Last error text"
        Info text "Text result of last attempt"
    }

    Cloud ||--|{ EdgeNodeMetric: "has"            
    EdgeNodeMetric ||--|{ Metric: "has"
        Metric {
        Name string "for example `CPU_Usage`"
        Label string "component=docker container; containerID=scylla_db_container"
        Kind enum "[Counter, Guage]"
    }
    Metric ||--|{ Sample : "array of"
    Sample {
        Value float "counter or guage value"
        Time timestamp "date and time ms"
    }

```

# Goroutines

```mermaid
    flowchart TD

    PassedInterfaces:::GROUP
    subgraph PassedInterfaces
        IEdgeNodeState:::HARD
        GetDesiredState["GetDesiredState()"]:::SOFT --- IEdgeNodeState
        ReportActualState["ReportActualState()"]:::SOFT ---IEdgeNodeState

        ISuperController:::HARD
        AchiveState["AchiveState()"]:::SOFT --- ISuperController
        
        IMetricCollectors:::HARD
        CollectMetrics["CollectMetrics()"]:::SOFT --- IMetricCollectors

        IMetricReporters:::HARD
        ReportMetrics["ReportMetrics()"]:::SOFT --- IMetricReporters
    end

    PassedInterfaces -.- |passed to| Edger

    Edger["Edger(...)"]:::SOFT

        direction LR
        Edger --- |"go⚡"| GetDesiredStateCycle>"getDesiredStateCycle()"]:::SOFT
        GetDesiredStateCycle -.-> DesiredState[["LastStateChanel {DesiredState}"]]:::HARD

        Edger --- |"go⚡"| ControllerCycle>"superControllerCycle()"]:::SOFT
        DesiredState -.-> ControllerCycle
        ControllerCycle -.-> ActualState[["LastStateChanel {ActualState}"]]:::HARD

        Edger --- |"go⚡"| ReportActualStateCycle>"reportActualStateCycle()"]:::SOFT
        ActualState -.->  ReportActualStateCycle

        Edger ----- |"go⚡"| CollectMetricsCycle>"collectMetricsCycle()"]:::SOFT
        CollectMetricsCycle -.-> Metrics[["chan Metrics"]]:::HARD

        Edger ----- |"go⚡"| ReportMetricsCycle>"reportMetricsCycle()"]:::SOFT
        Metrics -.-> ReportMetricsCycle

    classDef GROUP fill:#FFFFFF,stroke:#000000, stroke-width:1px, stroke-dasharray: 5 5    
    classDef B fill:#FFFFB5
    classDef SOFT fill:#B5FFFF
    classDef HARD fill:#C9E7B7
```    

# SuperController

```mermaid
erDiagram
    ISuperController{
        AchieveState method "calls for each desired state"
    }
    ISuperController ||--|{ MicroControllerFactory : "map [AttributeKind] MicroControllerFactory"
    ISuperController ||--|{ MicroController : "map [ID] MicroController"

    MicroControllerFactory ||..|{ MicroController : "creates"

```

SuperController.AchieveState() called from edger superCycle:
- for every new desired state,
- for each unreached desired state after a configurable time interval (AchieveAttemptInterval)

# Save/restore SuperController state

- Restore is initialized in mctrls.New() (ISuperController constructor)
- SuperController.AchieveState() saves last achieved state
