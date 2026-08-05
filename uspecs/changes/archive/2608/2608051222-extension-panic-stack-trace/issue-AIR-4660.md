# extengine: provide the stack on extension panic

- URL: https://untill.atlassian.net/browse/AIR-4660
- ID: AIR-4660
- State: in-progress
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov

## Why

panic in extension → have no idea where it happened

```
extension <name> panic: runtime error: invalid memory address or nil pointer dereference
```

## What

include stack trace in the log message on panic
