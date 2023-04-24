# Application

```dot
graph graphname {

    graph[rankdir=TB splines=ortho]
    node [ fontname = "Cambria" shape = "record" fontsize = 12]
    edge [dir=both arrowhead=none arrowtail=none]
    App[label="Application"]
    MainService[label="Main Service"]
    HiCoFunction[label="Hi-Co Function"]
    LoCoFunction[label="Lo-Co Function"]
    App -- Service[arrowhead=crow]
    Service -- MainService[arrowtail=empty]
    Service -- Module[arrowhead=crow]
    Module--Function[arrowhead=crow]
    Function--HiCoFunction[arrowtail=empty]
    Function--LoCoFunction[arrowtail=empty]
    Module--Struct[arrowhead=crow]
    Struct--Collection[arrowtail=empty]
    Module--View[arrowtail=empty]
}
```

- Only `Main Service` may have Hi-Co functions
- Normal service can modify only Projections
- Views are Projections which are managed by heeus
- `Hi-Co` - High Consistency
  - Only Hi-Co functions can modify database
- `Lo-Co` - Low Consistency
  - Lo-Co functions are read-only

# Application Deployment

```dot
graph graphname {
    graph[rankdir=TB splines=ortho]
    node [ fontname = "Cambria" shape = "record" fontsize = 12]
    edge [dir=both arrowhead=none arrowtail=none]
    Role[label="App Role"]
    WCluster[label="Worker Cluster"]
    AppDatabase [shape=cylinder]
    AppD[label="Application Deployment"]
    AppD -- WCluster[arrowhead=crow]
    WCluster -- Role[arrowhead=crow]
    WCluster -- AppDatabase
    Role -- Service[arrowhead=crow]
    Service -- Module[arrowhead=crow]
    Service -- Task [arrowhead=crow]
}
```

## Application Roles

```dot
graph graphname {
    graph[rankdir=TB splines=ortho]
    node [ fontname = "Cambria" shape = "record" fontsize = 12]
    edge [dir=both arrowhead=none arrowtail=none]
    Role[label="App Role"]
    CanaryRelease[label="Canary Release"]

    Role -- Main[arrowtail=empty]
    Role -- CanaryRelease[arrowtail=empty]
}
```

- `Main` role - application role requests are routed to by default
- Roles are needed for `Canary Releases`
  - https://martinfowler.com/bliki/CanaryRelease.html
- Each role has its own set of workspaces

# Application Database

```dot
graph graphname {

    graph[rankdir=TB splines=ortho]
    node [ fontname = "Cambria" shape = "record" fontsize = 12]
    edge [dir=both arrowhead=none arrowtail=none]
    Database [shape=cylinder label="AppDatabase"]
    Database -- Partition[arrowhead=crow]
    Partition -- Workspace[arrowhead=crow]
    Workspace -- WLog
    Workspace -- State
    Workspace -- Role
    State -- Record[arrowhead=crow]
    Workspace -- View[arrowtail=empty]
    Partition -- PLog
    Partition -- SagaExecution[arrowhead=crow]
}
```





