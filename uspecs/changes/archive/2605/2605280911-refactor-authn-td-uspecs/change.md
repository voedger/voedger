---
change_id: 2605280718-refactor-authn-td-uspecs
type: refactor
scope: auth
---
# Change request: Authn technical design uspecs alignment

## Why

The authn feature technical design needs to match current uspecs rules so reviewers can distinguish technical design, functional scenarios, context architecture, and implementation traceability. This keeps authentication design documentation easier to validate without changing runtime behavior.

## What

- No authentication behavior changes: public authn API status codes, response fields, and principal token semantics are preserved.
- The authn feature technical design conforms to uspecs expectations for Feature Technical Design artifacts.
- Existing traceability between authn scenarios, technical components, implementation paths, and coverage remains reviewable.

## Technical design

- [x] update: [prod/auth/authn--td.md](../../../../specs/prod/auth/authn--td.md)
  - refactor: align Feature Technical Design structure and content with current uspecs rules
  - fix: separate or remove content that belongs to functional design, context architecture, or implementation notes
  - update: preserve traceability to authn scenarios, auth architecture, code paths, and test coverage without changing documented authn behavior
