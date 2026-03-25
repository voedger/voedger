---
registered_at: 2026-03-25T11:41:25Z
change_id: 2603251141-qp-reject-unexpected-fields
baseline: 601cd2f9bc5444d30077737c7a48f7c7196d8219
archived_at: 2026-03-25T13:37:12Z
---

# Change request: return 400 bad request on unexpected request fields in QPv1, QPv2, and CP

## Why

QPv1, QPv2, and the command processor silently ignore unknown fields in requests. This makes it hard to detect client-side typos and integration bugs, as malformed requests succeed without any indication that fields were ignored.

## What

Return HTTP 400 bad request when a request contains unexpected fields:

- QPv1: reject unknown fields inside `args` and unknown top-level fields (alongside `args`, `elements`, `filters`, `orderBy`, `count`, `startFrom`)
- QPv2: reject unknown fields inside `args` URL parameter
- Command processor: reject unknown top-level fields other than `args`, `unloggedArgs`, `cuds`
- All processors: if the function/command declares no params (`argsType` is nil) but the client sends non-empty `args` (or `unloggedArgs` for CP), return 400
- Return a descriptive error message identifying the unexpected field(s)
