---
registered_at: 2026-04-18T08:30:59Z
change_id: 2604180830-http-errorlog-to-logger
baseline: 9eddaed6a696c253601bdaf175d58dc9e2f50c26
issue_url: https://untill.atlassian.net/browse/AIR-3587
---


# Change request: Redirect http server internal error log to voedger logger

## Why

The http server writes internal messages to its own error log, whose format does not match the slog-based logging standard used by voedger. As a result, internal http server messages can be missed when filtering logs by attributes such as `reqid`. See [issue.md](issue.md) for details.

## What

Route http server internal error output through the voedger logger:

- Configure `http.Server.ErrorLog` so its output is forwarded to the voedger logger
- Define the log stage and attributes used for these forwarded messages
