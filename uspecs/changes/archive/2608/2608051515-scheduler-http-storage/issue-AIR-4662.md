# voedger: enable sys.Storage_HTTP in scheduler state

- URL: https://untill.atlassian.net/browse/AIR-4662
- ID: AIR-4662
- State: in-progress
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov

## Why

Scheduler state exposes `sys.Storage_HTTP`, but the scheduler’s `IHTTPClient` is not initialized by the production VVM wiring. Any scheduled job that reads from HTTP storage can therefore panic with a nil-pointer dereference.

## What

* Scheduled jobs can use `sys.Storage_HTTP` without panicking.
* HTTP requests from scheduler state use the VVM-managed HTTP client.
* Scheduler HTTP-storage behavior is covered by an integration test.

