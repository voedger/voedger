# Voedger Design

- [Current design](https://internals.voedger.io/)
- [Historical design](https://github.com/voedger/voedger/blob/2b8829e4b053d77daa0106556b444f97a0e41406/design/README.md)

Non-migrated yet parts see below.

## Detailed design

Functional Design

- [Orchestration](orchestration/README.md)

Non-Functional Reqiurements, aka Quality Attributes, Quality Requirements, Qualities

- [Consistency](consistency)
- Security
  - Encryption: [HTTPS + ACME](https-acme)
  - [Authentication and Authorization (AuthNZ)](authnz)
- TBD: Maintainability, Perfomance, Portability, Usability ([ISO 25010](https://iso25000.com/index.php/en/iso-25000-standards/iso-25010), System and software quality models)

Technical Design

- [Bus](https://github.com/heeus/core/tree/main/ibus)
- [State](state/README.md)
- [Command Processor](commandprocessor/README.md)
- [Query Processor](queryprocessor/README.md)
- [Projectors](projectors/README.md)
- [Storage](storage/README.md)

## Misc

DevOps

- [Building](building)

Previous incompatible versions

- [Prior 2023-09-13](https://github.com/voedger/voedger/blob/7f9ff095d66e390028abe9037806dcd28bde5d9e/design/README.md)
