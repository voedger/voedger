# uhow: Change Request clarification

## Overview

Rules:

- Strictly follow the definitions from `uspecs/u/concepts.md` and `uspecs/u/conf.md`
- Web search is optional - performed only when Engineer specifies --web flag or mentions using web search, or when questions involve choosing between technologies/algorithms/patterns
- Questions must use numbered list format (1, 2, 3)
- First option (1) is the recommended choice

Parameters:

- Input
  - Existing How File (if any)
  - --web flag (optional) - Forces web search to be performed
  - Natural language mention of web search (optional) - Phrases like "with web search", "use web search", "search web"
- Output
  - Updated How File with new section
  - Ref. `uspecs/u/templates-how.md`

Implementation Plan is created/processed according to the rules described in the `Scenarios` section below using definitions and structures from the sections below.

## Definitions

### `Functional design` section does not exist in How File and needed

The section is needed if:

- Engineer asks for it
- Change Request has functional aspects to clarify (actors, domains, features)

### `Technical design` section does not exist in How File and needed

The section is needed if:

- Engineer asks for it
- Change Request has technical aspects to clarify (architecture, tech stack, components)

## Scenarios

```gherkin
Feature: Change Request clarification

  Engineer clarifies and brainstorms Change Request functional and technical design

  Background:
    Given AI Agent asks three questions in time
    And Web search is performed when Engineer requests it or when questions involve technology/algorithm/pattern choices

  Scenario Outline: Execute uspecs-how command
    When Engineer runs uspecs-how command
    Then AI Agent executes only one (the first available) <action> depending on <condition>
    Examples:
      | condition                                                          | action                                                               |
      | `Functional design` section does not exist in How File and needed  | Ask questions about functional design and write answers to How File  |
      | `Technical design` section does not exist in How File and needed   | Ask questions about technical design and write answers to How File   |
      | Nothing of the above                                               | Display message "No action needed"                                   |
```
