---
registered_at: 2026-05-20T10:26:13Z
change_id: 2605201026-fmt-sprintf-string-escape-rules
type: docs
baseline: e50729e8f343aceebe4189c5848147088e711df6
issue_url: https://untill.atlassian.net/browse/AIR-3985
scope: golang-style
archived_at: 2026-05-20T10:56:45Z
---

# Change request: Design rules for using %s and %q in fmt.Sprintf

## Why

In many places the codebase uses `fmt.Sprintf("%s", data)` where `data` carries content that must be escaped before being embedded into a larger string (JSON, URL query, file path, shell argument, etc.). Without explicit guidance, developers and AI agents keep producing unsafe or ambiguous formatting that leads to injection-style defects and hard-to-diagnose bugs.

## What

A code style guide that prescribes the correct formatting verb and escaping helper for string data passed to `fmt.Sprintf` (and related formatting functions):

- Survey of current usages of `fmt.Sprintf` with `%s`/`%q` on string data across the repository
- Rules that distinguish when `%q` is sufficient, when `%s` must be combined with `url.QueryEscape`, when `json.Marshal` is required, and analogous cases for other embedding contexts
- Guidance targeted at both human developers and AI coding agents
- Fixes to existing call sites that violate the resulting rules

## Construction

- [x] create: [.claude/skills/golang-fmt-string-escape/SKILL.md](../../../../../.claude/skills/golang-fmt-string-escape/SKILL.md)
  - Code style guide for `fmt.Sprintf` (and `fmt.Fprintf`, `fmt.Errorf`, `fmt.Printf`) when embedding string data into a larger structured string
  - Decision table by embedding context: human-readable log / error message, Go-quoted literal, JSON value, URL path segment, URL query parameter, shell argument, file path, SQL literal, HTML attribute
  - Per context: required verb and escaping helper (e.g. `%q` for Go-quoted; `%s` + `url.PathEscape` for URL path; `%s` + `url.QueryEscape` for URL query; `json.Marshal` then `%s` for JSON values; never raw `%s` for SQL literals — use parameterized queries; `%q` over `'%s'` for human messages that may contain spaces or empty strings)
  - Anti-patterns with bad/good code examples drawn from the codebase
  - Short checklist phrased so it is directly usable by AI coding agents
  - Initially authored as always-apply rule `.augment/rules/ar-golang-fmt-string-escape.md`; converted to an on-demand Claude Code Skill (auto-loaded via `description` matching on `fmt.Sprintf`/JSON/URL/HTML keywords). Original rule file deleted; `.gitignore` updated with `!/.claude` so the skill directory is tracked

- [x] update: violating `fmt.Sprintf`/`fmt.Errorf` call sites found by the audit to conform to the new guide
  - audit: ran `golangci-lint` with gocritic `sprintfQuotedString`, `redundantSprint`, `stringConcatSimplify`, `dynamicFmtString` + `perfsprint` across `./...`; found 153 `sprintfQuotedString` violations (most in `pkg/sys/it/*_test.go`)
  - fix (`"%s"` → `%q` for safe-ASCII/Stringer operands): `pkg/btstrp/impl.go`, `pkg/coreutils/federation/impl.go`, `pkg/processors/blobber/impl_write.go`, `pkg/processors/query2/impl_auth_refresh_handler.go`, `pkg/router/impl_apiv2.go`, `pkg/sys/workspace/impl.go`, `pkg/vvm/impl_requesthandler.go`
  - fix (HTML attribute/text via `html.EscapeString`): `pkg/processors/query2/impl_schemas_handler.go`, `pkg/processors/query2/impl_schemas_roles_handler.go`
  - fix (rewrote JSON bodies with potentially-unsafe operands to `json.Marshal` with `// notest` error handling): `pkg/cluster/impl_vsqlupdate2.go`, `pkg/coreutils/syserror.go` (Message/Data; QName kept on `%q`), `pkg/registry/impl_resetpassword.go`, `pkg/router/utils.go` (writeCommonError_V2), `pkg/sys/sqlquery/impl.go`, `pkg/sys/verifier/impl.go`, `pkg/sys/workspace/impl.go` (CreateWorkspaceID/InvokeCreateWorkspace/updateOwner/InitError), `pkg/sys/workspace/impl_deactivate.go`
  - fix (JSON structural integrity): `pkg/router/impl_reply_v1.go` — wrap stand-alone error fragment in `{}` and marshal `errorDescription` via `json.Marshal`
  - fix (refactor for self-balanced JSON): `pkg/sys/invite/impl_applyinviteevents.go` — `updateInviteViaCUD` now takes `map[string]any` instead of a JSON-fragment string; all call sites converted
  - deferred: remaining fixes in `_test.go` and `pkg/vit/` are covered by the guide and left for follow-up changes
  - left unchanged: enabling `sprintfQuotedString` permanently in `.golangci.yml` (separate change request; would require fixing the ~134 test-file violations first); `.golangci.yml` `goconst` tweak and `README.md` trailing-newline edit are incidental and unrelated to this change's scope
