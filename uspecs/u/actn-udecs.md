# Action: Change Request clarification

## Overview

Identify top 5 uncertainties in Change Request, answer them with alternatives, and write to Decision File.

## Instructions

Rules:

- Always read `uspecs/u/concepts.md` and `uspecs/u/conf.md` before proceeding and follow the definitions and rules defined there
- First alternative is the recommended choice
- Web search when Engineer requests it or when choosing between technologies/algorithms/patterns
- Integrate into existing Decision File, skip already covered uncertainties

Parameters:

- Input
  - Existing Decision File (if any)
  - Optional: area focus (natural language like "focus on authentication", "clarify database design")
  - Optional: web search (--web flag or phrases like "with web search", "use web", "web")
- Output
  - Updated Decision File (see `{templates_folder}/tmpl-decs.md`)

## Scenarios

```gherkin
Feature: Change Request clarification

  Engineer clarifies and brainstorms Change Request functional and technical design

  Background:
    Given AI Agent identifies top five uncertainties in the Change Request
    And AI Agent groups uncertainties by topic when appropriate
    And Web search is performed when Engineer requests it or when questions involve technology/algorithm/pattern choices

  Scenario Outline: Execute uspecs-decs command
    Given <condition>
    When Engineer runs uspecs-decs command
    Then AI Agent identifies top 5 uncertainties in <area>, answers them with alternatives, writes to Decision File
    Examples:
      | condition                      | area                        |
      | Engineer does not specify area | area identified by AI Agent |
      | Engineer specifies area        | area specified by Engineer  |
```
