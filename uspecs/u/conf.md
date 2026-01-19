# Configuration

## Parameters

- specs_folder: uspecs/specs
- changes_folder: uspecs/changes
- changes_archive: `$changes_folder/archive`
- templates: uspecs/u/templates.md
- templates_td: uspecs/u/template-td.md
  - Template for some Technical Design Specifications

## Artifacts

- Change Folder: a folder containing change.md and other optional artifacts that documents a proposed or completed change to the project. Named with format YYMMDD-{change-name}
  - Can be either Active (in `$changes_folder`) or Archived (in `$changes_archive`)
  - Active Change Folder files describe Active Change Request and its implementation
- Change Folder System Artifacts
  - Change File: `change.md`
  - Change Technical Design: `change-td.md`
  - Implementation Plan: `change-impl.md`
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
- Technical Design Specifications
  - Project Configuration File: files like `go.mod`, `go.work`, `package.json`, `requirements.txt`, `pubspec.yaml` etc. that define project dependencies and configuration
  - Domain Technology
    - Per domain: `$specs_folder/{domain}/{domain}--tech.md`
    - Defines tech stack, architecture patterns etc., UI/UX guidelines etc.
  - Domain Architecture
    - `$specs_folder/{domain}/{domain}--arch.md`
  - Domain Subsystem Architecture
     or `$specs_folder/{domain}/{subsystem}--arch.md`
  - Context Architecture
    - `{context-folder}/arch.md`
  - Context Subsystem Architecture
     or `{context-folder}/{subsystem}--arch.md`
  - Feature Technical Design
    - Per feature: `{context-folder}/{feature}--td.md`
  - Change Technical Design

## Definitions

- Change ID: name of Active Change Folder (without path)
