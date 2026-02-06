# uhow: Change Request clarification

## Overview

AI Agent identifies top 5 uncertainties in Change Request, answers them with alternatives, writes to How File.

Rules:

- Follow definitions from `uspecs/u/concepts.md` and `uspecs/u/conf.md`
- First alternative is the recommended choice
- Web search when Engineer requests it or when choosing between technologies/algorithms/patterns
- Integrate into existing how.md, skip already covered uncertainties

Input:

- Existing How File (if any)
- Optional: area focus (natural language like "focus on authentication", "clarify database design")
- Optional: web search (--web flag or phrases like "with web search", "use web", "web")

Output:

- Updated How File (see `uspecs/u/templates-how.md`)

## Scenarios

```gherkin
Feature: Change Request clarification

  Engineer clarifies and brainstorms Change Request functional and technical design

  Background:
    Given AI Agent identifies top five uncertainties in the Change Request
    And AI Agent groups uncertainties by topic when appropriate
    And Web search is performed when Engineer requests it or when questions involve technology/algorithm/pattern choices

  Scenario Outline: Execute uspecs-how command
    Given <condition>
    When Engineer runs uspecs-how command
    Then AI Agent identifies top 5 uncertainties in <area>, answers them with alternatives, writes to How File
    Examples:
      | condition                      | area                        |
      | Engineer does not specify area | area identified by AI Agent |
      | Engineer specifies area        | area specified by Engineer  |
```
