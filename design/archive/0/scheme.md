```dot
graph graphname {

    graph[rankdir=TB splines=ortho]
    node [ fontname = "Cambria" shape = "record" fontsize = 12]
    edge [dir=both arrowhead=none arrowtail=none]
    struct[label= "{struct|descr}"]
    field[label= "{field|descr\l}"]
    method[label= "{method|descr\largs\lresult\l}"]
    func[label= "{function|descr\lsynchronized\largs\lsize\lresult\l}"]
    View[label= "{view|descr\lsynchronized\largs\lsize\lresult\l}"]
    collection[label= "{collection|descr\lsize\l}"]

    scheme -- struct[arrowhead=crow]
    scheme -- collection[arrowhead=crow]
    scheme -- View[arrowhead=crow]
    struct -- field[arrowhead=crow]
    collection -- method[arrowhead=crow]
    collection -- field[arrowhead=crow]
    scheme -- func[arrowhead=crow]
}
```

- `structs` and `collections` are in same namespace
- `method`: is a function with ID as a first implicit parameter
- `synchronized`: true/false
- `size`: `none`, `one`, `tiny`, `small`, `huge`
  - `none` for structures which do not represent `collections`
    - If `size` is not `none` struct gets implicit field `ID`
  - `one` and `tiny`: entire collection can be read by `synchronized` functions
  - `one`, `tiny`, `small`: entire collection can be read by `parallel` functions
  - `huge`
    - allowed for `views` only
    - is not possible to get entire collection at all