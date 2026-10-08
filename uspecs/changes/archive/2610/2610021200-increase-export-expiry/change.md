---
change_id: 2610021050-increase-export-expiry
type: feat
issue_url: https://untill.atlassian.net/browse/AIR-4018
domains: [prod]
scope: [apps]
---

# Change request: Configurable temporary BLOB retention

Refs:

- [AIR-4018: Increase export expiry date](./issue-AIR-4018.md)

## Why

Need to have more expirations supported for temporary BLOBs

## What

In the Voedger application platform's apps context:

- API v2 clients can select a temporary BLOB retention period through the TTL header instead of being limited to the one-day value
- A temporary BLOB remains readable for the selected retention period and expires when that period elapses

## How

Decisions:

- Add an explicit 90-day temporary BLOB duration exposed as `TTL: 90d`, while retaining support for `TTL: 1d`; do not generalize TTL parsing to arbitrary durations
- Carry the selected duration through the existing day-based storage contract and duration-specific authorization command boundary
- Verify the new duration at the API mapping, BLOB processor, and end-to-end storage-expiration boundaries
- Reuse a single parameterized temporary BLOB expiration test flow for all supported durations, supplying the expected TTL to each invocation; do not add dedicated tests targeted specifically at 90-day expiration

Assumptions:

- Three months means a fixed 90-day retention period rather than three calendar months
- The FDM exporter will explicitly send `TTL: 90d`

Out of scope:

- Accepting arbitrary TTL values or TTLs expressed in seconds
- Changing current behavior for requests that omit the TTL header
- Updating the FDM exporter in the separate `airs-bp3` repository

References:

- [day-based temporary BLOB duration contract](../../../../../pkg/iblobstorage/types.go)
- [supported API TTL mappings](../../../../../pkg/coreutils/federation/consts.go)
- [temporary BLOB duration selection and authorization dispatch](../../../../../pkg/processors/blobber/impl_write.go)
- [system command authorization boundary](../../../../../pkg/sys/sys.vsql)
- [temporary BLOB lifetime integration behavior](../../../../../pkg/sys/it/impl_blob_test.go)
- [migration to directly stored day counts](../../../archive/2601/2601201509-tempblob-duration-type-stores-days/change.md)

## Construction

### Tests

- [x] update: [sys/it/impl_blob_test.go](../../../../../pkg/sys/it/impl_blob_test.go)
  - refactor: the existing temporary BLOB lifecycle test into one parameterized test flow invoked with the one-day and 90-day durations
  - update: supply the expected TTL to each invocation, advance the mock clock to that boundary, and verify the BLOB is readable immediately before expiration and unavailable when the TTL elapses
  - preserve: use the shared flow for both durations; do not add a dedicated test targeted specifically at 90-day expiration

- [x] update: [pkg/sys/sys.vsql](../../../../../pkg/sys/it/testdata/apps/test2.app1/image/pkg/sys/sys.vsql)
  - add: declare `RegisterTempBLOB90d` in the copied system schema used by the sidecar application fixture
  - preserve: keep the fixture schema aligned with runtime command registration so sidecar deployment succeeds

### Duration contract

- [x] update: [iblobstorage/consts.go](../../../../../pkg/iblobstorage/consts.go)
  - add: the 90-day `DurationType` constant while preserving the one-day constant and the existing day-based representation

### API and BLOB processor

- [x] update: [coreutils/federation/consts.go](../../../../../pkg/coreutils/federation/consts.go)
  - add: bidirectional mappings between `90d` and the 90-day duration so federation clients emit and accept the new supported TTL

- [x] update: [processors/blobber/consts.go](../../../../../pkg/processors/blobber/consts.go)
  - add: map the 90-day duration to its temporary BLOB registration command

- [x] update: [processors/blobber/impl_write.go](../../../../../pkg/processors/blobber/impl_write.go)
  - update: replace the one-day-only validation error with an error that remains accurate for the complete supported TTL set

### Authorization

- [x] update: [sys/sys.vsql](../../../../../pkg/sys/sys.vsql)
  - add: a tagged registration command for 90-day temporary BLOB uploads, parallel to the existing one-day command

- [x] update: [sys/blobber/provide.go](../../../../../pkg/sys/blobber/provide.go)
  - add: register the 90-day temporary BLOB command with the same no-op execution behavior as the one-day command
  - refactor: register both supported temporary BLOB commands through one shared name-driven loop

- [x] update: [processors/oldacl/consts.go](../../../../../pkg/processors/oldacl/consts.go)
  - add: the 90-day registration command QName used by the legacy Air authorization policy

- [x] update: [processors/oldacl/acl.go](../../../../../pkg/processors/oldacl/acl.go)
  - update: allow `air.BOReader` to execute both supported temporary BLOB registration commands

## Quick start

Create a temporary BLOB retained for 90 days by sending the new TTL value explicitly:

````text
POST /api/v2/apps/{owner}/{app}/workspaces/{wsid}/tblobs
Authorization: Bearer {PrincipalToken}
Content-Type: {blobContentType}
Blob-Name: {blobName}
TTL: 90d

{blobData}
````

Continue to use `TTL: 1d` for one-day retention. Omitting the TTL header is not supported by this change.
