---
registered_at: 2026-04-24T10:00:36Z
change_id: 2604241000-projector-state-guards
baseline: f91d1438f6fbf47ffd247bb8d06e47cfdff55e6e
issue_url: https://untill.atlassian.net/browse/AIR-3704
archived_at: 2026-04-24T14:32:29Z
---

# Change request: Add state guards to invite projectors

## Why

PR [#4511](https://github.com/voedger/voedger/pull/4511) fixed recovery from stuck invite states (ToBeInvited, ToBeJoined) by allowing commands to accept these transitional states. During review, a race condition was identified.

### Problem

Async projectors `ApplyInvitation` and `ApplyJoinWorkspace` do not check the current invite state before performing side effects and updating state via CUD. The sequence:

- Invite created -> State = ToBeInvited, projector event queued
- Owner cancels invite -> State = Cancelled (via CancelSentInvite)
- Projector runs from queue, sends email, sets State = Invited (overwrites Cancelled)

The same applies to `ApplyJoinWorkspace`:

- User joins -> State = ToBeJoined, projector event queued
- Owner re-invites -> State = ToBeInvited (via InitiateInvitationByEMail)
- Projector runs from queue, creates subject, sets State = Joined (overwrites ToBeInvited)

### Impact

- Cancelled invites silently become active again
- Re-invited users get joined to workspace with stale role assignments
- No error or log - the overwrite is silent

## What

- Layer 1 - projector state guards: each projector checks current invite state before doing work, skips if state changed (recovery action already happened)
- Layer 2 - command-based state transitions: replace direct `c.sys.CUD` calls in projectors with dedicated commands (`c.sys.CompleteInvitation`, `c.sys.CompleteJoinWorkspace`) that validate expected state at execution time, closing the TOCTOU gap between guard check and state update
- Prepare invite specifications
  - `uspecs/specs/prod/prod--domain.md`: expand auth context with workspace membership
  - `uspecs/specs/prod/auth/invites--td.md`: feature technical design for invite system
