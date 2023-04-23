# Must Be Re-implemented

- iplogcas, iwlogcas:  partition is kept in Cassandra partition which is very limited
  - Cassandra does not like big partitions
- iplog and iwlog implementations have copypasted - should be combined

# Should Be Re-implemented

- wlog may not keep PLogOffset
- Router: only one router
- Router: answerChannel.taskID is only partial solution

 # Questionable

- idb: type TWSID uint32: 4.294.967.295 workspaces per cluster