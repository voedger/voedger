---
registered_at: 2026-03-02T09:57:27Z
change_id: 2603020957-blob-emit-content-length
baseline: 120782065947f5fc7305be28f4f987e2685a602c
archived_at: 2026-03-02T10:33:32Z
---

# Change request: Emit blob size in Content-Length header on read

## Why

HTTP clients benefit from a `Content-Length` header to display progress and allocate buffers correctly. Currently the blob read path does not set this header, so clients cannot know the size of the blob upfront.

## What

Refactor the blob processor read path to emit the `Content-Length` header:

- Refactor the blob processor so the renamed function is called exactly once inside `IBLOBStorage.ReadBLOB`, setting the `Content-Type`, `BlobName`, and `Content-Length` headers — taking `Content-Length` from `iblobstorage.BLOBState.Size` returned by `ReadBLOB`
- Update the relevant integration tests in `impl_blob_test.go` to assert that the `Content-Length` header is present and correct
