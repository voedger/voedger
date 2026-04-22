# Implementation plan: Eliminate vit.MockBuckets and replace with real rate limit testing

## Construction

- [x] update: [pkg/sys/it/impl_verifier_test.go](../../../../../pkg/sys/it/impl_verifier_test.go)
  - update: `TestVerificationLimits` — remove `MockBuckets` call, send 3 real `InitiateEmailVerification` requests to deplete the real bucket (3/hour), expect 429 on 4th call, use `time.Hour` for time advance instead of `time.Minute`

- [x] update: [pkg/sys/it/impl_resetpassword_test.go](../../../../../pkg/sys/it/impl_resetpassword_test.go)
  - update: `TestResetPasswordLimits` / `InitiateResetPasswordByEmail` — remove `MockBuckets` call, send 3 real requests to deplete the real bucket (3/hour), expect 429 on 4th call, use `verifier.InitiateEmailVerification_Period` for time advance
  - update: `TestResetPasswordLimits` / `IssueVerifiedValueTokenForResetPassword` — remove `MockBuckets` call, send 3 real requests with wrong code to deplete the real bucket (3/hour), expect 429 on 4th call, use `verifier.IssueVerifiedValueToken_Period` for time advance

- [x] update: [pkg/vit/impl.go](../../../../../pkg/vit/impl.go)
  - remove: `MockBuckets` method from `VIT` struct

- [x] review
