# AIR-3665: make q.cluster.VSqlUpdate2 support all features of c.cluster.VSqlUpdate

- Key: AIR-3665
- Type: Sub-task
- Status: In Progress
- Assignee: Denis Gribanov (d.gribanov@dev.untill.com)
- URL: https://untill.atlassian.net/browse/AIR-3665

## Summary

make q.cluster.VSqlUpdate2 support all features of c.cluster.VSqlUpdate

## Description

What

- Frontend calls `q.cluster.VSqlUpdate2` only for all features of `c.cluster.VSqlUpdate`
- It always calls `c.cluster.LogVSqlUpdate`, then does the stuff related to the query
- Responses from the command are transformed to query response by the router
