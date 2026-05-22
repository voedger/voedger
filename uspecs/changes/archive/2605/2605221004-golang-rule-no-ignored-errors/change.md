---
registered_at: 2026-05-22T09:59:13Z
change_id: 2605220959-golang-rule-no-ignored-errors
type: chore
baseline: 224af22b2f197ae0358add9b4e159b7e60ad5285
issue_url: https://untill.atlassian.net/browse/AIR-3988
archived_at: 2026-05-22T10:04:21Z
---

# Change request: AI agent rule against ignoring Go errors

## Why

AI-generated Go code in this repository frequently discards errors with `_`, e.g. `res, _ := json.Marshal(x)`. Silent error discard hides real failures and diverges from the project's expected error-handling style.

## What

A new project-level AI agent rule that mandates explicit handling of errors from calls that are "infallible by construction" (e.g. `json.Marshal` over a fully-controlled struct):

- Disallow assigning errors to `_`
- Require the standard `if err != nil { // notest; return err / panic(err) }` form so coverage tools skip the unreachable branch
- Scoped to Go code; placed next to the existing Go conventions

### Rule, not skill

Delivered as an Augment rule (`.augment/rules/ar-golang.md`), not as a Claude skill (`.claude/skills/`):

- **Reliability**: rules are always loaded into the system prompt, so the convention fires on every Go edit. Skills are activated only when the agent recognises a match from the skill description, which is unreliable for a strict, ever-present style rule.
- **Cohesion**: the convention is a Go coding rule and belongs with the other Go conventions already in `ar-golang.md`; a separate skill would fragment Go guidance across two locations.
- **KISS**: a few lines appended to an existing rule file beat a new skill folder with its own SKILL.md scaffolding for a single short convention.
- **Tool coverage**: `.augment/rules` apply to the Augment Agent used in this repo; `.claude/skills` would only apply to Claude Code sessions.

A skill would be preferred only if the guidance grew into a multi-step procedure, needed rich reference material, or had to be shared with Claude Code workflows — none of which is the case here.

## Provisioning and configuration

- [x] update: [.augment/rules/ar-golang.md](../../../../../.augment/rules/ar-golang.md): append a bullet forbidding `_`-discard of errors and requiring the `if err != nil { // notest; return err / panic(err) }` form for calls that are infallible by construction (e.g. `json.Marshal` over a fully-controlled struct, `bytes.Buffer.Write`)
  - include a short before/after example so the agent applies the exact shape
  - keep the existing bullet style (single-line bullets, no headings)
