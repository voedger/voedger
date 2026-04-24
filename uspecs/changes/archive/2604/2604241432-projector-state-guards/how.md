# How: Add state guards to invite projectors

Two-layer protection against stale projector events overwriting recovery actions.

## Layer 1: Projector state guard

Each projector checks current invite state before doing any work. If state changed from expected (recovery action happened), skip silently:

- `ApplyInvitation`: check `State == ToBeInvited` after loading invite
- `ApplyJoinWorkspace`: check `State == ToBeJoined` after loading invite

If check fails, return `nil` - the invite was already processed or recovered.

## Layer 2: Command-based state transitions

Projectors invoke commands instead of direct CUD for final state transitions. Commands validate state again and return error if state changed during projector execution:

- `ApplyInvitation` calls `c.sys.CompleteInvitation` (ToBeInvited -> Invited)
- `ApplyJoinWorkspace` calls `c.sys.CompleteJoinWorkspace` (ToBeJoined -> Joined)

If command returns error (state changed between guard check and command execution), projector fails and is reapplied. On reapply, guard check sees actual state and skips.

## Flow

```text
Projector starts
  |-> Guard: check state == expected?
  |   |-> No: skip (return nil)
  |   |-> Yes: continue
  |-> Do work (send email, create subject)
  |-> Call command (validates state again)
      |-> State changed? Error -> projector fails -> reapplied
      |-> State still expected? Transition to final state
```

On reapply after failure:

```text
Projector starts again
  |-> Guard: check state == expected?
      |-> State is Cancelled/Invited/Joined: skip (return nil)
```

## Testing

Async projectors run in background goroutines - no VIT mechanism to pause or control
projector execution order. Two hooks per projector allow deterministic testing of both layers.

### Hooks

Each projector has two global `func()` vars (nil in production, zero cost):

- Hook 1 (before guard): called before invite record load and state check
- Hook 2 (after guard): called after guard passes, before side effects

```text
Projector starts
  |-> Hook1()                            <-- before invite load
  |-> Load invite record (reads current state)
  |-> Guard: state == expected?
  |   |-> No: skip (return nil)
  |   |-> Yes: continue
  |-> Hook2()                            <-- after guard
  |-> Do work (send email, create subject)
  |-> Call command (validates state again)
```

Hook1 is placed before the invite record load so that state changes during the block
are visible when the guard reads the record after unblock.

Hooks are nil in production (zero cost). Tests must restore via defer.

### Layer 1 test: block before guard, change state

Hook 1 blocks projector before guard. Test changes state while blocked. Guard sees
changed state and skips.

```text
Test (cancel from ToBeInvited):
  hook1 = func() { <-ch }        // block before guard
  InitiateInvitationByEMail()    // event queued, projector starts, blocks before invite load
  CancelSentInvite()             // state -> Cancelled
  close(ch)                      // unblock
  // Guard: Cancelled != ToBeInvited -> skip
  // Final state: Cancelled
```

```text
Test (re-invite from ToBeJoined):
  hook1 = func() { <-ch }        // block before guard
  ... get to ToBeJoined state
  InitiateInvitationByEMail()    // state -> ToBeInvited (re-invite)
  close(ch)                      // unblock join projector
  // Guard: ToBeInvited != ToBeJoined -> skip
  // Final state: Invited (after ApplyInvitation projector runs)
```

### Layer 2 test: block after guard, change state

Hook 2 blocks projector after guard passes. Uses `reached` channel so the test
knows the projector is blocked before changing state. Test changes state while
blocked. Command sees changed state and returns error. Projector fails and is
reapplied. On reapply, Layer 1 guard sees changed state and skips.

```text
Test (cancel ApplyInvitation after guard):
  reached, proceed = channels
  hook2 = func() { close(reached); <-proceed }
  InitiateInvitationByEMail()    // projector: guard passes (ToBeInvited), blocks
  <-reached                      // wait for projector to block
  CancelSentInvite()             // state -> Cancelled
  close(proceed)                 // unblock
  // CompleteInvitation sees Cancelled != ToBeInvited -> error
  // Projector fails, reapplied
  // Reapply: guard sees Cancelled -> skip
  // Final state: Cancelled
```

```text
Test (cancel ApplyJoinWorkspace after guard):
  reached, proceed = channels
  hook2 = func() { close(reached); <-proceed }
  ... get to ToBeJoined state
  InitiateJoinWorkspace()        // projector: guard passes (ToBeJoined), blocks
  <-reached                      // wait for projector to block
  CancelSentInvite()             // state -> Cancelled
  close(proceed)                 // unblock join projector
  // CompleteJoinWorkspace sees Cancelled != ToBeJoined -> error
  // Projector fails, reapplied
  // Reapply: guard sees Cancelled -> skip
  // Final state: Cancelled
```
