---
registered_at: 2026-04-19T10:01:25Z
change_id: 2604191001-http-err-log-to-voedger-logger
baseline: 1ba958eead549e01bbab94818220b814d15c3dda
issue_url: https://untill.atlassian.net/browse/AIR-3587
archived_at: 2026-04-19T12:55:47Z
---


# Change request: Redirect http server internal error log to voedger logger

## Why

The `http.Server` internal error log currently writes messages in a format that does not match the slog-based voedger logging standard. As a result, internal messages can be missed when filtering by attributes such as `reqid`. See [issue.md](issue.md) for details.

## What

Route the http server internal error log through the voedger logger:

- Configure `http.Server.ErrorLog` to use a bridge instead of the default logger
- Use `logger.NewStdErrorLogBridge` to produce slog-compatible entries
- Define an appropriate log stage and set of attributes for the bridged messages
