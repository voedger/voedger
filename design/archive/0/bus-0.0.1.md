# Partition Request Class Diagram

```dot
digraph graphname {

    graph[rankdir=BT splines=ortho]
    node [ fontname = "Cambria" shape = "record" fontsize = 12]
    edge [ arrowhead = "empty" ]
    PR [label= "{Partition Request||}"]
    MR [label= "{Write Request||}"]
    ROR [label= "{Read Request||}"]
    MR -> PR;
    ROR -> PR;

}
```

- Partition Request: `Запрос к разделу`

# App Container Structure

```dot
digraph graphname {
    compound=true;
    node [ fontname = "Cambria" shape = "record" fontsize = 12]

    subgraph cluster_router_out {
		label = "Router.prc"
        RouterOutput
	}

    subgraph cluster_container{
        label = "Container";
        CDer [label="cder.prc"]
        Secrets[shape = cylinder]
        subgraph cluster_appcache{
           label = "cache.prc";
           Cache[shape = cylinder]
        }
        subgraph cluster_app{
            label = "App.prc";
            App[label=PEngine]
            ipcache
            ipdb
            module1
            module2
        }
        subgraph cluster_db{
            label = "DB Apps";
            state [label="state.prc"]
            plog [label="plog.prc"]
            wlog [label="wlog.prc"]
            stock [label="stock.prc"]
        }
    }

    subgraph cluster_router_in {
		label = "Router.prc"
        RouterInput
	}

    RouterOutput -> App
    App -> RouterInput

    edge [dir=none style=dotted]
    ipdb -> App
    Secrets -> state [lhead=cluster_db]
    ipdb -> state [lhead=cluster_db]
    ipdb -> ipcache [lhead=cluster_db]
    App -> module1
    App -> module2
    Cache -> ipcache


}
```

- `cache.prc`
  - Separate process allows to update `App.prc`
  - Shared mem (with `App`) gives the best speed
  - Unix sockets gives simplicity



# PEngine Structure

```dot
digraph graphname {
    graph[rankdir=BT splines=ortho]
    node [ fontname = "Cambria" shape = "record" fontsize = 12]
    App[label="App"]
    PEngine
    PHandler[label="PartitionHandler"]
    
    
    edge [dir=both arrowhead=none arrowtail=none]
    PEngine -> App
    PHandler -> PEngine [arrowtail=crow]
    PReadEngine -> PEngine
    WriteHandler -> PHandler
    RWRouter -> PHandler
    PCache -> PHandler
    ReadChannel -> PReadEngine
    ReadHandler -> ReadChannel [arrowtail=crow]
}
```

# PEngine Dataflow

```dot
digraph graphname {
    node [ fontname = "Cambria" shape = "record" fontsize = 12]

    subgraph cluster_engine {
        label = "PEngine";
        subgraph cluster_ph{
            label = "partitionHandler1"
            RWRouter [label="RWRouter"]
            Writer [label="WriteHandler"]
            ipdb
        }
        subgraph cluster_readers {
            label = "PReadEngine"
            readHandler1
            readHandler2
            ReadChannel
        }
    }
    subgraph cluster_router_out {
		label = "Router"
        RouterOutput
	}
	subgraph cluster_router_in {
		label = "Router"
        RouterInput
	}


    RouterOutput-> RWRouter
    RWRouter -> Writer
    RWRouter -> ReadChannel
    ReadChannel -> readHandler1
    ReadChannel -> readHandler2
    readHandler1 -> RouterInput
    readHandler2 -> RouterInput
    Writer -> RouterInput
    ipdb -> readHandler1 [style=dotted dir=none]
    ipdb -> readHandler2 [style=dotted  dir=none]
    ipdb -> Writer [style=dotted dir=none]
}
```
