# voedger: derive architecture from app context

- URL: https://untill.atlassian.net/browse/AIR-4154
- ID: AIR-4154
- State: in-progress
- Author: Denis Gribanov
- Assignees: Denis Gribanov
- Labels: none

## Why

missing architecture for App context mentioned in [domain.md](https://github.com/voedger/voedger/blob/main/uspecs/specs/prod/domain.md)

## What

- derive architecture from App context considering exisitng documentation and codebase. Devide the architecture to deployment and processing (processors).
  - For deployment: consider vsql ddl, deployment desriptors, app partitions engine, bootstrap, protection against uncompatible schema and deployment descriptor change.
  - For processing: consider all processors pipelines, authnz, acl, response sending, app partitions engine.
