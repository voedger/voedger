---
registered_at: 2026-03-20T09:26:40Z
change_id: 2603200926-support-sysraw-query-arg
issue_url: https://untill.atlassian.net/browse/AIR-3351
baseline: 7ea4354ab277b8b7ddc97fc3b2ed35bacc0f9c95
archived_at: 2026-03-25T09:21:36Z
---

# Change request: Support sys.Raw argument in Query Processor V2

## Why

Query Processor V2 needs to handle queries that use `sys.Raw` as their argument type. Currently, `sys.Raw` arguments may not be properly supported in the query processing pipeline, which limits the flexibility of query definitions.

## What

Add support for `sys.Raw` argument type in Query Processor V2:

- Handle `sys.Raw` as a valid query parameter type in the query processing and OpenAPI schema generation
- Ensure queries with `sys.Raw` arguments are correctly represented in the generated OpenAPI schema
