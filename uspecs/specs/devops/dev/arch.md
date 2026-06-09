# Context architecture: devops/dev

Development workflow automation for repository pull request review feedback. The context owns GitHub event routing for automated PR reviews, Writer-authorized manual review requests, NonWriter command feedback, and ReviewProvider invocation with repository rules.

## External actors

Roles:

- `@NonWriter`
  - User below Write repository permission who can open and update pull requests, but cannot request automated review.

- `@Writer`
  - User with Write or higher repository permission who reviews pull requests and can request additional automated review.

Systems:

- `*GitHub`
  - Hosts pull requests, comments, repository secrets, repository rules, workflow events, workflow permissions, and published pull request reviews.

- `*ReviewProvider`
  - Produces automated review feedback from pull request changes, repository context, repository rules, and optional extra review instructions.

## Components

### Layers

```text
External actors and systems
    |
    +-- @NonWriter
    +-- @Writer
    +-- *GitHub
    +-- *ReviewProvider
    |
    v
Workflow boundary
    |
    +-- [[PR review workflow]]
    |
    v
Trigger handling
    |
    +-- [Automatic review job]
    +-- [Comment review job]
    |
    v
Command handling
    |
    +-- [Review command parser]
    +-- [Writer permission check]
    +-- [NonWriter feedback]
    |
    v
Provider integration
    |
    +-- [Review action]
    +-- [Repository rules]
    +-- [(Augment session secret)]
```

### Workflow boundary

- `[[PR review workflow]]`
  - Standalone GitHub Actions workflow that owns the `dev` context's pull request review automation without modifying CI, storage, deployment, or branch-protection workflows.
  - impl: [.github/workflows/pr-review.yml#PR review](../../../../.github/workflows/pr-review.yml)

### Trigger handling

- `[Automatic review job]`
  - Runs only for `pull_request_target` events in `voedger/voedger` when a pull request is opened or marked ready for review. It posts a start comment that tells Writers how to request another review, sends the pull request number and repository name to `[Review action]`, and posts a failure comment with the GitHub Actions run URL if `[Review action]` fails.
  - impl: [.github/workflows/pr-review.yml#jobs.automatic-review](../../../../.github/workflows/pr-review.yml)

- `[Comment review job]`
  - Runs only for `issue_comment` `created` events in `voedger/voedger` whose issue is a pull request. It routes created PR comments through `[Review command parser]`, ignores issue comments that are not attached to pull requests, and never receives edited or deleted comments.
  - impl: [.github/workflows/pr-review.yml#jobs.comment-review](../../../../.github/workflows/pr-review.yml)

### Command handling

- `[Review command parser]`
  - Trims the PR comment body, accepts exactly `/review` or `/review` followed by whitespace as the leading command, rejects command-prefix lookalikes such as `/reviewed`, and extracts the remaining text as extra review instructions.
  - impl: [.github/workflows/pr-review.yml#steps.Parse review command](../../../../.github/workflows/pr-review.yml)

- `[Writer permission check]`
  - Resolves the commenter permission through the GitHub collaborator permission API. `admin`, `maintain`, and `write` are treated as `@Writer`; every other permission, missing collaborator record, or API lookup failure is treated as `@NonWriter`.
  - impl: [.github/workflows/pr-review.yml#steps.Check commenter permission](../../../../.github/workflows/pr-review.yml)

- `[NonWriter feedback]`
  - Posts a PR comment explaining that only `@Writer` can request automated review when a `@NonWriter` creates a valid `/review` command.
  - impl: [.github/workflows/pr-review.yml#steps.Reply to NonWriter](../../../../.github/workflows/pr-review.yml)

### Provider integration

- `[Review action]`
  - Calls `augmentcode/review-pr@v0.2.0` with `[(Augment session secret)]`, `GITHUB_TOKEN`, pull request number, repository name, and `[Repository rules]`. Manual reviews also pass extracted extra instructions as custom guidelines. Successful results remain ReviewProvider pull request reviews; status comments do not replace review results.
  - impl: [.github/workflows/pr-review.yml#uses.augmentcode/review-pr@v0.2.0](../../../../.github/workflows/pr-review.yml)

- `[Repository rules]`
  - Versioned review guidance supplied to `*ReviewProvider`.
  - impl: [.augment/rules/ar-common-develop.md](../../../../.augment/rules/ar-common-develop.md)
  - impl: [.augment/rules/ar-golang.md](../../../../.augment/rules/ar-golang.md)
  - impl: [.augment/rules/ar-markdown.md](../../../../.augment/rules/ar-markdown.md)

- `[(Augment session secret)]`
  - GitHub Actions repository secret `AUGMENT_SESSION_AUTH` containing ReviewProvider session credentials. The secret is consumed only by `[Review action]`.

## Scenarios

### Automatic pull request review

```text
*GitHub
  -> [[PR review workflow]] (pull_request_target event)
  -> [Automatic review job]
  -> *GitHub (PR start comment with `/review` instructions)
  -> [Review action]
  -> [(Augment session secret)]
  -> [Repository rules]
  -> *ReviewProvider
  -> *GitHub (pull request review)
```

### Writer requests another review

```text
@Writer creates PR comment "/review {extra instructions}"
  -> *GitHub
  -> [[PR review workflow]] (issue_comment.created event)
  -> [Comment review job]
  -> [Review command parser] extracts extra instructions
  -> [Writer permission check] resolves write-or-higher permission
  -> *GitHub (PR start comment)
  -> [Review action]
  -> [(Augment session secret)]
  -> [Repository rules]
  -> *ReviewProvider
  -> *GitHub (pull request review)
```

### NonWriter requests another review

```text
@NonWriter creates PR comment "/review"
  -> *GitHub
  -> [[PR review workflow]] (issue_comment.created event)
  -> [Comment review job]
  -> [Review command parser] accepts the command
  -> [Writer permission check] resolves below-Write permission
  -> [NonWriter feedback]
  -> *GitHub (PR comment)
```

### Ignored comment

```text
*GitHub
  -> [[PR review workflow]] (issue_comment.created event)
  -> [Comment review job]
  -> [Review command parser] rejects non-leading command, command-prefix lookalike, or non-PR issue comment
  -> stop without [Review action]
```

## Cross-cutting concerns

### Security

- Every Component in `Trigger handling` runs only in `voedger/voedger` and must not execute scripts from the pull request head.
- Every `[Review action]` invocation receives only `contents: read`, `pull-requests: write`, and `issues: write` through `GITHUB_TOKEN`.
- Every `[(Augment session secret)]` use passes the secret directly to `[Review action]`; no Component may print or transform the secret.
- Every `[Writer permission check]` failure must resolve to `@NonWriter`.

### Concurrency

- Every Component in `Trigger handling` serializes review runs by pull request number using the `pr-review-{number}` concurrency group.
- No review run is cancelled in progress by a newer review request for the same pull request.

### Configuration

- Every `[Review action]` invocation uses the same `[Repository rules]` set so automatic and Writer-requested reviews apply the same repository guidance.
- Manual extra instructions narrow a single Writer-requested review and do not alter `[Repository rules]`.

### Context dependencies

- The `dev` context depends on `*GitHub` for event delivery, permission lookup, secret storage, and review/comment publication.
- The `dev` context depends on `*ReviewProvider` for review generation; provider-specific credentials and action versioning stay behind `[Review action]`.
