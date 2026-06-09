# Decisions: Review feedback trigger policy

## Ambiguity: automatic review trigger scope

Decision: Automatic reviews run only on `pull_request_target` `opened` and `ready_for_review`; `opened` applies to draft and non-draft pull requests, and `ready_for_review` applies when a draft pull request is marked ready.

- Pros: matches GitHub Actions event vocabulary; reduces review-provider cost and PR-thread noise; preserves early review for both draft and non-draft PR openings; keeps a second automatic review for draft PRs when they become ready
- Cons: new commits and reopened pull requests require a Writer to request `/review`
- Confidence: high

Alternatives:

1. Keep `synchronize` and `reopened` automatic reviews
   - Pros: maximizes automatic coverage after new commits and reopened work
   - Cons: noisier and more expensive on active PRs; duplicates the manual `/review` rerun path
   - Confidence: medium
2. Run automatic reviews only on `ready_for_review`
   - Pros: minimizes review-provider usage
   - Cons: non-draft PRs opened ready for review would not receive initial automatic feedback
   - Confidence: low

## Vagueness: where successful review results appear

Decision: Successful reviews remain visible as ReviewProvider pull request reviews; PR status comments only announce start, failure, or authorization denial.

- Pros: keeps review findings in GitHub's native review UI; avoids adding duplicate success comments; clarifies that comments are operational status, not review output
- Cons: users must distinguish the status comment from the review result
- Confidence: high

Alternatives:

1. Add a success comment after every completed review
   - Pros: makes completion visible in the PR conversation
   - Cons: duplicates the review result and increases comment noise
   - Confidence: medium

## Vagueness: PR-thread status comment text

Decision: Use concise PR comments, with `/review` instructions only in the automatic review start comment:

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

- Pros: gives users immediate feedback; the first automatic comment tells Writers how to request another review; failure comments include the GitHub Actions run URL for diagnosis; emojis make status comments scannable without long text
- Cons: emojis are slightly less formal than plain status text; the automatic start comment is longer than Writer-requested start comments
- Confidence: user-provided

Alternatives:

1. Use plain text without emojis
   - Pros: most formal and consistent with engineering prose
   - Cons: less visually distinct in a busy PR thread
   - Confidence: medium
2. Include `/review` instructions in every start comment
   - Pros: keeps the manual rerun command visible on every review start
   - Cons: repeats guidance after Writer-requested reviews and increases comment noise
   - Confidence: medium
