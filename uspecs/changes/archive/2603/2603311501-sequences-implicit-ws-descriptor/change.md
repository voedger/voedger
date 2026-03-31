---
registered_at: 2026-03-31T14:05:43Z
change_id: 2603311405-sequences-implicit-ws-descriptor
baseline: b173c6a3e631686370782455c70970dcd9e7d74c
archived_at: 2026-03-31T15:01:26Z
---

# Change request: Sequences: enforce implicit workspace descriptor

## Why

If `WorkspaceDescriptor` is not declared, it is impossible to define `wsKind` (which is the `QNameID` of `WorkspaceDescriptor`). `wsKind` is required for sequences, and the existing workaround (see [#3656](https://github.com/voedger/voedger/issues/3656)) is overly complicated.

## What

Enforce an implicit workspace descriptor so sequences always have a valid `wsKind`:

- Parser must create an empty `WorkspaceDescriptor` automatically if one is not declared in the schema for non-abstract workspaces
- AppDef compatibility must prevent a `WorkspaceDescriptor` declared later from using a different name
- Add tests covering the appdef compat scenarios:
  - Explicit descriptor renamed between versions
  - Implicit descriptor replaced by explicit descriptor with a different name
