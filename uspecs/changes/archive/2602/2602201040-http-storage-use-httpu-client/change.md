---
registered_at: 2026-02-19T13:31:37Z
change_id: 2602191331-http-storage-use-httpu-client
baseline: f38d48c38dc8eaafa0eeb9c6c4834ba6e1aeb310
archived_at: 2026-02-20T10:40:32Z
---

# Change request: HTTP state storage: use httpu.IHTTPClient instead of state.IHTTPClient

## Why

The HTTP state storage currently uses a local `state.IHTTPClient` interface and falls back to the standard HTTP client when not set. This means the default retry policy and other features provided by `httpu.IHTTPClient` are not used.

## What

Replace `state.IHTTPClient` with `httpu.IHTTPClient` in HTTP state storage:

- Use `httpu.IHTTPClient` in `impl_http_storage.go` instead of the local `state.IHTTPClient`
  -  do not re-implement http read logic, use `httpu.IHTTPClient` where possible
- Rename `teststate.PutHTTPHandler` to `PutHTTPMock`
- Eliminate HTTP-related interfaces and types from the `state` package where possible, relying on the `httpu` package instead
