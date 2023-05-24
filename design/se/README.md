# Standart Edition (SE)

Principles

- Node must be clean Ubuntu node
  - Reason: We believe it will avoid possible conflicts between installed software and reduce operation costs
- Balancer (e.g.  https://www.hetzner.com/cloud/load-balancer) should be used for redirecting traffic to the voedger containers
- Edger is not used (though its functionality is similar to what we need to maintain cluster nodes)
  - Reason: Edger requires connection to working Application which does not exist yet
- Orchestrator: swarm
  - Every node is manager


## Nodes

```mermaid
    erDiagram

    SECluster ||--|| AppNode1: ""
    SECluster ||--|| AppNode2: ""
    SECluster ||--|| DBNode1: ""
    SECluster ||--|| DBNode2: ""
    SECluster ||--|| DBNode3: ""    

    AppNode1 ||..|| SEDockerStack: ""
    AppNode2 ||..|| SEDockerStack: ""

    AppNode1 ||..|| MonDockerStack: ""
    AppNode2 ||..|| MonDockerStack: ""

    DBNode1 ||..|| DBDockerStack: ""
    DBNode2 ||..|| DBDockerStack: ""
    DBNode3 ||..|| DBDockerStack: ""

    MonDockerStack ||--|| "prometheus1,2": ""
    MonDockerStack ||--|| "alertmanager1,2": ""
    MonDockerStack ||--|| "graphana1,2": ""

    SEDockerStack ||--|| "voedger": ""

    DBDockerStack ||--|| "scylla1,2,3": ""
```

## Using grafana

