# Action: Implementation approach

## Overview

AI Agent gives an idea about implementation approach for a Change Request, writes to How File.

Rules:

- Follow definitions from `uspecs/u/concepts.md` and `uspecs/u/conf.md`
- Focus on approach, not on detailed design

Input:

- Change File and other files in Active Change Folder
- Existing How File (if any)

Output:

- Updated How File (see `$templates_folder/tmpl-how.md`)

## Scenarios

```gherkin
Feature: Implementation approach guidance

  Engineer gets an idea about implementation approach for a Change Request

  Scenario Outline: Execute uspecs-how command
    Given <condition>
    When Engineer runs uspecs-how command
    Then AI Agent <action>
    Examples:
      | condition               | action                                                                              |
      | How File does not exist | creates How File with Approach                                                      |
      | How File exists         | adds key elements from tmpl-td.md per AI Agent decision                             |
```
