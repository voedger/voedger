---
registered_at: 2026-04-15T11:31:57Z
change_id: 2604151131-fix-retry-body-reader
baseline: acfb394b2f2c941f0bd173b354b4e83cda8b894d
issue_url: https://untill.atlassian.net/browse/AIR-3578
archived_at: 2026-04-15T11:44:25Z
---

# Change request: Fix body loss on retry in HTTPClient

## Why

When `IHTTPClient.ReqReader` is used with an `io.Reader` body and a retry occurs, the body is lost on subsequent attempts because the `io.Reader` has already been consumed. This results in empty request bodies on retries.

See [issue.md](issue.md) for details.

## What

Fix the retry handler in `implIHTTPClient` to preserve the request body across retries:

- Implement a test that reproduces the body-loss-on-retry problem
- Prevent body loss by buffering the reader content before the retry loop
