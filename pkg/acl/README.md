# acl

- Access Control List implementation
- @author Maxim Geraskin

## Motivation

- [acl: (ws, principals, resource, operation) to bool](https://github.com/voedger/voedger/issues/949)

```sql
    -- sys
    ROLE System;
    ROLE Admin;
    ROLE LocationUser;
    ROLE LocationManager;
    ROLE Application; -- Projector is executed with this role

    -- Grants declared only within workspace

    GRANT ALL ON ALL TABLES WITH TAG BackofficeTag TO LocationManager;
    GRANT INSERT,UPDATE ON ALL TABLES WITH TAG BackofficeTag TO LocationUser;
    GRANT SELECT ON TABLE Orders TO LocationUser;
    GRANT UPDATE (CloseDatetime, Client) ON TABLE Bill TO LocationUser;
    GRANT EXECUTE ON COMMAND Orders TO LocationUser;
    GRANT EXECUTE ON QUERY Query1 TO LocationUser;
    GRANT EXECUTE ON ALL QUERIES WITH TAG PosTag TO LocationUser;

    GRANT INSERT ON WORKSPACE Workspace1 TO Role1;
```

## Context

- “Principal P from Workspace W is [Allowed/Denied] Operation O on Resources matching ResourcePattern RP” [design/authnz](../../design/authnz/README.md)

## Functional Design

1. For each new AppDef for every Workspace type get an `IACLBuilder` using `NewACLBuilder()` and build `IACL`
2. Use `IACL`: get a request, determine Workspace type, find `IACL` for this Workspace type, check if request is allowed