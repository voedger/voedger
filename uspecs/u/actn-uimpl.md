# Action: Change request implementation

## Overview

Execute implementation plan items for an Active Change Request, one scenario at a time.

## Instructions

Rules:

- Strictly follow the definitions from `uspecs/u/concepts.md` and `uspecs/u/conf.md`
  - Use file name patterns from there, not from the codebase
- Critical: Only one scenario must be executed per command run. After executing the action, the AI Agent must stop further processing and wait for the next command.
- All relative links must be correct paths from the file being edited to the target file. Verify that relative paths resolve to the intended file before writing them.

Parameters:

- Input
  - Files in Active Change Folder, including Implementation Plan
- Output
  - Implementation Plan

Flow:

- Read all files present in the Active Change Folder (change.md, impl.md, issue.md, decs.md, how.md and others - whichever exist) before any further processing
- Determine which scenario matches from the `Scenarios` section:
  - If all to-do items are checked -> Execute "all to-do items checked" scenario
  - If some to-do items are unchecked -> Execute "some to-do items unchecked" scenario
  - Edge cases (no active change, multiple changes) -> Execute corresponding edge case scenario
- Within the matching scenario:
  - For "all to-do items checked" scenario: Check conditions in order from Examples table, execute ONLY the first matching action, then stop
  - For "some to-do items unchecked" scenario: Implement each unchecked item and check it immediately after implementation (stop at Review Item if unchecked)
  - For edge cases: Follow the specific scenario behavior

Use definitions from sections below and structures from `{templates_folder}/tmpl-impl.md` when executing actions

## Definitions

### `Functional design` section does not exist and it is needed

The section is needed if:

- Domain Files exist and define External actors
- Change Request description impacts Functional Design Specifications (only files inside `{specs_folder}`)

Impact:

- Domain File
  - Change Request implies introducing new domains
  - CRUD for concepts, actors, etc.
- Scenarios File
  - Change Request implies introducing new features
  - CRUD for existing scenarios
- Requirements File
  - Change Request implies introducing new Requirements File
    - Try to fit requirements into Scenarios File first
  - CRUD for requirements

### `Provisioning and configuration` section does not exist and it is needed

The section is needed if Change Request description implies:

- Provisioning
  - Adding, updating, or removing project dependencies
  - Installing new tools, SDKs, or applications system-wide
  - Setting up external services (databases, message queues, etc.)
- Configuration (including Project Configuration Files)
  - Modifying environment variables or secrets
  - Updating build or deployment settings
  - Changing linter, formatter, or test runner configs
  - Adjusting CI/CD pipeline definitions

### `Technical design` section does not exist and it is needed

Follow this decision hierarchy (in order):

- First priority: Update existing Technical Design Specifications
  - Search for existing architecture and technical design files that cover the affected components
  - Update when the change request affects design elements already documented in:
    - Domain Architecture
    - Domain Subsystem Architecture
    - Context
    - Feature Technical Design
- Second priority: Create Feature Technical Design
  - Create when:
    - Explicitly requested by the user
    - OR Codebase follows a pattern of Feature Technical Design per feature (maintain consistency)

## Scenarios

```gherkin
Feature: Implementation plan management

  Engineer implements change request

  Scenario Outline: Execute uspecs-impl command, all to-do items checked
    Given all to-do items in Implementation Plan are checked
    When Engineer runs uspecs-impl command
    Then Implementation Plan is created if not existing
    And AI Agent executes only one (the first available) <action> depending on <condition>
    Examples:
      | condition                                                                | action                                                                                |
      | `Functional design` section does not exist and it is needed              | Create `Functional design` section with checkbox items referencing spec files         |
      | `Provisioning and configuration` section does not exist and it is needed | Create `Provisioning and configuration` section with installation/configuration steps |
      | `Technical design` section does not exist and it is needed               | Create `Technical design` section with checkbox items referencing design files        |
      | `Construction` section does not exist and it is needed                   | Create `Construction` section and optionally `Quick start` section                    |
      | Nothing of the above                                                     | Display message "No action needed"                                                    |
    And AI Agent stops execution after performing the action

  Scenario: Execute uspecs-impl command, some to-do items unchecked
    Given some to-do items in Implementation Plan are unchecked
    When Engineer runs uspecs-impl command
    Then AI Agent implements each unchecked To-Do Item and checks it immediately after implementation
    But it stops on Review Item if it is unchecked


  Rule: Edge cases

    Scenario: No Active Change Request exists
      Given no Active Change Request exists in changes folder
      When Engineer runs uspecs-impl command
      Then AI Agent displays error "No Active Change Request found"
      And Implementation Plan is not created

    Scenario: Multiple Active Change Requests exist
      Given multiple Active Change Requests exist in changes folder
      And AI Agent may not infer from the context which one to use
      When Engineer runs uspecs-impl command
      Then AI Agent displays error "Multiple Active Change Requests found. Please specify which one to use"
      And lists all Active Change Request folders
      And allows Engineer to select one and proceed
```
