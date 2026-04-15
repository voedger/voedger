# Implementation plan: Fix body loss on retry in HTTPClient

## Construction

- [x] update: [impl_test.go](../../../../../../pkg/goutils/httpu/impl_test.go)
  - add: `TestReqReaderBodyPreservedOnRetry` — server returns 503 on first attempt and 200 on second, assert request body is identical on both attempts (now should fail)
- [x] Review
- [x] update: [impl.go](../../../../../../pkg/goutils/httpu/impl.go)
  - fix: In `req()`, buffer `opts.bodyReader` into `[]byte` via `io.ReadAll` after `compileOpts`; inside the retry closure create a fresh `bytes.NewReader` from the buffered bytes instead of reusing `opts.bodyReader`
