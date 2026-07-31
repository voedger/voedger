# Decisions: Replace accepted invitations without revoking membership

## Uncertainty: Handling a second accepted invitation for an active member

Decision: An active Subject identifies its single controlling accepted invitation through `Subject.InviteEmail`. Resolve that address through `InviteIndexView`, cancel the resolved previous invitation without deactivating the Subject or JoinedWorkspace, then accept the new invitation, apply its roles, and replace `Subject.InviteEmail` with the new `Invite.Email`.

- Pros: Allows the new invitation to replace the previous one while preserving uninterrupted workspace access and gives each Subject one authoritative controlling invitation.
- Cons: Gives cancellation two different side-effect profiles and cannot automatically replace membership when neither the stored address nor the current canonical-login/alias pair identifies a unique previous invitation.
- Confidence: user-provided

Alternatives:

1. Reject the second acceptance with `409 Conflict`
   - Pros: Preserves current membership ownership and requires no invitation-transfer semantics.
   - Cons: Leaves the additional invitation pending and prevents intentional replacement through acceptance.
   - Confidence: high
2. Supersede the previous invitation with a dedicated state
   - Pros: Makes replacement explicit in the state model and audit history.
   - Cons: Requires a new state and corresponding command, projector, compatibility, and migration behavior.
   - Confidence: medium

## Uncertainty: Component responsible for retiring the previous invitation

Decision: `InitiateJoinWorkspace` performs command-side validation and records the canonical membership identity, while `ApplyInviteEvents` performs the replacement in PLog order and remains the sole writer of final invitation states and membership side effects.

- Pros: Preserves the single-projector architecture, serializes concurrent acceptance events, and supports idempotent replay and recovery.
- Cons: Replacement failures discovered during projection cannot always be returned synchronously by `InitiateJoinWorkspace`.
- Confidence: high

Alternatives:

1. Let `InitiateJoinWorkspace` directly retire the previous invitation
   - Pros: Provides an immediate outcome to the caller.
   - Cons: Introduces another writer of final invitation state and creates races with queued or replayed events.
   - Confidence: low
2. Introduce a dedicated replacement command
   - Pros: Gives invitation replacement an explicit operation and contract.
   - Cons: Expands the API and still requires coordination with the join event and projector.
   - Confidence: medium

## Uncertainty: Subject field used to identify the controlling invitation

Decision: Add `Subject.InviteEmail` as the single authoritative link to the controlling accepted invitation. Store the exact immutable `Invite.Email` value and resolve it through `InviteIndexView` when the projector must load the previous invitation.

- Pros: Expresses the one-controlling-invitation invariant directly on Subject and reuses the existing email-to-InviteID index.
- Cons: Duplicates the invitation address on Subject and requires fallback lookup for existing Subjects without the field.
- Confidence: user-provided

Alternatives:

1. Add `Subject.InviteID ref`
   - Pros: Uses a stable direct record reference and avoids an index lookup.
   - Cons: Requires a new reference field and the same compatibility handling for existing Subjects.
   - Confidence: high
2. Add a reverse invitation view keyed by `SubjectID`
   - Pros: Avoids changing Subject and can expose existing duplicate invitation links.
   - Cons: Requires a new maintained view and a rule for choosing among multiple joined invitations.
   - Confidence: medium

## Uncertainty: Compatibility for existing Subjects without a resolvable InviteEmail

Decision: When `Subject.InviteEmail` is absent or cannot be resolved, use the authenticated canonical login and active login alias to look up invitations through `InviteIndexView`, excluding the invitation currently being accepted. Continue replacement only when exactly one candidate is joined and its `SubjectID` matches the active Subject; otherwise return `409 Conflict` with no state changes. A successful replacement stores the new `Invite.Email` in `Subject.InviteEmail`.

- Pros: Handles the common canonical-login and active-alias legacy cases without a migration while refusing ambiguous ownership.
- Cons: Cannot recover automatically when the previous invitation used an old or removed alias.
- Confidence: high

Alternatives:

