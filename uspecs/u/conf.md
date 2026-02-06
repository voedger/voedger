# Configuration

## Parameters

- specs_folder: uspecs/specs
- changes_folder: uspecs/changes
- changes_archive: `$changes_folder/archive`
- templates: uspecs/u/templates.md
- templates_td: uspecs/u/template-td.md
  - Template for some Technical Design Specifications

## Artifacts

- Change Folder: a folder containing change.md and other optional artifacts that documents a proposed or completed change to the project. Named with format ymdHM-{change-name}
  - ymdHM format specification:
    - y: 2-digit year (e.g., 26 for 2026, 27 for 2027)
    - m: 2-digit month (01-12)
    - d: 2-digit day (01-31)
    - H: 2-digit hour (00-23)
    - M: 2-digit minute (00-59)
    - Must use current local date
    - Example: For 2006-01-02 15:04, use prefix "2601021504"
  - change-name format specification:
    - Use kebab-case
    - Keep length under 40 characters (ideal: 15-30 characters)
    - Focus on core action or feature, avoid redundant words
    - Use abbreviations when appropriate to reduce length
    - Examples: `remove-uspecs-prefix`, `fetch-issue-to-change`, `alpha-code-bp3-endpoints`
  - Can be either Active (in `$changes_folder`) or Archived (in `$changes_archive`)
  - Active Change Folder files describe Active Change Request and its implementation
- Change Folder System Artifacts
  - Change File: `change.md`
  - Issue File: `issue.md`
    - Describes the issue that prompted the change, if applicable
  - Change Technical Design: `td.md`
  - Implementation Plan: `impl.md`
  - How File: `how.md`
    - Clarification and brainstorming about Change Request functional and technical design
- Domain Folder: `$specs_folder/{domain}/`
- Context Folder: `$specs_folder/{domain}/{context-id}/`
  - Contains
    - zero or many `Scenarios File`, `Requirements File`, `Technical Design File`
    - zero or one `Architecture File`
- Functional Design Specifications
  - Domain File: `$specs_folder/{domain}/{domain}--domain.md`
  - Feature Files
    - Scenarios File: `{context-folder}/{feature}.feature`
    - Requirements File: `{context-folder}/{feature}--reqs.md`
      - Requirements that do not fit into Scenarios File
- Project Configuration File
  - Files like `go.mod`, `go.work`, `package.json`, `requirements.txt`, `pubspec.yaml` etc. that define project dependencies and configuration
- Technical Design Specifications
  - Domain Technology
    - Per domain: `$specs_folder/{domain}/{domain}--tech.md`
    - Defines tech stack, architecture patterns etc., UI/UX guidelines etc.
  - Domain Architecture
    - `$specs_folder/{domain}/{domain}--arch.md`
  - Domain Subsystem Architecture
     or `$specs_folder/{domain}/{subsystem}--arch.md`
  - Context Architecture
    - `{context-folder}/{context}--arch.md`
  - Context Subsystem Architecture
     or `{context-folder}/{subsystem}--arch.md`
  - Feature Technical Design
    - Per feature: `{context-folder}/{feature}--td.md`
  - Change Technical Design

### Elements

Review Item, one of:

```markdown
- [ ] Review
- [ ] review
- Review
- review
```

## Definitions

- Change ID: name of Active Change Folder (without path)
