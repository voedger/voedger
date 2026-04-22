---
registered_at: 2026-03-26T13:39:30Z
change_id: 2603261339-elim-mock-buckets
baseline: 43fecfd41a6c0244a700f0579fd4370e522e60fa
issue_url: https://untill.atlassian.net/browse/AIR-3414
archived_at: 2026-03-27T12:58:55Z
---

# Change request: Eliminate vit.MockBuckets and replace with real rate limit testing

## Why

The current integration tests use `vit.MockBuckets` to simulate rate limiting behavior, which bypasses the actual rate limiting mechanism. This makes the tests less realistic and hides potential issues with real bucket depletion. See [issue.md](issue.md) for details.

## What

Replace mock-based rate limit testing with realistic integration tests:

- Remove `vit.MockBuckets` and its usage from the test infrastructure
- Update the affected integration test with an approach that sends additional HTTP requests with time advances between them to force actual bucket depletion
