# Implementation plan

## Construction

- [x] update: [pkg/vvm/provide.go](../../../pkg/vvm/provide.go)
  - fix: Add `"EmailSender"` to Wire struct directive for `BasicSchedulerConfig`
- [x] update: [pkg/vvm/wire_gen.go](../../../pkg/vvm/wire_gen.go)
  - fix: Regenerate to include `EmailSender: iEmailSender` in `BasicSchedulerConfig` initialization
- [x] create: [pkg/vit/schemaTestApp2WithJobSendMail.vsql](../../../pkg/vit/schemaTestApp2WithJobSendMail.vsql)
  - add: Schema declaring `JobSendEmail` job with `INTENTS(sys.SendMail)`
- [x] update: [pkg/vit/consts.go](../../../pkg/vit/consts.go)
  - add: Embed `schemaTestApp2WithJobSendMail.vsql` via `//go:embed` directive
- [x] update: [pkg/vit/shared_cfgs.go](../../../pkg/vit/shared_cfgs.go)
  - add: `ProvideApp2WithJobSendMail` provider with builtin job that sends email via `sys.Storage_SendMail`
- [x] update: [pkg/sys/it/impl_jobs_test.go](../../../pkg/sys/it/impl_jobs_test.go)
  - add: `TestJobs_SendEmail` test that launches job app and captures sent email
- [x] review: Run `TestJobs_SendEmail` and verify it passes
