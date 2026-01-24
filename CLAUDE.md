# Agents instructions

## Naming conventions

| Category                | Convention    | Example                                |
|-------------------------|---------------|----------------------------------------|
| Specs file/folder names | kebab-case    | `spec-impact.md`, `init-project/`      |
| Entity names in specs   | Title Case    | `Human`, `External System`, `AI Agent` |
| Gherkin scenarious      | Sentence case | `User registration`, `Email delivery`  |
| Descriptive text        | Sentence case | `Sends transactional emails`           |
| Section headers         | Sentence case | `## Specifications impact`             |

<!-- uspecs:triggering_instructions:begin -->

## Execution instructions

When request mentions:

- uchange: Use rules from `uspecs/u/actn-changes.md` to create change
- uarchive: Use rules from  `uspecs/u/actn-changes.md` to Archive change
- uimpl: Use rules from `uspecs/u/actn-impl.md` to manage Implementation Plan

Use files from `./uspecs/u` as an initial reference when user mentions uspecs

<!-- uspecs:triggering_instructions:end -->