1. Always return `409 Conflict` when `Subject.InviteEmail` is absent or unresolved
   - Pros: Never guesses which invitation controls membership.
   - Cons: Existing members cannot replace invitations until their Subjects are migrated.
   - Confidence: high
2. Migrate existing Subjects before enabling replacement
   - Pros: Provides deterministic links and an opportunity to detect existing duplicates.
   - Cons: Requires migration and rollout work plus a policy for ambiguous historical data.
   - Confidence: medium

## Uncertainty: Concurrent invitation acceptances before projection

Decision: Apply valid acceptance events in PLog order. Each event for a different `Invited` invitation replaces the Subject's current controlling invitation, and the last successfully applied event becomes the sole controller. An event replay for the same invitation is skipped after that invitation has reached `Joined`.

- Pros: Produces deterministic last-wins behavior, preserves the single-projector architecture, and maintains exactly one controlling invitation after every applied event.
- Cons: Multiple callers may receive successful command responses even though an earlier accepted invitation is quickly superseded by a later event.
- Confidence: high

Alternatives:

1. Let the first applied invitation win and skip later acceptance events
   - Pros: Prevents immediate replacement of a newly established controlling invitation.
   - Cons: A later caller may receive command success even though its invitation is never applied.
   - Confidence: medium
2. Add command-side synchronization or uniqueness enforcement
   - Pros: Allows later callers to receive an immediate conflict.
   - Cons: Requires a broader synchronization mechanism outside the existing projector ordering.
   - Confidence: medium

## Ambiguity: Meaning of one invitation linked to a Subject

Decision: Enforce at most one active controlling invitation per Subject. The active link is the `Joined` invitation whose `Email` equals `Subject.InviteEmail`. Cancelled, left, or otherwise retired invitations may retain `SubjectID` as historical provenance but cannot control roles or membership cancellation.

- Pros: Preserves invitation history while making the active ownership invariant explicit and avoiding unnecessary mutation of historical records.
- Cons: Queries that inspect `Invite.SubjectID` must also consider invitation state and `Subject.InviteEmail` rather than treating every historical reference as active.
- Confidence: user-provided

Alternatives:

1. Clear the previous invitation's `SubjectID` during replacement
   - Pros: Leaves only the active invitation with a direct Subject reference.
   - Cons: Loses historical linkage and complicates auditing and diagnostics.
   - Confidence: medium
2. Preserve `SubjectID` and add an explicit supersession link
   - Pros: Records the full replacement chain.
   - Cons: Adds fields and lifecycle rules that are not required to prevent inactive invitations from affecting membership.
   - Confidence: medium

## Ambiguity: InviteEmail lifecycle when the Subject becomes inactive

Decision: Retain `Subject.InviteEmail` when cancellation or leave makes the Subject inactive. Treat it as provenance for the last controlling invitation, not as an active link, and overwrite it when a later rejoin succeeds.

- Pros: Preserves the last invitation association for diagnostics and history without requiring fallback lookup during rejoin.
- Cons: Consumers must check Subject activity before treating `InviteEmail` as the active controlling link.
- Confidence: high

Alternatives:

1. Clear `Subject.InviteEmail` when membership is cancelled or left
   - Pros: A non-empty value would always describe a current or potentially active link.
   - Cons: Loses the last invitation association and requires fallback lookup on rejoin.
   - Confidence: medium

## Ambiguity: Meaning of InviteEmail being optional for backward compatibility

Decision: Describe the persisted-data behavior explicitly: `Subject.InviteEmail` may be empty on Subjects created before the field was introduced; every successful join, rejoin, or replacement writes it; and the canonical-login/alias fallback applies when it is empty.

- Pros: States exactly which records can lack the value and how the system handles them without implying an API compatibility concern.
- Cons: Requires a longer explanation than the generic backward-compatibility phrase.
- Confidence: high

Alternatives:

1. Make `Subject.InviteEmail` required and migrate existing Subjects
   - Pros: Every Subject immediately satisfies the populated-field invariant.
   - Cons: Contradicts the selected no-migration fallback and expands rollout scope.
   - Confidence: low
