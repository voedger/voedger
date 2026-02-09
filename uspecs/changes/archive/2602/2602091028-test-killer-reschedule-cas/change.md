---
registered_at: 2026-02-09T10:09:15Z
change_id: 2602091008-test-killer-reschedule-cas
baseline: 90d0cf4752d8dbeaefd169db73f79c2b1ffda47d
archived_at: 2026-02-09T10:28:17Z
---

# Change request: Unit tests for killer rescheduling on CAS outcomes

## Why

The `maintainLeadership` loop in `pkg/ielections/impl.go` reschedules the killer differently depending on CompareAndSwap results: pre-CAS killer before the call, post-CAS killer on success, and release on `!ok`. These killer rescheduling paths are not covered by unit tests.

## What

Add unit tests to verify killer rescheduling logic in `maintainLeadership`:

- Pre-CAS killer is scheduled before each CompareAndSwap call
- On CAS success: killer is rescheduled with `killDeadlineFactor`
- On CAS `!ok`: leadership is released, no killer rescheduled
- On CAS error after all retries: leadership is released
