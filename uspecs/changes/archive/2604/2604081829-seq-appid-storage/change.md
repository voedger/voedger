---
registered_at: 2026-04-08T14:15:21Z
change_id: 2604081415-seq-appid-storage
baseline: 789bd332e92912b1776a08371572c9973fe9452d
issue_url: https://untill.atlassian.net/browse/AIR-3530
archived_at: 2026-04-08T18:29:10Z
---

# Change request: Scope VVM PLog offset storage by AppID

## Why

`IVVMSeqStorageAdapter.GetNumber` and `PutNumbers` already use storage keys that include `appID`, but PLog offset keys previously did not. As a result, different applications sharing the same partition could interfere with each other and receive sequential numbers (1, 2) instead of independent numbering (1, 1). See [issue.md](issue.md) for details.

## What

Updated the VVM storage adapter to scope PLog offset storage by `appID`:
- `IVVMSeqStorageAdapter.GetNumber` and `PutNumbers` already accept `appID` and use it in their storage keys
- VVM storage adapter implementation now encodes `appID` into PLog offset keys per application
- Added comprehensive tests verifying that different apps on the same partition get independent sequence numbers
- Updated sequences architecture documentation with VVM storage key structure details
