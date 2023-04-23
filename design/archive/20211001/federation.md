# Federation

```dot
graph graphname {

    graph[rankdir=TB splines=ortho]
    node [ fontname = "Cambria" shape = "record" fontsize = 12]
    edge [dir=both arrowhead=none arrowtail=none]
    Federation
    MCluster[label="Master Cluster"]
    WCluster[label="Worker Cluster"]
    Federation -- MCluster
    Federation -- Region[arrowhead=crow]
    Region -- WCluster [arrowhead=crow]
    WCluster -- DC[arrowhead=crow]
}
```

- Client requests go to clusters (Master or Worker)
- Cluster can span few datacenters
  - AKA `Stretched cluster`
  - See also links in [ha.md](ha.md)

# Amazon

```dot
graph graphname {

    graph[rankdir=TB splines=ortho]
    node [ fontname = "Cambria" shape = "record" fontsize = 12]
    edge [dir=both arrowhead=none arrowtail=none]
    Amazon -- Region[arrowhead=crow]
    Region -- AvailabilityZone[arrowhead=crow]
}
```

- Each Region is completely independent
- Each Availability Zone is isolated, but the Availability Zones in a Region are connected through low-latency links

|  Amazon | Heeus     |
|:-------:|:---------:|
| Amazon     |Federation |
| Region     |Region     |
| Avail. Zone| Cluster   |


# Links

- https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-regions-availability-zones.html


