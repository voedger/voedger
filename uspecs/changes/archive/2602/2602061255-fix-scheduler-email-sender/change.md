---
registered_at: 2026-02-06T12:45:32Z
change_id: 2602061545-fix-scheduler-email-sender
baseline: 1bb08f7a2307599ebbc7c185fc821de749b56fad
archived_at: 2026-02-06T12:55:13Z
---

# Change request: Fix nil EmailSender in scheduler jobs

## Why

Scheduler jobs that send email via `sys.Storage_SendMail` panic with nil pointer dereference because `EmailSender` is not injected into `BasicSchedulerConfig` by Wire. The async actualizer path works correctly because it uses a separate config provider that explicitly passes `emailSender`.

## What

Fix Wire dependency injection for scheduler email sending:

- Add `"EmailSender"` to the Wire struct directive for `BasicSchedulerConfig` in `pkg/vvm/provide.go`
- Regenerate `pkg/vvm/wire_gen.go` to include `EmailSender` in the constructed config

Add integration test reproducing the issue:

- Add `TestJobs_SendEmail` test in `pkg/sys/it/impl_jobs_test.go`
- Add test app provider `ProvideApp2WithJobSendMail` with a builtin job that sends email
- Add schema `schemaTestApp2WithJobSendMail.vsql` declaring a job with `INTENTS(sys.SendMail)`
