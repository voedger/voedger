# voedger: support RetryOn in storage federation command key

- URL: https://untill.atlassian.net/browse/AIR-5264
- ID: AIR-5264
- State: In Progress⚒️
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov
- Linked PRs/issues: Parent AIR-5216 (DEV: Germany fiscalization issues); blocks AIR-5217 (WASM fiscalcloud: projector incomplete due to 503 in federation calls)

## Why

Need to apply the standard retry mechanism on a specified status codes on invoking command via `FederationCommand` storage to avoid duplication of retry mechanism on the caller side

## What

* support `RetryOn` key in `FederationCommand` storage. Value - comma-separated string of status codes to retry
* use standard retry on each provided status code
