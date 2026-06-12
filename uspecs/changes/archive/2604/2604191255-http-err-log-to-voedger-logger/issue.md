# AIR-3587: redirect internal error log of http service to voedger logger

- **Key**: [AIR-3587](https://untill.atlassian.net/browse/AIR-3587)
- **Type**: Sub-task
- **Status**: In Progress
- **Assignee**: d.gribanov@dev.untill.com

## Why

The http server internal may write messages to its error log. The format does not match the slog-based logging standard, so we may miss some internal messages when filtering by attributes such as `reqid`.

## What

- Use `http.Server.ErrorLog`
- Use `logger.NewStdErrorLogBridge`
- Think out the stage and attributes to log with
