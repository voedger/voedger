---
registered_at: 2026-02-27T07:03:05Z
change_id: 2602270703-logger-ctx-attrs
baseline: 96d70eed79c988e4645fac2c105e06b0a30875bd
archived_at: 2026-02-27T12:26:47Z
---

# Change request: Implement VerboseCtx and ErrorCtx in logger package

## Why

Callers often need to attach structured attributes (e.g. request-id, workspace, app name) to log entries without threading them explicitly through every log call. Storing such attributes in `context.Context` under a dedicated key and reading them inside `VerboseCtx`/`ErrorCtx` makes this transparent and consistent.

## What

Add context-aware logging functions to the `logger` package:

- `VerboseCtx(ctx context.Context, args ...interface{})` – logs at Verbose level, enriching the entry with `slog.Attr` values stored in `ctx`
- `ErrorCtx(ctx context.Context, args ...interface{})` – logs at Error level, enriching the entry with `slog.Attr` values stored in `ctx`
and so on.

Add standard slog atributes:

- vapp={string}
  - examples: untill.fiscalcloud, unitll.airsbp
- feat={string}
  - examples: magicmenu
- reqid={requestIDNumber}
- wsid={wsidNumber}
- extension={string}
  - exmpales: c.sys.UploadBLOBHelper

Supporting additions:

- `logger.WithContextAttrs(ctx context.Context, name string, value any)` – returns a new context with the given attributes appended to any already stored

## How

- use `slog.NewTextHandler` as log message engine
- keep use log levels from `logger` package
- store set of attributes in context under a dedicated key with multithreading protection, e.g. `SyncMap`
