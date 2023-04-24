# Datacenter HA

```dot
digraph name {
    node [ fontname="Cambria" shape=rect fontsize=12]

    subgraph cluster_dc1 {
        label = "dc1";
        cas1_1 [shape = "cylinder"]
        cas1_2 [shape = "cylinder"]
        app1_1
    }
    subgraph cluster_dc2 {
        label = "dc2";
        cas2_1 [shape = "cylinder"]
        cas2_2 [shape = "cylinder"]
        app1_2 
    }
    subgraph cluster_dc3 {
        label = "dc3";
        cas3_1 [shape = "cylinder"]
        cas3_2 [shape = "cylinder"]
        app1_3
    }

    edge [dir=both style=dotted]
    app1_1 -> cas1_1
    app1_1 -> cas2_1
    app1_2 -> cas1_1
    app1_2 -> cas3_1
    app1_3 -> cas2_2
    app1_3 -> cas1_2
}
```

# App Update

- Zero Downtime
    - Clients do not get server errors (e.g. 503)
    - Latency growth MUST be minimized
- Persistent Cache

```dot
digraph cluster {
    node [ fontname = "Cambria" fontsize = 12 shape = "rect"]

    subgraph cluster_ac {
        label = "App Container";
        cache [label="cache.prc"]
        old [label="oldApp.prc"]
        new [label="newApp.prc" style=dashed]
        cder [label="cder.prc"]
    }
    hbuilder -> hcc [style=dotted]
    hcc -> cder
    cder -> new
    cder -> old
    cache -> new [dir=none style=dotted]
    cache -> old [dir=none style=dotted]
}
```

- cache.prc is a separate process which shares cache memory with apps
- Own memory manager
  - https://github.com/couchbase/go-slab 

## App Update: Java

```dot
digraph name {
    node [ fontname = "Cambria" fontsize = 12 shape = "rect"]

    subgraph cluster_node {
        label = "node";
        cache [label="cache.prc"]
        old [label="oldApp.fatjar"]
        new [label="newApp.fatjar" style=dashed]
        cder [label="cder.prc"]
        core [label="core.jar"]
    }
    hbuilder -> hcc [style=dotted]
    hcc -> cder
    cder -> core
    core -> old
    core -> new
    cache -> core [dir=none style=dotted]
}
```

- Cache can be inside `core.jar`, but  will be lost during core.jar update

# Node/Container Failure

```dot
digraph graphname {

    graph[rankdir=BT splines=ortho]
    node [ fontname = "Cambria" shape = "rect" fontsize = 12]
    edge [dir=both arrowhead=none arrowtail=none]

    Database[shape = "cylinder"]
    PLog[shape = "cylinder"]
    WLog[shape = "cylinder"]
    WLogP[label="WLog.Partition"]
    State[shape = "cylinder"]
    StateP[label="State.Partition"]
    Partition[label="PLog.Partition"]
    Workspace
    Container [label="Main App Container" shape=box3d]

    Container -> Database[arrowtail=crow]
    Partition -> PLog [arrowtail=crow]
    Partition -> Container [arrowtail=crow]
    Workspace -> Partition [arrowtail=crow]
    PLog -> Database
    WLog -> Database
    State -> Database
    WLogP -> Workspace
    WLogP -> WLog [arrowtail=crow]
    StateP-> Workspace
    StateP->State [arrowtail=crow]
    

}
```

## Distributed Request Handling
```dot
digraph name {
    node [ fontname = "Cambria" fontsize = 12 shape = "rect"]
    fd[label="Detect container failure"]
    cu[label="Mark container as `Unavailable`"]
    fd -> cu
}
```

## Partitioned  Request Handling

```dot
digraph name {
    node [ fontname = "Cambria" fontsize = 12 shape = "rect"]
    fd[label="Detect container failure"]
    cu[label="Mark container as `Unavailable`"]
    en[label="Elect container for PLog.Parition"]
    iph[label="Initialize PartitionHandler"]
    fd -> cu
    cu -> en
    en -> iph
}
```

# Links 

- [Дешевле, надежнее, проще / Александр Христофоров (Одноклассники)](https://youtu.be/Hs2txKgnpAk?t=130)
- [Maintaining Consistency Across Data Centers(Randy Fradin, BlackRock) | Cassandra Summit 2016](https://www.slideshare.net/DataStax/maintaining-consistency-across-data-centers-randy-fradin-blackrock-cassandra-summit-2016)
  - Maintaining Consistency Across Data Centers or: How I Learned to Stop Worrying About WAN Latency Randy Fradin BlackRock
  - Challenge 1: Latency With all that latency on each operation, isn’t performance terrible? 
  - Actually, this wasn’t such a problem: 
  - 10ms+ latency per operation is acceptable for many apps 
  - Minimize use of sequential operations 
  - High throughput still achievable

