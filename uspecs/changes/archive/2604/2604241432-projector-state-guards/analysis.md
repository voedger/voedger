# PR review analysis: https://github.com/voedger/voedger/pull/4511

## Review 1: Gemini

### Comment

> By allowing CancelSentInvite / InitiateInvitationByEMail to accept transitional states (State_ToBeInvited / State_ToBeJoined), there is now a concrete race with already-enqueued async projectors: ApplyInvitation and ApplyJoinWorkspace do not check the current invite State before sending email / creating joined workspace / setting State=Invited|Joined. If an old projector event retries after an operator cancels or re-invites from these transitional states, it can overwrite the recovered state (e.g., flip Cancelled back to Invited/Joined or join a user after cancellation). Consider adding a guard in the corresponding projectors to no-op unless the invite is still in the expected transitional state, or otherwise version/invalidate stale events so recovery actions are durable.

### Analysis

Gemini identified two concerns:

1. **Race condition**: Valid concern - projectors unconditionally update state without checking if recovery action already changed it.
2. **Suggestion**: Add state guards to projectors - this is exactly the fix we implemented.

### Verdict: Valid, addressed by this change request

---

## Review 2: Copilot

### Ccomment (Copilot)

> Should also add State_ToUpdateRoles, State_ToBeCancelled, State_ToBeLeft to allow recovery from stuck states.

### Analysis (Copilot)

Copilot suggested expanding recovery states to include all transitional states.

| State         | Meaning                         | Recovery appropriate? | Reason                                                            |
|---------------|---------------------------------|-----------------------|-------------------------------------------------------------------|
| ToBeInvited   | Email not sent yet              | Yes                   | Pre-join, invite can be re-sent or cancelled                      |
| ToBeJoined    | User clicked link, join pending | Yes                   | Pre-join, user can be re-invited or cancelled                     |
| ToUpdateRoles | Admin updating roles            | No                    | Post-join, different semantic - roles update, not invite recovery |
| ToBeCancelled | Admin removing user             | No                    | Post-join, this is a removal operation, not an invitation state   |
| ToBeLeft      | User leaving workspace          | No                    | Post-join, user-initiated action, not admin invitation issue      |

Post-join states (`ToUpdateRoles`, `ToBeCancelled`, `ToBeLeft`) represent operations on already-joined members. They have different failure handling:

- `ToUpdateRoles`: Roles update is idempotent, can be retried by admin
- `ToBeCancelled`: Removal is terminal, if stuck the user still has access which is a different class of problem
- `ToBeLeft`: User-initiated, retry by user

### Verdict: Rejected

Adding these would conflate pre-join invitation recovery with post-join member management. These states need separate handling if stuck-state recovery is needed for them.

---

## Review 3: Human (implied)

The race condition identified by Gemini requires architectural fix beyond just adding state guards. Current design has projectors directly mutating state via CUD, bypassing command validation. Better approach is using commands for state transitions, allowing validation at each step.

This change request implements:

1. State guards in projectors (quick check before doing work)
2. Commands for final state transitions (validated state changes)
3. Tests for race scenarios