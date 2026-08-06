---
change_id: 2608051451-scheduler-http-storage
type: fix
issue_url: https://untill.atlassian.net/browse/AIR-4662
domains: [prod]
scope: [apps, storage]
---

# Change request: Scheduler HTTP storage availability

Refs:

- [AIR-4662: voedger: enable sys.Storage_HTTP in scheduler state](./issue-AIR-4662.md)

## Why

Scheduled jobs can access HTTP storage exposed by the Voedger application platform, but the production VVM does not initialize the scheduler’s HTTP client. An HTTP storage read therefore ends in a nil-pointer panic instead of completing the request.

## What

Symptom: A scheduled job that reads from HTTP storage panics with a nil-pointer dereference.

```text
scheduled job reads HTTP storage
      |
      v
scheduler creates host state
      |
      v
VVM scheduler configuration       <-- fault: omits the managed HTTP client
      |
      v
HTTP storage receives a nil client
      |
      v
HTTP storage invokes ReqReader
      |
      v
nil-pointer panic                  (symptom)
```

Corrected behavior: Scheduled jobs use the VVM-managed HTTP client for HTTP storage reads and no longer panic because the client is uninitialized.

## How

Decisions:

- Keep HTTP client ownership and lifecycle at the VVM boundary, and inject the same managed client into scheduler configuration that is shared with other processor state factories.
- Preserve the existing scheduler-state registration and HTTP storage contract; propagate the managed dependency through the existing scheduler state path instead of creating a scheduler-specific client or a nil-client fallback in HTTP storage.
- Verify the scheduler-to-HTTP-storage client path without external I/O by reading an intentionally invalid URL, so the managed client is exercised but rejects the URL before reaching its transport.

Assumptions:

- None

Out of scope:

- Changing HTTP storage request, response, timeout, or retry semantics.

References:

- [VVM dependency graph and managed client lifetime](../../../../../pkg/vvm/provide.go)
- [generated scheduler configuration](../../../../../pkg/vvm/wire_gen.go)
- [scheduler dependency boundary](../../../../../pkg/processors/schedulers/interface.go)
- [scheduler state construction](../../../../../pkg/processors/schedulers/impl_scheduler.go)
- [scheduler HTTP storage registration](../../../../../pkg/state/stateprovide/impl_scheduler_state.go)
- [HTTP storage client use](../../../../../pkg/sys/storages/impl_http_storage.go)
- [scheduler integration-test pattern](../../../../../pkg/processors/schedulers/impl_test.go)

## Construction

### Tests

- [x] update: [sys/it/impl_jobs_test.go](../../../../../pkg/sys/it/impl_jobs_test.go)
  - add: an integration test that deploys the HTTP scheduler-job fixture through the real VVM and advances isolated scheduler time to run the job
  - verify: the job reaches the managed HTTP client through `sys.Storage_HTTP` and persists a success marker without sending an external HTTP request

- [x] create: [vit/schemaTestApp2WithJobHTTP.vsql](../../../../../pkg/vit/schemaTestApp2WithJobHTTP.vsql)
  - provide: a dedicated VIT application schema for scheduler HTTP storage coverage
  - declare: a result view with a `StorageObtained` marker and a built-in scheduled job with read access to `sys.Http` and intent access to that view
  - support: the end-to-end scenario without changing permissions of existing job fixtures

- [x] update: [vit/consts.go](../../../../../pkg/vit/consts.go)
  - embed: the scheduler HTTP-storage VSQL fixture for VIT application construction

- [x] update: [vit/shared_cfgs.go](../../../../../pkg/vit/shared_cfgs.go)
  - add: a fixture builder that registers the built-in HTTP-storage job implementation
  - read: `sys.Storage_HTTP` with an intentionally invalid URL and require the managed client to return `url.Error` before transport execution
  - persist: `StorageObtained` in the fixture result view after the client path is exercised successfully

### Runtime wiring

- [x] update: [vvm/provide.go](../../../../../pkg/vvm/provide.go)
  - inject: the VVM-managed `IHTTPClient` when Wire constructs `BasicSchedulerConfig`, matching other processor configurations
  - preserve: VVM ownership and cleanup of the shared client and all existing HTTP storage semantics

- [x] run: `go generate ./pkg/vvm/provide.go`
  - update: [vvm/wire_gen.go](../../../../../pkg/vvm/wire_gen.go) so generated scheduler configuration assigns the managed HTTP client

- [x] run: `go test ./pkg/sys/it -run TestJobs_HTTPStorage`
  - verify: scheduler HTTP storage works through production VVM dependency wiring
