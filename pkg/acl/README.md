# acl

- Simple ACL
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

```

## Context

- “Principal P from Workspace W is [Allowed/Denied] Operation O on Resources matching ResourcePattern RP” [design/authnz](../../design/authnz/README.md)

## Principles

acl is a simple, base implementation of the above context. 

- It does not support tags
- It does not support resource inheritance (like granting access to a table gives same access to all columns - no, it is not supported)
- It does not distinct between operations

All these things should be implemented in a separate package, which will use acl as a base.

## Functional Design

1. Build IACL
  - For each new AppDef for every Workspace type get an `IACLBuilder` using `NewACLBuilder()` and build `IACL`
2. Use `IACL`
  - Get a Request
  - Determine Workspace type and Principals
  - Determine all Principal's Roles, including inherited Roles (if any)
  - Determine all Resources (e.g. each table field is a separate resource)
  - Query `IACL` for each (Principal, Resource, Operation) triplet
    - To check access to a table query Table resource first, if it is not ok query each Field resource