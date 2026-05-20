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

- [x] create: [rules/ar-golang-fmt-string-escape.md](../../../../../.augment/rules/ar-golang-fmt-string-escape.md)
  - Code style guide for `fmt.Sprintf` (and `fmt.Fprintf`, `fmt.Errorf`, `fmt.Printf`) when embedding string data into a larger structured string
  - Decision table by embedding context: human-readable log / error message, Go-quoted literal, JSON value, URL path segment, URL query parameter, shell argument, file path, SQL literal, HTML attribute
  - Per context: required verb and escaping helper (e.g. `%q` for Go-quoted; `%s` + `url.PathEscape` for URL path; `%s` + `url.QueryEscape` for URL query; `json.Marshal` then `%s` for JSON values; never raw `%s` for SQL literals — use parameterized queries; `%q` over `'%s'` for human messages that may contain spaces or empty strings)
  - Anti-patterns with bad/good code examples drawn from the codebase
  - Short checklist phrased so it is directly usable by AI coding agents (mirrors the style of existing `.augment/rules/ar-golang.md`)

- [x] update: violating `fmt.Sprintf`/`fmt.Errorf` call sites found by the audit to conform to the new guide
  - audit: ran `golangci-lint` with gocritic `sprintfQuotedString`, `redundantSprint`, `stringConcatSimplify`, `dynamicFmtString` + `perfsprint` across `./...`; found 153 `sprintfQuotedString` violations (most in `pkg/sys/it/*_test.go`)
  - fix: replaced `"%s"` → `%q` in 19 production-code sites across `pkg/btstrp/impl.go`, `pkg/coreutils/federation/impl.go`, `pkg/processors/blobber/impl_write.go`, `pkg/processors/query2/impl_auth_refresh_handler.go`, `pkg/processors/query2/impl_schemas_handler.go`, `pkg/processors/query2/impl_schemas_roles_handler.go`, `pkg/registry/impl_resetpassword.go`, `pkg/router/impl_apiv2.go`, `pkg/router/impl_reply_v1.go`, `pkg/sys/verifier/impl.go`, `pkg/sys/workspace/impl.go`, `pkg/vvm/impl_requesthandler.go`
  - out of scope: full `json.Marshal` rewrites of JSON-body builders and fixes to `_test.go` / `pkg/vit/` violations; covered by the guide and deferred to follow-up changes
  - left unchanged: enabling `sprintfQuotedString` permanently in `.golangci.yml` (separate change request; would require fixing the ~134 test-file violations first)
