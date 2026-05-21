---
registered_at: 2026-05-20T14:22:47Z
change_id: 2605201422-escape-hyphen-in-login-regexp
type: fix
baseline: e50729e8f343aceebe4189c5848147088e711df6
issue_url: https://untill.atlassian.net/browse/AIR-3992
scope: registry
archived_at: 2026-05-21T06:58:15Z
---

# Change request: Escape hyphen in login validation regexp

## Why

`validLoginRegexp` in `pkg/registry/consts.go` was intended to allow `+`, `-`, and `/` as literal characters in a login, but the hyphen sits between `+` and `\/` inside the character class, so the regexp engine interprets it as a range (`+` .. `/`) instead of a literal `-`. The range `[+-\/]` therefore matches 5 characters instead of the 3 originally intended:

| Code point | Char | Originally intended | Allowed after fix |
| ---------- | ---- | ------------------- | ----------------- |
| U+002B     | `+`  | yes                 | yes               |
| U+002C     | `,`  | no                  | no                |
| U+002D     | `-`  | yes                 | yes               |
| U+002E     | `.`  | no                  | yes               |
| U+002F     | `/`  | yes                 | yes               |

`,` and `.` are currently accepted by accident. The fix removes `,` from the allowed set and explicitly adds `.` (kept allowed on purpose, since existing logins may already contain it in non-leading/trailing/double positions).

## What

Fix the regexp so the hyphen is treated as a literal character, and align tests with the resulting allowed character set:

- Escape the `-` in `validLoginRegexp` (or move it to a position where it is unambiguously literal)
- Explicitly add `.` to the allowed character set
- Add/adjust unit tests to cover `,` char is not allowed for logins

## Construction

- [x] update: [it/impl_signupin_test.go](../../../../../pkg/sys/it/impl_signupin_test.go)
  - add: `","` (and `"test,foo@test.com"`) to the `wrongLogins` list in `TestSignUpErrors/subject name constraint violation` to assert `,` is rejected after the fix
  - add: positive sub-test verifying logins containing `+`, `-`, `/` (e.g. `a+b@x.com`, `a-b@x.com`, `a/b@x.com`) succeed via `vit.SignUp`

- [x] update: [registry/consts.go](../../../../../pkg/registry/consts.go)
  - fix: escape `-` in `validLoginRegexp` (`+-\/` -> `+\-\/`) so the hyphen is a literal character instead of forming a range with `+` and `/`
  - add: `.` to the `validLoginRegexp` character class (preserves backward compatibility with existing logins that contain `.` in non-leading/trailing/double positions)
