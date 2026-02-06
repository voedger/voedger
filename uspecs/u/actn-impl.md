# uimpl: Change request implementation

## Implementation overview

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

1. Determine which scenario matches from the `Scenarios` section:
   - If all to-do items are checked -> Execute "all to-do items checked" scenario
   - If some to-do items are unchecked -> Execute "some to-do items unchecked" scenario
   - Edge cases (no active change, multiple changes) -> Execute corresponding edge case scenario

2. Within the matching scenario:
   - For "all to-do items checked" scenario: Check conditions in order from Examples table, execute ONLY the first matching action, then stop
   - For "some to-do items unchecked" scenario: Implement and check all unchecked items (stop at Review Item if unchecked)
   - For edge cases: Follow the specific scenario behavior

3. Use definitions and structures from sections below when executing actions

## Definitions

### `Functional design` section does not exist and it is needed

The section is needed if:

- Domain Files exist and define External actors
- Change Request description impacts Functional Design Specifications (only files inside `$specs_folder`)

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
- Last resort: Create Change Technical Design
  - Create ONLY when:
    - Change Request implementation requires new components, functions, modules, or data structures
    - These elements are NOT already described in existing Technical Design Specifications
    - The change does NOT constitute a cohesive feature (otherwise use Feature TD)
    - No existing architecture file is appropriate for the changes
  - The Change Technical Design file should follow the template from `$templates_td`

## Structures

**Important:** The sections (Functional design, Technical design, Construction) contain checkbox lists that **reference** files to create or update. Do not put the actual design content there. The actual content goes into separate files.

### Functional Design Specifications

Ref. Functional Design Specifications in the `$templates` file.

### Technical Design Specifications

Ref. `$templates_td` file.

### Section: Provisioning and configuration

Rules:

- Always prefer to use CLI commands
- For provisioning
  - Make sure that required components are not already installed
  - Specify latest possible stable version, always use web search to find it
  - Detect current OS - provide OS-specific instructions only
  - Group by category
  - Prefer vendor-independent alternatives when available

Example:

```markdown
**Provisioning:**

- [ ] install: Docker 24.0+
  - `winget install Docker.DockerDesktop` or `https://docs.docker.com/get-docker/`

**Configuration:**

- [ ] update: [package.json](../../package.json): Add express web framework
  - `npm install express`

- [ ] update: [tsconfig.json](../../tsconfig.json): Enable strict mode
  - `Manual edit - Set strict: true` 
```  

### Sections: Functional design, Technical design, Construction

These sections contain checkbox lists referencing files to create or update.

Format:

```markdown
- [ ] {action}: [{folder}/{filename}](relative-path)
  - {action}: Description of changes
```

Rules:

- Always use actual relative paths from the Change File to particular file (e.g., ../../specs/domain/myctx/my.feature)
- Use relative paths for both existing files and new files that don't exist yet
- Technical design section
  - Reference Change Technical Design when creating new design documentation
  - Reference existing architecture files (e.g., `../../specs/prod/apps/vvm-orch--arch.md`) when updating them
  - Use templates from `$templates_td` for structure of new files
- Construction section
  - If design sections exist, run `git diff <baseline> -- $specs_folder/` to identify exact spec changes (baseline from Change File frontmatter)
  - List all non-specification files that need to be created or modified, not already covered by other sections
  - Includes source files, tests, documentation, scripts, configuration - any file changes

Example:

```markdown

## Functional design

- [ ] update: [myctx/my.feature](../../specs/domain/myctx/my.feature)
  - add: Branch push validation (main blocked, feature allowed)

## Technical design

- [ ] update: [apps/vvm-orch--arch.md](../../specs/prod/apps/vvm-orch--arch.md)
  - update: Leadership renewal interval documentation (1s instead of TTL/2)

## Construction

- [ ] update: [internal/auth/validator.go](../../../internal/auth/validator.go)
  - fix: null pointer when validating empty email field
- [ ] update: [internal/auth/validator_test.go](../../../internal/auth/validator_test.go)
  - add: Test case for empty email validation
- [ ] update: [README.md](../../../README.md)
  - update: supported Go version to 1.21+
- [ ] create: [scripts/migrate-db.sh](../../../scripts/migrate-db.sh)
  - add: Database migration script for auth schema changes
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
    Then AI Agent implements and checks all unchecked To-Do Items in Implementation Plan
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
