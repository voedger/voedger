# AIR-3656: Reject c.cluster.VSqlUpdate if c.sys.CUD would get to the same command processor

- Key: AIR-3656
- Type: Bug
- Status: In Progress
- Assignee: d.gribanov@dev.untill.com
- URL: <https://untill.atlassian.net/browse/AIR-3656>

## Why

Executed:

```sql
c.cluster.VSqlUpdate: update untill.fiscalcloud.140737488486400.fiscalcloud.Customer.322685000131099 set ClientID = '', ClientConfigured = false
```

The request hung.

## What

If the `c.sys.CUD` would go to the WSID that is serviced by the same command processor, then `c.sys.CUD` will hang for sure, so reject the request.

## Plan

- New `q.cluster.VSqlUpdate2` + `c.cluster.LogVSqlUpdate` + router
- Router reroutes `c.cluster.VSqlUpdate` to `q.cluster.VSqlUpdate2` and transforms the response to match command response format
- `q.cluster.VSqlUpdate2`:
  - Calls `c.cluster.LogVSqlUpdate`
  - Calls CUD in the target workspace
  - Returns WLog offsets of `c.cluster.LogVSqlUpdate` and CUD
- `c.cluster.LogVSqlUpdate` does nothing, just logs params
- Ask frontend to use `q.cluster.VSqlUpdate2`
- Wait until Live uses `q.cluster.VSqlUpdate2`
- Get rid of `c.cluster.VSqlUpdate` and special routing
