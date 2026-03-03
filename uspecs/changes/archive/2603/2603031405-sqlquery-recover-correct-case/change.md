---
registered_at: 2026-03-03T10:51:37Z
change_id: 2603031051-sqlquery-recover-correct-case
baseline: b69e2dfee3306dad9a7f413776939b42ba130ff4
archived_at: 2026-03-03T14:05:17Z
---

# Change request: Recover correct case in parsed SQL query fields and tables

## Why

The vitess SQL parser lowercases identifiers, causing `sys.BLOB` to become `sys.blob` and field names like `Img1` to become `img1`. A rough workaround exists that special-cases "blob" → "BLOB", but a proper solution is needed that recovers the original case from the app schema.

## What

Case-insensitive matching against the app schema for table and field names in SQL queries:

- When `IWorkspace.Type(table)` returns nil, iterate `IWorkspace.Types()` to find a case-insensitive match and use the correctly-cased name
- When `qNameType.(appdef.IWithFields).Field(field)` returns nil, iterate `qNameType.(appdef.IWithFields).Fields()` to find a case-insensitive match and use the correctly-cased name
- Eliminate the existing hack that converts "blob" to "BLOB"
