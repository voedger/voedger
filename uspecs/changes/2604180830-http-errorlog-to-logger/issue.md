# AIR-3587: Redirect internal error log of http service to voedger logger

- **Key**: AIR-3587
- **Type**: Sub-task
- **Status**: In Progress
- **Assignee**: d.gribanov@dev.untill.com
- **URL**: <https://untill.atlassian.net/browse/AIR-3587>

## Why

The http server internally could write messages to its error log. The format does not match the slog-based logging standard, so internal messages may be missed when filtering by e.g. `reqid`.

## What

- Use `http.Server.Config.ErrorLog`
- Think out stage and attributes to log with
