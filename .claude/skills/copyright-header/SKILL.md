---
name: copyright-header
description: Use this skill whenever creating a new source file in this repository with extension `.go` (including `_test.go`) or `.vsql`. Emits the canonical copyright header with the current calendar year, the legal entity `unTill Software Development Group B.V.`, and an `@author` line populated from the current Git user (`git config user.name`). Does not apply to files that already exist; do not rewrite committed headers.
user-invocable: false
---

## When to use

- Creating any new `.go` file (production or `_test.go`)
- Creating any new `.vsql` file

## When not to use

- Editing an existing file - leave its header untouched even if it uses the legacy wording
- Generated files that already carry a generator-specific header (e.g. `// Code generated ... DO NOT EDIT.`) - keep the generator header

## Procedure

1. Resolve the current calendar year (`<YEAR>`)
2. Resolve `<AUTHOR>` by running `git config user.name`; if it is empty, fall back to the configured commit identity in the active environment
3. Choose the template by file extension:
   - `.go` -> Go template
   - `.vsql` -> VSQL template
4. Emit the header as the first lines of the file, followed by one blank line, then the package / SQL statements
5. Do not add a trailing period, do not localize the entity name, do not abbreviate it

## Templates

### Go (`.go`, `_test.go`)

```go
/*
 * Copyright (c) <YEAR>-present unTill Software Development Group B.V.
 * @author <AUTHOR>
 */

package <package>
```

### VSQL (`.vsql`)

```sql
-- Copyright (c) <YEAR>-present unTill Software Development Group B.V.
-- @author <AUTHOR>

<first SQL statement>
```

## Rules

- The year is fixed at file creation; it is not bumped on later edits
- Exactly one `@author` line per new file (the creator); additional authors may be appended on subsequent edits as separate `@author` lines, but that is out of scope for this skill
- Never substitute `unTill Pro, Ltd.`, `Sigma-Soft, Ltd.`, or any other historical entity - those wordings remain only on files that already carry them
- Never wrap the header in an extra blank comment line above or below the `/*` / `*/`
- For Go, the header must precede the `package` clause with exactly one blank line between them
- For VSQL, the header must precede the first statement with exactly one blank line between them

## Anti-patterns

bad -- wrong entity, missing author:

```go
/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package foo
```

bad -- year fixed to a past value on a file created today:

```go
/*
 * Copyright (c) 2021-present unTill Software Development Group B.V.
 * @author Jane Doe
 */
```

bad -- VSQL using a block comment instead of line comments:

```sql
/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Jane Doe
 */

ABSTRACT WORKSPACE Workspace ();
```

good -- Go, current year, current Git user:

```go
/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Jane Doe
 */

package foo
```

good -- VSQL, current year, current Git user:

```sql
-- Copyright (c) 2026-present unTill Software Development Group B.V.
-- @author Jane Doe

ABSTRACT WORKSPACE Workspace ();
```
