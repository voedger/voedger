---
registered_at: 2026-05-19T14:42:39Z
change_id: 2605191442-clarify-verified-field-token-error
type: fix
scope: istructsmem
baseline: e50729e8f343aceebe4189c5848147088e711df6
issue_url: https://untill.atlassian.net/browse/AIR-3973
archived_at: 2026-05-19T14:54:42Z
---

# Change request: Clarify error when verification token for a verified field is invalid

## Why

When a CUD on a verified field is submitted with an invalid verification-value token (e.g. `update untill.airs-bp.%d.air.UserProfile.%d set EMail='<bad-token>'`), the response surfaces a bare JWT error such as `token is malformed: token contains an invalid number of segments. invalid token`. The message is indistinguishable from a failure of the request's principal/authorization token, so the caller debugs the wrong token and the verified-field flow looks broken.

## What

Make every failure of a verified-field value-token explicit about its origin so the caller can immediately tell it apart from a principal-token failure:

- Wrap the underlying `IAppTokens.ValidateToken` error with the row's QName and the verified field's name (e.g. `verification token for field «<entity>.<field>» is invalid: <underlying error>`)
- Wrong-payload errors raised after successful token parsing (verification kind mismatch, wrong entity, wrong field, value clarification failure) follow the same prefix so all verified-field token failures share one recognizable shape
- Underlying error is preserved (wrapped with `%w`) so callers can still match the original sentinel

## Construction

- [x] update: [istructsmem/types.go](../../../../../pkg/istructsmem/types.go) — `verifyToken`
  - update: wrap the `AppTokens().ValidateToken` error with the row's QName and the verified field's name (e.g. `verification token for %v.%s is invalid: <underlying>`) using `%w` so the original sentinel stays matchable
  - keep: the post-parse errors (`ErrInvalidVerificationKind`, wrong-entity, wrong-field, `clarifyJSONValue` failure) already mention the row and field — verify their messages also clearly identify the verified-field token origin and align wording with the new prefix

- [x] update: [istructsmem/validation_test.go](../../../../../pkg/istructsmem/validation_test.go)
  - add: subtest in the verified-field block asserting that a malformed token (e.g. arbitrary non-JWT string) returns an error containing both the entity QName and the verified field name, and unwraps to the underlying token-parse error
  - keep: existing `ErrInvalidVerificationKindError`, wrong-entity, and wrong-field subtests — adjust their expected-message assertions if the prefix wording changes

- [x] Review
