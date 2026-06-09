---
change_id: 2606090756-augment-pr-review-workflow
type: ci
issue_url: https://github.com/augmentcode/review-pr
domains: [devops]
scope: [dev]
---

# Change request: Automated reviews for pull requests

## Why

Pull requests should receive fast automated review feedback before human review starts. Repository automation is part of the engineering operations domain rather than the Voedger production runtime, so the intended behavior should be captured in a dedicated `devops` specification before the workflow is added. The repository already carries review guidance for human developers and AI agents, so the PR review automation should reuse the same project guidance instead of relying only on generic review criteria.

## What

Introduce specified automated PR review behavior for repository participants:

- The `devops` domain describes Voedger engineering operations, including repository automation used by NonWriters and Writers
- The `dev` context contains the `pr-reviews` feature, which defines when automated reviews are created and who can request them
- Every draft or non-draft pull request receives an automated review when it is opened, updated, reopened, or marked ready for review
- Writer can request another review by creating a PR comment whose trimmed body starts with `/review` as a command
- Manual comment-triggered reviews are ignored on non-PR issues, edited or deleted comments, comments where `/review` is not the leading command, command-prefix lookalikes such as `/reviewed`, and comments from NonWriters
- Reviews use the repository's existing development, Go, and Markdown guidance
- The workflow uses a repository secret for review-provider authentication and keeps GitHub token permissions limited to reading contents, writing pull request reviews, and writing command feedback comments

## How

Decisions:

- Add the `devops` domain and place `pr-reviews` under the `dev` context before adding workflow automation
- Add a `dev` context architecture document to describe PR review workflow boundaries, triggers, permissions, and ReviewProvider integration
- Add a standalone PR review workflow instead of modifying existing CI workflows
- Use `pull_request_target` for automatic PR reviews and `issue_comment` `created` events for the `/review` command
- Use `augmentcode/review-pr@v0.2.0` with `AUGMENT_SESSION_AUTH`, `GITHUB_TOKEN`, and the repository's Augment rule files
- Keep workflow permissions to `contents: read`, `pull-requests: write`, and `issues: write` for command feedback comments; serialize review runs per pull request

Out of scope:

- Replacing required human review or changing branch protection
- Changing existing CI test, lint, storage, or deployment workflows
- Managing the lifecycle of the personal Augment session credential outside GitHub secrets

References (internal):

- [workflow overview](../../../../../.github/workflows/README.md)
- [current PR workflow trigger pattern](../../../../../.github/workflows/ci_pr.yml)
- [storage PR workflow trigger pattern](../../../../../.github/workflows/ci-pkg-storage.yml)
- [common Augment development rules](../../../../../.augment/rules/ar-common-develop.md)
- [Go Augment rules](../../../../../.augment/rules/ar-golang.md)
- [Markdown Augment rules](../../../../../.augment/rules/ar-markdown.md)

References (external):

- [Augment PR review action v0.2.0](https://raw.githubusercontent.com/augmentcode/review-pr/v0.2.0/action.yml)
- [GitHub `pull_request_target` event](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#pull_request_target)
- [GitHub `issue_comment` event](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#issue_comment)

## Domain specifications

- [x] create: [devops/domain.md](../../../../specs/devops/domain.md)
  - Domain Specification for Voedger engineering operations: actors, core concepts, and the `dev` context for repository development automation

## Functional design

- [x] create: [devops/dev/pr-reviews.feature](../../../../specs/devops/dev/pr-reviews.feature)
  - Feature specification with scenarios for automatic PR reviews and authorized `/review` comment-triggered reruns

## Provisioning and configuration

- [x] configure: GitHub repository secret `AUGMENT_SESSION_AUTH`
  - Store Augment session JSON credentials from `auggie tokens print` or `~/.augment/session.json` as a GitHub Actions repository secret

- [x] create: [.github/workflows/pr-review.yml](../../../../../.github/workflows/pr-review.yml)
  - Add standalone PR review workflow for `pull_request_target` pull request lifecycle events and `issue_comment` `created` events
  - Use `augmentcode/review-pr@v0.2.0` for automated review generation with `AUGMENT_SESSION_AUTH`, `GITHUB_TOKEN`, pull request number, repository name, and repository rule files
  - Gate `/review` comment-triggered reviews to Writer and return a clear message for NonWriter requests
  - Keep workflow token permissions scoped to `contents: read`, `pull-requests: write`, and `issues: write` for user-facing command feedback
  - Serialize review runs per pull request

## Technical design

- [x] create: [devops/dev/arch.md](../../../../specs/devops/dev/arch.md)
  - Context Architecture: PR review workflow boundaries, trigger routing, Writer authorization, command feedback, concurrency, and ReviewProvider integration

## Construction

- [x] update: [.github/workflows/README.md](../../../../../.github/workflows/README.md)
  - add `pr-review.yml` to the workflow overview table with `pull_request_target` and `issue_comment` triggers
  - add workflow details for automatic reviews, Writer `/review` requests, NonWriter feedback, repository rules, and ReviewProvider integration
