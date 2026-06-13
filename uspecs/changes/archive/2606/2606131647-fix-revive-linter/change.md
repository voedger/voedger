---
registered_at: 2026-05-15T07:29:16Z
change_id: 2605150729-fix-revive-linter
type: fix
baseline: e14db6996bb6c70f80693907672d39e984332873
issue_url: https://untill.atlassian.net/browse/AIR-3905
---

# Change request: Fix and respect revive linter

## Why

The `revive` linter is enabled in `.golangci.yml` but the codebase contains unresolved violations and the linter configuration is not fully respected during development. See [issue.md](issue.md) for details.

## What

Bring the codebase in line with the `revive` linter configured in `.golangci.yml`:

- Fix existing `revive` violations across the repository
- Ensure `revive` rules (`indent-error-flow`, `add-constant`) are respected in new and existing code
- Adjust `revive` configuration in `.golangci.yml` only where justified, keeping the rule set meaningful
