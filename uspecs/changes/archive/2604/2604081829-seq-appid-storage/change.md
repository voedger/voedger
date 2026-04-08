---
registered_at: 2026-04-08T14:15:21Z
change_id: 2604081415-seq-appid-storage
baseline: 789bd332e92912b1776a08371572c9973fe9452d
issue_url: https://untill.atlassian.net/browse/AIR-3530
archived_at: 2026-04-08T18:29:10Z
---

# Change request: Consider AppID in VVM sequence storage

## Why

IVVMSeqStorageAdapter stores sequence numbers without considering AppID, causing different applications sharing the same partition to receive sequential numbers (1, 2) instead of independent numbering (1, 1). See [issue.md](issue.md) for details.

## What

Updated VVM sequence storage adapter to scope sequence numbers by AppID:

- `IVVMSeqStorageAdapter.GetNumber` and `PutNumbers` already accept `appID` — storage keys include AppID
- VVM storage adapter implementation encodes AppID into PLog offset and sequence number storage keys
- Added comprehensive tests verifying that different apps on the same partition get independent sequence numbers
- Updated sequences architecture documentation with VVM storage key structure details
