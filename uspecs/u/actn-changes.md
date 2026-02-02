# Changes management

Always read `uspecs/u/concepts.md` and `uspecs/u/conf.md`  before proceeding any instructions.

## Create change (new change)

Rules:

- Strictly follow the definitions from `uspecs/u/concepts.md` and `uspecs/u/conf.md`
- Strictly use definitions from the Definitions section
- When you fetch issue content
  - Convert it to rich markdown format suitable for Issue File

Parameters:

- Input
  - Change description
- Output
  - Active Change Folder with Change File
  - Issue File (if issue reference provided)

Flow:

- Follow instructions from Scenarios section
- Fail fast if Active Change Folder cannot be created or already exists
- Create Active Change Folder with Change File:
  - If issue reference provided:
    - Try to fetch issue content
    - If fetch succeeds:
      - Save fetched content to Issue File (issue.md)
      - Create Change File following Change File Template 1
      - In Why section, reference Issue File: `See [issue.md](issue.md) for details.`
    - If fetch fails:
      - Create Change File following Change File Template 1 (no Issue File, no reference)
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
  - Folder moved to `$changes_folder/archive/yymm/` with archived_at metadata (if all items are checked or cancelled)

Flow:

- Identify Active Change Folder to archive, if unclear, ask user to specify folder name
- Execute `bash uspecs/u/scripts/uspecs.sh change archive <absolute-path-to-change-folder>`
- Analyze output, show to user and STOP

## Scenarios

```gherkin
Feature: Create change request
  Engineer asks AI Agent to create change request

  Scenario: Create change request without issue reference
    When Engineer asks AI Agent to create change request without issue reference
    Then Active Change Folder is created with Change File
    And Change File follows Change File Template 1
    And Frontmatter does not have issue_url value

  Scenario Outline: Create change request with issue reference
    Given AI Agent <ability> to fetch issue content from the referenced issue URL
    When Engineer asks AI Agent to create change request with issue reference
    Then Active Change Folder is created with Change File
    And Change File follows Change File Template 1
    And Frontmatter has issue_url value set to the referenced issue URL
    And Issue File <issue-file-created-and-contains> the fetched issue contents in markdown format
    And Change File <references> Issue File in the Why section
    Examples:
      | ability                        | references                    | issue-file-created-and-contains |
      | has ability to fetch content   | references Issue File         | contains fetched issue content  |
      | does not have ability to fetch | does not reference Issue File | is not created                  |
```
