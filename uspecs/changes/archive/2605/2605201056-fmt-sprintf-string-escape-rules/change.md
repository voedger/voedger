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

- [x] create: [pkg/goutils/jsonu/impl.go](../../../../../pkg/goutils/jsonu/impl.go) + [types.go](../../../../../pkg/goutils/jsonu/types.go) + [impl_test.go](../../../../../pkg/goutils/jsonu/impl_test.go) + [README.md](../../../../../pkg/goutils/jsonu/README.md)
  - `Jprintf(format, args...)` — drop-in replacement for `fmt.Sprintf` that JSON-escapes `string`, `~string`, `fmt.Stringer`, and `error` arguments via `json.Marshal` (handles control chars, invalid UTF-8 → `\ufffd`, U+2028/U+2029); keeps non-string args unchanged
  - `error` arg support: values implementing `error` are escaped via `Error()`; when a type implements both `error` and `fmt.Stringer`, `Error()` wins (mirrors `fmt.Sprintf` precedence)
  - verb semantics: `%s`/`%v` emit JSON-escaped content without quotes (caller supplies quotes in the template); `%q` emits a complete JSON string literal (escaped content wrapped in `"..."`) so templates can stay compact; flags, width and precision are honored via lazy `fmt.Formatter` wrappers in [types.go](../../../../../pkg/goutils/jsonu/types.go) (`jsonString`, `jsonStringer`, `jsonError`)
  - readable JSON templates without the raw-`fmt` `%q` pitfalls (Go-only escapes `\v`, `\xNN` that JSON parsers reject) and without per-call `json.Marshal` boilerplate
  - subtests cover: control chars, HTML-sensitive chars, line/paragraph separators, invalid UTF-8, named string types, Stringer args, error args (plain, `fmt.Errorf`-wrapped, custom types, error-vs-Stringer precedence), `%q` round-trip + width padding, nil-typed-pointer Stringer panic; plus `TestJprintf_BasicUsage` end-to-end example
  - `Jfprintf(w io.Writer, format, args...) (n int, err error)` — same verbs and escaping as `Jprintf` but writes directly to an `io.Writer` (e.g. `*bytes.Buffer`, `http.ResponseWriter`); shares the `jprintfArgs` helper with `Jprintf`. Covered by `TestJfprintf` subtests: `*bytes.Buffer`, `httptest.ResponseRecorder`, JSON-unsafe inputs (`\v`, invalid UTF-8, backslash), returned byte count, writer error propagation via `failingWriter`

- [x] update: violating `fmt.Sprintf`/`fmt.Errorf` call sites found by the audit to conform to the new guide
  - audit: ran `golangci-lint` with gocritic `sprintfQuotedString`, `redundantSprint`, `stringConcatSimplify`, `dynamicFmtString` + `perfsprint` across `./...`; found 153 `sprintfQuotedString` violations (most in `pkg/sys/it/*_test.go`)
  - fix (`"%s"` → `%q` for safe-ASCII/Stringer operands): `pkg/btstrp/impl.go`, `pkg/coreutils/federation/impl.go`, `pkg/processors/blobber/impl_write.go`, `pkg/processors/query2/impl_auth_refresh_handler.go`, `pkg/router/impl_apiv2.go`, `pkg/vvm/impl_requesthandler.go`
  - fix (HTML attribute/text via `html.EscapeString`): `pkg/processors/query2/impl_schemas_handler.go`, `pkg/processors/query2/impl_schemas_roles_handler.go`
  - fix (rewrote JSON bodies with potentially-unsafe operands to `jsonu.Jprintf`): `pkg/cluster/impl_vsqlupdate2.go`, `pkg/coreutils/syserror.go` (Message/Data/QName for both APIV1 and APIV2), `pkg/registry/impl_resetpassword.go`, `pkg/router/utils.go` (`writeCommonError_V2`), `pkg/sys/sqlquery/impl.go`, `pkg/sys/verifier/impl.go`, `pkg/sys/workspace/impl.go` (CreateWorkspaceID/InvokeCreateWorkspace/updateOwner/InitError), `pkg/sys/workspace/impl_deactivate.go`
  - fix (JSON structural integrity): `pkg/router/impl_reply_v1.go` — wrap stand-alone error fragment in `{}` and emit `errorDescription` via `jsonu.Jprintf`
  - fix (refactor for self-balanced JSON): `pkg/sys/invite/impl_applyinviteevents.go` — `updateInviteViaCUD` keeps the JSON-fragment `string` parameter; all call sites build that fragment with `jsonu.Jprintf` so each operand (`VerificationCode`, `Roles`, `WSName`, `Login`, ...) is JSON-escaped
  - fix (`"%s"` → `%q` inside existing `jsonu.Jprintf`/`Jfprintf` templates): post-rewrite refinement applied across `pkg/coreutils/syserror.go`, `pkg/router/utils.go`, `pkg/router/impl_reply_v1.go`, `pkg/registry/impl_resetpassword.go`, `pkg/sys/sqlquery/impl.go`, `pkg/sys/invite/impl_applyinviteevents.go`, `pkg/sys/verifier/impl.go`, `pkg/sys/workspace/impl.go`, `pkg/cluster/impl_vsqlupdate2.go` — `%q` removes the manual `"..."` wrapping and emits a complete JSON string literal; URL-builder `fmt.Sprintf("api/%s/%d/...", ...)` calls intentionally kept as `%s` (URL path components, not JSON strings)
  - migrate (`fmt.Fprintf` → `jsonu.Jfprintf`): `pkg/coreutils/syserror.go` `ToJSON_APIV1`/`ToJSON_APIV2` now write directly into the `*bytes.Buffer` via `jsonu.Jfprintf`; infallible writer errors handled with `// notest` + `panic(err)` per `ar-golang.md`
  - deferred: remaining fixes in `_test.go` and `pkg/vit/` are covered by the guide and left for follow-up changes
  - left unchanged: enabling `sprintfQuotedString` permanently in `.golangci.yml` (separate change request; would require fixing the ~134 test-file violations first); `.golangci.yml` `goconst` tweak and `README.md` trailing-newline edit are incidental and unrelated to this change's scope
