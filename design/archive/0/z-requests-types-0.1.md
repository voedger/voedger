# Requests Handling

```dot
graph graphname {

    graph[rankdir=TB splines=ortho]
    node [ fontname = "Cambria" shape = "record" fontsize = 12]
    edge [dir=both arrowhead=none arrowtail=none]

    R [label= "Request"]
    H [label= "Handling"]
    S [label= "Serializable"]
    P [label= "Parallel"]
    WR [label= "Write Requests"]
    ROR [label= "Read-Only Requests"]

    R -- H[arrowtail=diamond]
    H -- S [arrowtail=empty]
    H -- P [arrowtail=empty]
    S -- WR [style=dashed]
    S -- ROR [style=dashed]
    P -- ROR [style=dashed]
}
```

- `Serializable` is applied for Partition
- `Serializable Handling` leads to `Serializable Isolation`

# Glossary

- `Serializable Handling`: Последовательная обработка
- `Consistency`: Согласованность
- `Strong consistency`: Сильная согласованность
  - After the update completes, any subsequent access (by A, B, or C) will return the updated value
  - После завершения обновления любой последующий доступ к данным вернет обновленное значение
  - Such requests must be handled in a tiny time
- `Weak consistency`: Слабая согласованность
  - The system does not guarantee that subsequent accesses will return the updated value
  - Система не гарантирует, что последующие обращения к данным вернут обновленное значение
  - Weak consistency requests can be handled in long time

# Links

- https://habr.com/ru/post/100891/
- https://www.allthingsdistributed.com/2008/12/eventually_consistent.html
