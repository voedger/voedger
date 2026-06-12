# Domain subsystem architecture: email sending

## Overview

Email sending is the platform path that turns application-side email side effects into `sys.SendMail` state operations and delegates delivery to the configured email sender.

## Components

- [Email-producing VSQL declarations](../../../pkg/sys/sys.vsql) - declare system commands, projectors, and `STORAGE SendMail`.
- [User profile email declarations](../../../pkg/sys/userprofile.vsql) - declare email verification command and projector wiring.
- [Email verification processing](../../../pkg/sys/verifier/impl.go) - initiates verification and emits verification-code email intents.
- [Invitation email processing](../../../pkg/sys/invite/impl_applyinviteevents.go) - emits invitation and role-update email intents from async invite processing.
- [Scheduler email job example](../../../pkg/vit/schemaTestApp2WithJobSendMail.vsql) - app jobs that declare `INTENTS(sys.SendMail)` can emit email intents while running in scheduler state.
- [Async actualizer state](../../../pkg/state/stateprovide/impl_async_actualizer_state.go) - exposes `sys.SendMail` to async projectors with get and insert access.
- [Scheduler state](../../../pkg/state/stateprovide/impl_scheduler_state.go) - exposes `sys.SendMail` to scheduled jobs with get and insert access.
- [SendMail storage](../../../pkg/sys/storages/impl_send_mail_storage.go) - validates message and SMTP fields, builds `state.EmailMessage`, and invokes the email sender.
- [Email sender interface](../../../pkg/state/types.go) - abstracts outbound delivery behind `state.IEmailSender`.
- [SMTP configuration](../../../pkg/sys/smtp/types.go) - supplies host, port, sender address, username, and password secret name to email-producing code.
- [App secret storage](../../../pkg/sys/storages/impl_app_secrets_storage.go) - provides the SMTP password secret to projectors and jobs at runtime.

```text
Email producers
  |
  +-- ap.sys.ApplySendEmailVerificationCode
  +-- ap.sys.ApplyInviteEvents
  +-- Jobs with INTENTS(sys.SendMail)
        |
        v

Execution state
  |
  +-- Async actualizer state
  |     |
  |     +-- sys.AppSecret storage
  |     +-- sys.SendMail storage
  |
  +-- Scheduler state
        |
        +-- sys.AppSecret storage
        +-- sys.SendMail storage
              |
              v

Delivery boundary
  |
  +-- sys.SendMail storage
        |
        +-- validates SMTP and message fields
        +-- builds state.EmailMessage
        +-- invokes state.IEmailSender
              |
              v

Transport
  |
  +-- state.IEmailSender
        |
        v

External system
  |
  +-- SMTP server
```

## Key flows

### Email verification code

```text
Client
  |
  | request verification token/code
  v
q.sys.InitiateEmailVerification
  |
  | create verification token and code
  v
Federation
  |
  | call c.sys.SendEmailVerificationCode with system token
  v
c.sys.SendEmailVerificationCode
  |
  | PLog event
  v
ap.sys.ApplySendEmailVerificationCode
  |
  | read SMTP password from sys.AppSecret
  | create sys.SendMail key as intent
  v
Async actualizer state
  |
  | apply intent
  v
sys.SendMail storage
  |
  | send EmailMessage
  v
state.IEmailSender
```

The projector skips events older than three days to avoid re-sending verification emails after a projector rename (see voedger/voedger#275).

### Invitation and role-update emails

```text
Invite command
  |
  | PLog event with invite CUD
  v
ap.sys.ApplyInviteEvents
  |
  | skip pre-refactor or stale events
  | read invite/workspace state
  | read SMTP password from sys.AppSecret
  | create sys.SendMail key as intent
  v
Async actualizer state
  |
  | apply intent
  v
sys.SendMail storage
  |
  | send EmailMessage
  v
state.IEmailSender

ap.sys.ApplyInviteEvents
  |
  | write final invite state
  v
Workspace records/views
```

`ap.sys.ApplyInviteEvents` is responsible for current invite email side effects. It uses a CUD-side `Version` discriminator to avoid re-sending emails from already-processed historical events.

### Scheduler email job

```text
Scheduler
  |
  | invoke due job
  v
Job with INTENTS(sys.SendMail)
  |
  | create or read sys.SendMail key
  v
Scheduler state
  |
  | apply/get SendMail operation
  v
sys.SendMail storage
  |
  | send EmailMessage
  v
state.IEmailSender
```

Scheduler state exposes the same `sys.SendMail` storage as async actualizers so jobs use the same delivery boundary.
