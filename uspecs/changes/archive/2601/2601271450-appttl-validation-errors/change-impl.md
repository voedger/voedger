# Implementation plan: Wrap IAppTTLStorage validation errors with HTTP status codes

## Construction

- [x] update: [pkg/vvm/storage/errors.go](../../../pkg/vvm/storage/errors.go)
  - Add `ErrAppTTLValidation` sentinel error for validation error category

- [x] update: [pkg/vvm/storage/impl_appttl.go](../../../pkg/vvm/storage/impl_appttl.go)
  - Modify `validateKey`, `validateValue`, `validateTTL` to wrap errors with `ErrAppTTLValidation`
  - Use `fmt.Errorf("%w: %w", ErrAppTTLValidation, specificError)` pattern

- [x] update: [pkg/vvm/storage/impl_appttl_test.go](../../../pkg/vvm/storage/impl_appttl_test.go)
  - Update validation tests to also check `errors.Is(err, ErrAppTTLValidation)`
  - Verify both category error and specific error can be detected

- [x] review
