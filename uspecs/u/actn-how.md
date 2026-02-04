# uhow: Change Request clarification

## Overview

Rules:

- Strictly follow the definitions from `uspecs/u/concepts.md` and `uspecs/u/conf.md`
- Web search is optional - performed only when Engineer specifies --web flag or mentions using web search, or when questions involve choosing between technologies/algorithms/patterns
- First alternative is the recommended choice

Parameters:

- Input
  - Existing How File (if any)
  - Forced web search option
    - --web flag
    - Natural language - phrases like "with web search", "use web search", "search web" or just "web"
  - Forced functional design clarification
    - --fd flag
    - Natural language - phrases like "functional design", "fd"
  - Forced technical design clarification
    - --td flag
    - Natural language - phrases like "technical design", "td"
  - Forced provisioning and configuration clarification
    - --prov flag
    - Natural language - phrases like "provisioning", "configuration", "prov"
  - Forced construction clarification
    - --con flag
    - Natural language - phrases like "construction", "con"
- Output
  - Updated How File with new section
  - Ref. `uspecs/u/templates-how.md`

Implementation Plan is created/processed according to the rules described in the `Scenarios` section below using definitions and structures from the sections below.

## Definitions

### `Functional design` section does not exist in How File and needed

The section is needed if:

- Engineer forces it
- Change Request has functional aspects to clarify (actors, domains, features)

### `Technical design` section does not exist in How File and needed

The section is needed if:

- Engineer forces it
- Change Request has technical aspects to clarify (architecture, tech stack, components)

### `Provisioning and configuration` section does not exist in How File and needed

The section is needed if:

- Engineer forces it
- Change Request has provisioning aspects to clarify (dependencies, tools, SDKs, external services)
- Change Request has configuration aspects to clarify (environment variables, build settings, CI/CD)

### `Construction` section does not exist in How File and needed

The section is needed if:

- Engineer forces it
- Change Request has construction aspects to clarify (coding patterns, testing strategies, file organization)

## Scenarios

```gherkin
Feature: Change Request clarification

  Engineer clarifies and brainstorms Change Request functional and technical design

  Background:
    Given AI Agent identifies three uncertainties in the Change Request
    And Web search is performed when Engineer requests it or when questions involve technology/algorithm/pattern choices

  Scenario Outline: Execute uspecs-how command
    When Engineer runs uspecs-how command
    Then AI Agent executes only one (the first available) <action> depending on <condition>
    Examples:
      | condition                                                                                       | action                                                                                                          |
      | `Functional design` section does not exist in How File and needed                               | Identify uncertainties about functional design, answer them with alternatives, write to How File                |
      | `Functional design` section does exist but Engineer forces further clarification                | Identify uncertainties about functional design, answer them with alternatives, write to How File                |
      | `Technical design` section does not exist in How File and needed                                | Identify uncertainties about technical design, answer them with alternatives, write to How File                 |
      | `Technical design` section does exist but Engineer forces further clarification                 | Identify uncertainties about technical design, answer them with alternatives, write to How File                 |
      | `Provisioning and configuration` section does not exist in How File and needed                  | Identify uncertainties about provisioning and configuration, answer them with alternatives, write to How File   |
      | `Provisioning and configuration` section does exist but Engineer forces further clarification   | Identify uncertainties about provisioning and configuration, answer them with alternatives, write to How File   |
      | `Construction` section does not exist in How File and needed                                    | Identify uncertainties about construction, answer them with alternatives, write to How File                     |
      | `Construction` section does exist but Engineer forces further clarification                     | Identify uncertainties about construction, answer them with alternatives, write to How File                     |
      | Nothing of the above                                                                            | Display message "No action needed"
```
