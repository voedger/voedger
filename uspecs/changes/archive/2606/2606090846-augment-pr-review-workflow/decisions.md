# Decisions: Automated reviews for pull requests

## Vagueness: the `pr-reviews` feature has no owning context

Decision: Place `pr-reviews` in a generalized `dev` context inside the `devops` domain.

- Pros: keeps the domain focused on engineering operations while giving development automation a stable bounded context; leaves room for future development workflow features beyond pull request review and future review providers beyond the current implementation
- Cons: the context is broader than the initial PR review feature and will need discipline to avoid becoming a catch-all
- Confidence: user-provided

Alternatives:

1. Place `pr-reviews` directly under the `devops` domain without a context
   - Pros: fewer folders and less structure for the first devops specification
   - Cons: does not follow the uspecs model where features belong to exactly one context; creates a layout that would need reshaping as more devops features appear
   - Confidence: low
2. Place `pr-reviews` in a dedicated `pull-requests` context
   - Pros: precise fit for the initial feature and its lifecycle events
   - Cons: too narrow if the devops domain is expected to group broader development workflows under one context
   - Confidence: high
3. Place `pr-reviews` under a broader `repository` context
   - Pros: could cover issues, branches, releases, and repository settings later
   - Cons: less focused on the development workflow and more likely to mix unrelated repository administration concerns
   - Confidence: medium

## Ambiguity: whether automatic reviews include draft pull requests

Decision: Review both draft and non-draft pull requests when they are opened, updated, reopened, or marked ready for review.

- Pros: matches the requested "all PRs" scope; gives early feedback before a draft is ready for human review; `pull_request_target` supports the selected activity types
- Cons: may spend review-provider quota on work that is still incomplete
- Confidence: high

Alternatives:

1. Skip draft pull requests until they are marked ready for review
   - Pros: avoids reviewing intentionally incomplete work
   - Cons: contradicts the requested "all PRs" scope; delays feedback that could help while the PR is still being shaped
   - Confidence: low
2. Review only opened and synchronized pull requests
   - Pros: simpler trigger set
   - Cons: misses explicit ready-for-review transitions and makes draft handling less visible in the spec
   - Confidence: medium

## Ambiguity: what PR comment text triggers a manual review

Decision: A manual review is triggered only by a newly created PR comment from Writer whose trimmed body starts with `/review` as a command.

- Pros: gives Writer a short command; aligns authorization with a domain-defined role backed by GitHub Write or higher repository permission; supports natural command comments and optional trailing text; avoids accidental triggers when the phrase is quoted or mentioned later in a discussion; excludes command-prefix lookalikes such as `/reviewed`; uses the `issue_comment` event only for PR comments
- Cons: users must put the command at the start of the comment; edited comments will not trigger a review
- Confidence: user-provided

Alternatives:

1. Trigger when the comment contains `/review` anywhere
   - Pros: convenient for casual comments
   - Cons: accidental triggers are likely when people quote instructions or discuss the command
   - Confidence: low
2. Trigger only when the entire comment is exactly `/review`
   - Pros: maximally explicit and easy to test
   - Cons: prevents harmless trailing context such as `/review after latest push`
   - Confidence: medium
3. Trigger when the trimmed body starts with `/augment review`
   - Pros: explicit command ownership and lower chance of collision with future bots or repository automation
   - Cons: longer to type than `/review`
   - Confidence: high
