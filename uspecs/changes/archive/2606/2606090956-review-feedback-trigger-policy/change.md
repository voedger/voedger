---
change_id: 2606090922-review-feedback-trigger-policy
type: ci
domains: [devops]
scope: [dev]
---

# Change request: Review feedback trigger policy

## Why

The automated PR review workflow should balance timely feedback with noise and review-provider cost. Reviewers need PR-thread acknowledgement when automatic or manual review automation starts, and a clear PR-thread signal when review automation fails.

## What

Adjust PR review automation behavior in the `devops` `dev` context:

- Automatic reviews run only on `pull_request_target` `opened` and `ready_for_review` events; `opened` covers draft and non-draft pull requests, and `ready_for_review` covers draft pull requests becoming ready
- New commits and reopened pull requests rely on Writer-requested `/review` reruns instead of automatic reruns
- Automatic reviews create a PR comment indicating that automated review has started and telling Writers how to request another review
- Writer-requested reviews create a PR comment indicating that automated review has started
- Failed automatic and Writer-requested reviews create a PR comment with the GitHub Actions run URL
- Successful reviews remain visible as ReviewProvider pull request reviews; status comments do not replace review results
- NonWriter `/review` requests continue to be rejected with a clear PR comment

Expected PR comments:

```text
👀 Automated review started.

Writers can request another review with `/review` or `/review <extra instructions>`.
```

```text
👀 Automated review started.
```

```text
⚠️ Automated review failed.

Details: <run-url>
```

```text
🔒 Only Writer can request an automated review with `/review`.
```

## Functional design

- [x] update: [devops/dev/pr-reviews.feature](../../../../specs/devops/dev/pr-reviews.feature)
  - remove: automatic review scenarios for `synchronize` and `reopened`
  - add: scenarios for automatic start comments with `/review` instructions and Writer-requested start comments
  - add: scenarios for failure comments with the GitHub Actions run URL
  - add: scenario or assertion that successful review results remain ReviewProvider pull request reviews
  - update: NonWriter denial scenario to expect the `🔒` denial comment text

## Technical design

- [x] update: [devops/dev/arch.md](../../../../specs/devops/dev/arch.md)
  - align automatic review trigger handling with `opened` and `ready_for_review`
  - document automatic start comments with `/review` instructions, Writer-requested start comments, failure comments, and ReviewProvider pull request review results

## Provisioning and configuration

- [x] update: [.github/workflows/pr-review.yml](../../../../../.github/workflows/pr-review.yml)
  - remove: `synchronize` and `reopened` from automatic `pull_request_target` activity types
  - add: PR status comment before automatic and Writer-requested review starts, with `/review` instructions on the automatic start comment
  - add: failure PR status comment with the GitHub Actions run URL for automatic and Writer-requested reviews
  - preserve: no success status comment; successful results remain ReviewProvider pull request reviews
  - update: NonWriter denial comment to the expected `🔒` text

## Construction

- [x] update: [.github/workflows/README.md](../../../../../.github/workflows/README.md)
  - align documented PR review workflow triggers with `opened` and `ready_for_review`
  - document automatic start comments with `/review` instructions, Writer-requested start comments, failure comments, and ReviewProvider pull request review results
