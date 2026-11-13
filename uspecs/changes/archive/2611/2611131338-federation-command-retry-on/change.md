---
change_id: 2611131328-federation-command-retry-on
type: feat
issue_url: https://untill.atlassian.net/browse/AIR-5264
domains: [prod]
scope: [extensions]
---

# Change request: Configurable federation command retries

Refs:

- [AIR-5264: voedger: support RetryOn in storage federation command key](./issue-AIR-5264.md)

## Why

Application developers currently need to duplicate retry logic when invoking federation commands that return temporary HTTP errors. Allowing callers to select retry status codes lets them use the platform's standard retry mechanism instead.

## What

In the Voedger application platform's extensions context:

- Callers can configure federation command retries with a comma-separated string of HTTP status codes in the command storage key
- Federation command calls use the standard retry mechanism for each configured status code

## How

Decisions:

- Keep retry selection in the command storage adapter and delegate execution to the existing federation request options and HTTP retry mechanism, preserving its backoff, retry duration, request replay, and cancellation behavior
- Add policies only for explicitly supplied status codes, preserving existing federation behavior when the retry key is omitted or empty
- Keep retry selection independent of expected response codes, so a configured retry status is retried before the final response is evaluated against expected codes
- Follow the existing expected-code list parsing convention, returning conversion errors before sending a request, and include retry configuration in storage key equality
- Verify the storage-to-HTTP path with a real federation client and controlled HTTP responses, checking repeated requests and final results rather than only configured mock returns

Assumptions:

- None

Out of scope:

- Retrying emulated federation command handlers or adding retry selection to other storages
- New retry timing controls or changes to the shared HTTP retry policy

References:

- [command storage key handling and request construction](../../../../../pkg/sys/storages/impl_federation_cmd_storage.go)
- [federation request option forwarding](../../../../../pkg/coreutils/federation/impl.go)
- [federation client retry defaults](../../../../../pkg/coreutils/federation/provide.go)
- [per-status retry configuration](../../../../../pkg/goutils/httpu/impl_opts.go)
- [HTTP retry execution and response evaluation](../../../../../pkg/goutils/httpu/impl.go)
- [existing federation retry and cancellation coverage](../../../../../pkg/coreutils/federation/impl_test.go)

## Construction

- [x] create: [sys/storages/impl_federation_cmd_storage_test.go](../../../../../pkg/sys/storages/impl_federation_cmd_storage_test.go)
  - Exercise command storage with a real federation client and a controlled local HTTP server, reusing existing application-structure test setup
  - Verify a comma-separated retry list handles successive 503 and 502 responses before returning a successful command result, preserving the request body and authorization across attempts
  - Verify omitted and empty retry values preserve single-attempt behavior, and a status absent from a nonempty retry list is returned without retry
  - Verify retry selection remains independent of expected codes, including a status present in both lists
  - Verify malformed retry values return an error without issuing an HTTP request, following existing expected-code parsing conventions
  - Verify keys with matching retry configuration compare equal and keys with different retry configuration compare unequal
  - Use one root assertion instance per test, brief comments explaining assertions, and bounded test contexts so broken retry behavior cannot hang the suite

- [x] update: [pkg/sys/const.go](../../../../../pkg/sys/const.go)
  - Add the public federation command field constant for `RetryOn`

- [x] update: [sys/storages/impl_federation_cmd_storage.go](../../../../../pkg/sys/storages/impl_federation_cmd_storage.go)
  - Accept and retain the retry string through the command key builder and include it in key equality
  - Parse the comma-separated retry codes before invoking federation, skipping empty entries and propagating integer conversion errors
  - Append a standard per-status HTTP retry option for each supplied code while preserving expected-code, authorization, response decoding, and emulation behavior
  - Reuse the existing HTTP retry machinery without introducing a storage-level retry loop or enabling the full default retry policy

- [x] update: [pkg/sys/sys.vsql](../../../../../pkg/sys/sys.vsql)
  - Document the optional `RetryOn` command storage key, its comma-separated format, and its default of no additional status retries
  - Explain that retry codes control repeated attempts independently of expected response codes and use the standard HTTP retry duration

## Quick start

When building a federation command key, select the statuses to retry:

```go
kb.PutString(sys.Storage_FederationCommand_Field_RetryOn, "503,502")
```

Set the command, body, and other key fields as usual, then read the command storage value. Responses with status 503 or 502 use the standard retry mechanism, with its default one-minute maximum retry duration and cancellation through the federation client's context. Omit `RetryOn` or set it to an empty string to preserve existing retry behavior. `ExpectedCodes` continues to control which final response statuses are accepted.
