---
registered_at: 2026-05-21T14:11:10Z
change_id: 2605211411-user-email-sending-arch
type: docs
scope: apps
baseline: c27f74d7bd4b234f964a9724a0c2ae54d69aadb4
archived_at: 2026-05-21T14:47:22Z
---

# Change request: Email sending subsystem architecture

## Why

Voedger needs a clear subsystem architecture for sending emails so contributors can reason consistently about responsibilities, boundaries, and operational behavior. The architecture should make the email flow understandable before implementation or review decisions depend on scattered local knowledge.

## What

This documentation change delivers subsystem architecture guidance for the email sending area.

- Define the subsystem responsibilities and boundaries for sending emails to users.
- Describe the architectural flow, integration points, and key design constraints readers need when changing or reviewing this subsystem.

## How

Decisions:

- Add a subsystem architecture document for the email sending area, separate from authentication, because the subsystem is about addressing and delivering messages rather than proving identity or authorizing access.
- Describe the real processing path: existing commands, queries, projectors, and jobs that need email create `sys.SendMail` state keys in async actualizer or scheduler state.
- Treat `sys.SendMail` storage as the delivery boundary: it validates required SMTP/message fields, builds `state.EmailMessage`, and delegates delivery to `state.IEmailSender`.
- Describe current email-producing flows, including invitation emails, email verification codes, and scheduled jobs using `INTENTS(sys.SendMail)`.
- Include failure and replay/idempotency considerations as architectural constraints based on current projector behavior and `SendMail` storage behavior.

Out of scope:

- Introducing a generic `c.SendEmailToUser` command or `ap.ApplySendEmail` projector.
- SMTP provider selection, bounce handling, delivery analytics, or user notification preferences.
- Reworking existing invitation and verification email behavior.

References:

- [product domain overview](../../../../specs/prod/domain.md)
- [invite email design](../../../../specs/prod/auth/invites--td.md)
- [system schema email storage](../../../../../pkg/sys/sys.vsql)
- [user profile schema](../../../../../pkg/sys/userprofile.vsql)
- [email verification processing](../../../../../pkg/sys/verifier/impl.go)
- [invitation email processing](../../../../../pkg/sys/invite/impl_applyinviteevents.go)
- [send mail storage implementation](../../../../../pkg/sys/storages/impl_send_mail_storage.go)
- [email sender interface](../../../../../pkg/state/types.go)
- [async actualizer state wiring](../../../../../pkg/state/stateprovide/impl_async_actualizer_state.go)
- [scheduler state wiring](../../../../../pkg/state/stateprovide/impl_scheduler_state.go)

## Technical design

- [x] create: [arch-user-email-sending.md](../../../../specs/prod/arch-user-email-sending.md)
  - Domain Subsystem Architecture: real processing path for user email sending through `sys.SendMail`, async actualizers, scheduler jobs, `state.IEmailSender`, SMTP configuration, failure behavior, and replay/idempotency constraints
