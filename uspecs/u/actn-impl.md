# uimpl: Change request implementation

## Implementation overview

Rules:

- Strictly follow the definitions from `uspecs/u/concepts.md` and `uspecs/u/conf.md`
  - Use file name patters from there, not from the codebase

Parameters:

- Input
  - Files in Active Change Folder, including Implementation Plan
- Output
  - Implementation Plan

Implementation Plan is created/processed according to the rules described in the `Implementation scenarios` section below. Some abstract conditions and data structures are explained in subsequent sections.

Rules:

- Each uimpl command execution must:
  - Strictly follow the Background rules and definitions
  - Evaluate scenarios in order
  - Execute the FIRST matching scenario
  - STOP after executing one scenario
  - Require another uimpl invocation to proceed to next scenario

## Conditions

- "affected" (e.g. "If Functional Design Specifications affected by Change Request")
  - Means specifications should be created/updated/deleted
  - Particular actions are inferred from Active Change Request description and completed to-do items in Implementation Plan
- "If Technical Design Specifications affected": follow general "affected" definition plus:
  - Change Technical Design file should be created if there are some flows that are not covered by other Technical Design Specifications files

## Structures

### Functional Design Specifications

Ref. Functional Design Specifications in the `$templates` file.

### Technical Design Specifications

Ref. `$templates_td` file.

### Sections: Functional design, Technical design, Construction

Format:

```markdown
- [ ] action: [{folder}/{filename}](relative-path)
  - Description of changes
```

Rules:

- Always use actual relative paths from the Change File  to particular file (e.g., ../../specs/domain/myctx/my.feature)
- Use relative paths for both existing files and new files that don't exist yet
- Functional design section
  - Use "Cover [behavior]" instead of "Add scenario" or "Include scenario"
    - Example: "Cover branch push validation (main blocked, feature allowed)"
- Construction section
  - List all non-specification files that need to be created or modified, not already covered by other sections
  - Includes source files, tests, documentation, scripts, configuration - any file changes

Example:

```markdown

## Functional design

- [ ] update: [myctx/my.feature](../../specs/domain/myctx/my.feature)
  - Cover branch push validation (main blocked, feature allowed)

### Construction

- [ ] update: [internal/auth/validator.go](../../../internal/auth/validator.go)
  - Fix null pointer when validating empty email field
- [ ] update: [internal/auth/validator_test.go](../../../internal/auth/validator_test.go)
  - Add test case for empty email validation
- [ ] update: [README.md](../../../README.md)
  - Update supported Go version to 1.21+
- [ ] create: [scripts/migrate-db.sh](../../../scripts/migrate-db.sh)
  - Database migration script for auth schema changes
```

### Section: Quick start

- When to include: New features, APIs, CLI commands, or configuration changes that users need to learn how to use
- Skip if: Internal refactoring, bug fixes, or changes with no user-facing impact
- Show how to use the feature with minimal examples

#### Example

Run migration with dry-run mode:

```bash
./scripts/migrate-db.sh --dry-run up
```

Generate report with date filter:

```bash
npm run report -- --from=2024-01-01 --to=2024-12-31
```

---

## Implementation scenarios

```gherkin

Feature: Change request implementation

  Engineer implements change request

  Background:

    Given Implementation Plan Full Structure
      | Section                                       | Presence conditions                                                            |
      | # Implementation plan: {Change request title} | Always                                                                         |
      | ## Functional design                          | If Functional Design Specifications affected by Change Request                 |
      | ## System configuration                       | If System configuration affected by Change Request                             |
      | ## Technical design                           | If Technical Design Specifications affected                                    |
      | ## Construction                               | If non-specification files affected                                            |
      | ## Quick start                                | When new features, APIs, CLI commands, or configuration changes are introduced |
    And Available To-Do Items are defined as follows
      | Condition                                        | Available To-Do Items                            |
      | `[ ] Review` or `[ ] review` item exists         | All unchecked items that precede the Review Item |
      | `[ ] Review` or `[ ] review` item does NOT exist | All unchecked items                              |      

  Scenario: Create Implementation Plan when Functional Design Specifications affected
    Given Active Change Request affects Functional Design Specifications
    And Implementation Plan does not exist
    When Engineer runs uspecs-impl command
    Then Implementation Plan is created
    And Implementation Plan contains ONLY the following sections
      | Section                                       | Presence conditions |
      | # Implementation plan: {Change request title} | Always              |
      | ## Functional design                          | Always              |
    And Functional design section contains unchecked to-do items

  Scenario: Create Implementation Plan when Functional Design Specifications NOT affected
    Given Active Change Request does NOT affect Functional Design Specifications
    And Implementation Plan does not exist
    When Engineer runs uspecs-impl command
    Then Implementation Plan is created and has all sections from Implementation Plan Full Structure whose Presence conditions are met
    And all created sections contain unchecked to-do items

  Scenario: Apply to-do items from Functional design section
    Given Implementation Plan exists
    And it contains "## Functional design" section
    And some to-do items in "## Functional design" section are unchecked
    When Engineer runs uspecs-impl command
    Then all Available To-Do Items in Functional design are implemented and checked
    And missing sections from Implementation Plan Full Structure are created according to Presence conditions
    And newly created sections contain unchecked to-do items
    And no other to-do items are implemented/checked

  Scenario: Apply to-do items from non-Functional design sections
    Given Implementation Plan exists
    And (Functional design section does not exist) OR (Functional design section is complete)
    And some other sections are incomplete
    When Engineer runs uspecs-impl command
    Then all Available To-Do Items in Implementation Plan are implemented and checked

  Rule: Edge case scenarios

    Scenario: Unchecked Review Item
      Given Implementation Plan exists
      And there are no Available To-Do Items
      And unchecked Review Item exists
      When Engineer runs uspecs-impl command
      Then AI Agent displays message "Please review the implementation plan before proceeding and check the [ ] Review item"

    Scenario: Implementation Plan complete - no action needed
      Given Implementation Plan exists
      And all to-do items in all sections are checked
      When Engineer runs uspecs-impl command
      Then AI Agent displays message "Implementation Plan is complete"
      And no changes are made to Implementation Plan

    Scenario: No Active Change Request exists
      Given no Active Change Request exists in changes folder
      When Engineer runs uspecs-impl command
      Then AI Agent displays error "No Active Change Request found"
      And Implementation Plan is not created

    Scenario: Multiple Active Change Requests exist
      Given multiple Active Change Requests exist in changes folder
      And AI Agent may not infer from the context which one to use
      When Engineer runs uspecs-impl command
      Then AI Agent displays error "Multiple Active Change Requests found. Please specify change folder"
      And lists all Active Change Request folders
```
