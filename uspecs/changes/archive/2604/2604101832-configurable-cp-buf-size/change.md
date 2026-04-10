---
registered_at: 2026-04-10T18:00:09Z
change_id: 2604101800-configurable-cp-buf-size
baseline: 5a3c8cb9a41bdae1b923b77ba57dfca6f7cf344c
issue_url: https://untill.atlassian.net/browse/AIR-3544
archived_at: 2026-04-10T18:32:46Z
---

# Change request: Make command processor channel buffer size configurable

## Why

The command processor channel buffer size is hardcoded to 10, causing commands to silently queue and hold goroutines long after clients have disconnected. During outages this leads to hour-long stale command processing. All other processor channels already use buffer size 0 for fail-fast behavior.

See [issue.md](issue.md) for details.

## What

Add a configurable `CommandProcessorChannelBufferSize` field to `VVMConfig`:

- Add `CommandProcessorChannelBufferSize` field (type `uint`) to `VVMConfig` with default value `0`
- Use it instead of the hardcoded `DefaultNumCommandProcessors` when provisioning the command processor channel group in `provide.go`

