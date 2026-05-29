---
change_id: 2605290953-correct-copyright-skill
type: feat
issue_url: https://untill.atlassian.net/browse/AIR-4097
---

# Change request: Skill enforcing the correct copyright header on new files

Refs:

- [AIR-4097: implement skill that will force to produce the correct copyright](./issue-AIR-4097.md)

## Why

AI-assisted file creation currently emits an outdated copyright header attributed to the wrong legal entity and without an author line, which then has to be corrected by hand on every new file. A repository-level skill is needed so the agent always produces the canonical header for the current owner and the current contributor without manual intervention.

## What

Add a development-tooling skill that governs the copyright header authored on every newly created source file in the repository:

- Triggers automatically whenever the agent creates a new source file in the Go codebase or in the `sql/vsql` sources
- Emits a header that uses the current calendar year, the current legal entity, and the current Git user as author
- Replaces the previously emitted legacy header so that no new file is committed with the outdated wording
- Leaves existing files and existing headers untouched, so the change has no effect on already-committed sources

## How

Decisions:

- Encode the policy as a Claude Code skill at `.claude/skills/copyright-header/SKILL.md`, mirroring the layout of the existing `golang-fmt-string-escape` skill
- Phrase the `description` so the skill auto-activates on creation of any `.go` or `.vsql` source file (the only file families called out by the issue)
- Pin the entity to `unTill Software Development Group B.V.` and require the current calendar year, replacing the legacy `unTill Pro, Ltd.` / `Sigma-Soft, Ltd.` wording for newly created files
- Require an `@author <name>` line populated from the current Git user (`git config user.name`)
- Provide two header templates verbatim in the skill body: a `/* ... */` block for Go, a `-- ...` line block for VSQL
- Apply only when the agent creates a new file; existing files keep their committed header

Out of scope:

- Rewriting copyright headers on already-committed files
- Updating the year on files that change in subsequent edits
- Mirroring the policy as an `.augment/rules/` always-applied rule (deferred unless the skill proves unreliable)

References:

- [existing Claude skills folder](../../../../../.claude/skills)
- [reference skill layout and frontmatter](../../../../../.claude/skills/golang-fmt-string-escape/SKILL.md)
- [Go header example](../../../../../pkg/itokensjwt/types.go)
- [VSQL header example](../../../../../pkg/sys/sys.vsql)
