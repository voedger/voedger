# Changes management

Always read `uspecs/u/concepts.md` and `uspecs/u/conf.md`  before proceeding any instructions.

## Create change (new change)

Rules:

- Strictly follow the definitions from `uspecs/u/concepts.md` and `uspecs/u/conf.md`
- Strictly use definitions from the Definitions section
- When you fetch issue content
  - Convert it to rich markdown format suitable for inclusion in Change File
  - Change File ## Description section must be as close as possible to the issue description

Parameters:

- Input
  - Change description
- Output
  - Active Change Folder with Change File

Flow:

- Follow instructions from Scenarios section
- Fail fast if Active Change Folder cannot be created or already exists
- Create Active Change Folder with Change File:
  - If issue reference provided: fetch issue content and include in Change File body
  - If no issue reference: create Change File following Change File Template 1
- Add frontmatter metadata:
  - If issue reference was provided: `bash uspecs/u/scripts/uspecs.sh change frontmatter <absolute-path-to-change-file> --issue-url <issue-url>`
  - If no issue reference: `bash uspecs/u/scripts/uspecs.sh change frontmatter <absolute-path-to-change-file>`
- STOP after creating the Change File with frontmatter
- Show user what was created

## Definitions

- Change File Template 1: ref. `uspecs/u/templates.md`

## Archive change

Parameters:

- Input
  - Active Change Folder path
- Output
  - Folder moved to `$changes_folder/archive` with archived_at metadata (if no uncompleted items)

Flow:

- Identify Active Change Folder to archive, if unclear, ask user to specify folder name
- Execute `bash uspecs/u/scripts/uspecs.sh change archive <absolute-path-to-change-folder>`
- Analyze output, show to user and STOP

## Scenarios

```gherkin
Feature: Create change request
  Developer asks AI Agent to create change request

  Scenario: Create change request without issue reference
    When Developer asks AI Agent to create change request without issue reference
    Then Active Change Folder is created with Change File
    And Change File follows Change File Template 1
    And Frontmatter does not have issue_url value

  Scenario Outline: Create change request with issue reference
    When Developer asks AI Agent to create change request with issue reference
    Then Active Change Folder is created with Change File
    And AI Agent fetches the referenced issue content and convert it to markdown
    And Change File does not follow Change File Template 1
    And Change File contains the fetched issue contents in markdown format
    And Frontmatter has issue_url value set to the referenced issue URL
```
