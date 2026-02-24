# Implementation plan

## Construction

### httpu package

- [x] update: [pkg/goutils/httpu/provide.go](../../../pkg/goutils/httpu/provide.go)
  - add: `NewIHTTPClientWithTransport` factory that accepts custom `http.RoundTripper`
  - refactor: extract `newIHTTPClient` helper shared by both factories

### Interface and type changes

- [x] update: [pkg/state/types.go](../../../pkg/state/types.go)
  - remove: `IHTTPClient` interface (replaced by `httpu.IHTTPClient`)
  - update: `StateOpts.CustomHTTPClient` field type from `IHTTPClient` to `httpu.IHTTPClient`

### Storage implementation

- [x] update: [pkg/sys/storages/impl_http_storage.go](../../../pkg/sys/storages/impl_http_storage.go)
  - update: `httpStorage.customClient` field type from `state.IHTTPClient` to `httpu.IHTTPClient`
  - update: `NewHTTPStorage` parameter type and create default `httpu.IHTTPClient` when nil is passed
  - update: `Read` to use `httpu.ReqReader` with options instead of `customClient.Request()` / `http.DefaultClient.Do()`
  - remove: fallback to `http.DefaultClient.Do()`

### Test framework

- [x] update: [pkg/state/teststate/interface.go](../../../pkg/state/teststate/interface.go)
  - rename: `PutHTTPHandler` to `PutHTTPMock` in `ITestAPI` interface

- [x] update: [pkg/state/teststate/impl.go](../../../pkg/state/teststate/impl.go)
  - add: `httpClient httpu.IHTTPClient` field and `testRoundTripper` that delegates to `httpHandler`. Add comprehensive comment describing why this RoundTripper is needed
  - remove: `Request` method (replaced by `testRoundTripper` + `httpu.IHTTPClient`)
  - update: `NewTestState` to create `httpu.IHTTPClient` via `httpu.NewIHTTPClientWithTransport`
  - update: `buildState` to pass `httpClient` as `StateOpts.CustomHTTPClient`
  - rename: `PutHTTPHandler` to `PutHTTPMock`

### Test callers

- [x] update: [pkg/iextengine/wazero/\_testdata/tests/wasm/main_test.go](../../../pkg/iextengine/wazero/_testdata/tests/wasm/main_test.go)
  - rename: `PutHTTPHandler` call to `PutHTTPMock`
