---
registered_at: 2026-01-27T14:37:09Z
change_id: 260127-appttl-validation-errors
baseline: 06013298a63252aa9c69b04e6b6eedbff2422709
archived_at: 2026-01-27T14:50:25Z
---

# Change request: Wrap IAppTTLStorage validation errors with HTTP status codes

## Why

IAppTTLStorage methods return validation errors (ErrKeyEmpty, ErrKeyTooLong, ErrValueTooLong, ErrInvalidTTL) that should result in HTTP 400 Bad Request responses. Currently these errors are plain `errors.New()` which would result in HTTP 500 Internal Server Error when returned from command/query handlers.

## What

Create a dedicated validation error type in `pkg/vvm/storage` and wrap validation errors with it:

- Add `ErrAppTTLValidation` sentinel error in `pkg/vvm/storage/errors.go`
- Wrap existing validation errors (ErrKeyEmpty, ErrKeyTooLong, ErrValueTooLong, ErrInvalidTTL) with `ErrAppTTLValidation`
- Upper layers can check `errors.Is(err, storage.ErrAppTTLValidation)` to determine if error is validation-related

Implementation:

- Define `ErrAppTTLValidation = errors.New("app TTL storage validation error")`
- Modify validation methods to return `fmt.Errorf("%w: %w", ErrAppTTLValidation, ErrKeyEmpty)` pattern
- This allows checking both the category (`ErrAppTTLValidation`) and specific error (`ErrKeyEmpty`)

Error handling flow:

- Validation error occurs in IAppTTLStorage method
- Error is wrapped: `fmt.Errorf("%w: %w", ErrAppTTLValidation, specificError)`
- Upper HTTP layer checks `errors.Is(err, storage.ErrAppTTLValidation)` → HTTP 400
- Non-validation errors (storage failures) are not wrapped → HTTP 500

Affected methods:

- `TTLGet` - wrap ErrKeyEmpty, ErrKeyTooLong
- `InsertIfNotExists` - wrap ErrKeyEmpty, ErrKeyTooLong, ErrValueTooLong, ErrInvalidTTL
- `CompareAndSwap` - wrap ErrKeyEmpty, ErrKeyTooLong, ErrValueTooLong, ErrInvalidTTL
- `CompareAndDelete` - wrap ErrKeyEmpty, ErrKeyTooLong
