---
change_id: 2607290917-safer-pull-request-trigger
type: fix
issue_url: https://untill.atlassian.net/browse/AIR-4609
domains: [devops]
---

# Change request: Safer pull request workflow triggers

Refs:

- [AIR-4609: voedger: use pull_request triggering event instead of pull_request_target](./issue-4609.md)

## Why

Pull requests from forks currently fail because repository workflows use the trusted `pull_request_target` context while downstream CI checks out pull request code. Auto-merge no longer requires that trigger, so pull request automation should use the safer `pull_request` security boundary and preserve the intended CI and review behavior.

## What

Symptom: A CI run for a pull request from a fork fails when the downstream checkout refuses to load fork code in the trusted `pull_request_target` context.

```text
contributor opens a pull request from a fork
      |
      v
.github/workflows/ci_pr.yml receives pull_request_target
      |               <-- fault: starts pull request CI in a trusted base-repository context
      v
reusable CI workflow attempts to check out fork pull request code
      |
      v
actions/checkout refuses the unsafe checkout   (symptom)
```

Corrected behavior: CI workflows that check out and execute pull request code, including code from forks, use `pull_request` triggers and read-only credentials, while separately privileged automation preserves its intended behavior without executing untrusted code.

## How

Decisions:

- Use `pull_request` for CI workflows that check out and execute contributor code, while leaving trusted `push` execution paths unchanged.
- Run pull request CI without repository secrets by granting only read access and mapping the run-scoped `GITHUB_TOKEN` to reusable workflow inputs that need repository or pull request reads.
- Keep failure issue creation and other write-capable side effects restricted to trusted push executions; pull request failures are reported only through their workflow check results.
- Retain `pull_request_target` and `issue_comment` for the separately specified PR review workflow because it requires repository secrets and write permissions and is constrained not to execute pull request scripts.
- Preserve existing path filters, changed-file classification, and test selection so the trigger migration changes the security context rather than CI coverage.

Assumptions:

- The shared `untillpro/ci-action` pull request workflow accepts the caller's run-scoped `GITHUB_TOKEN` through its existing `reporeading_token` contract, and every dependency needed by fork CI is publicly readable.

References (internal):

- [primary pull request CI boundary](../../../../../.github/workflows/ci_pr.yml)
- [storage pull request routing and change classification](../../../../../.github/workflows/ci-pkg-storage.yml)
- [Cassandra failure notification path](../../../../../.github/workflows/ci_cas.yml)
- [DynamoDB failure notification path](../../../../../.github/workflows/ci_amazon.yml)
- [privileged review workflow security contract](../../../../specs/devops/dev/arch.md)

References (external):

- [shared pull request CI contract](https://github.com/untillpro/ci-action/blob/main/.github/workflows/ci_pr.yml)
- [GitHub pull request event and fork restrictions](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#pull_request)
- [GitHub guidance for secure workflow triggers](https://docs.github.com/en/actions/reference/security/secure-use)
- [GitHub reusable workflow secret passing](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows#passing-inputs-and-secrets-to-a-reusable-workflow)

## Provisioning and configuration

GitHub Actions:

- [x] update: [primary pull request CI workflow](../../../../../.github/workflows/ci_pr.yml): replace the `pull_request_target` trigger with `pull_request`, grant only `contents: read` and `pull-requests: read`, and pass the run-scoped `GITHUB_TOKEN` as the reusable workflow's `reporeading_token` (manual edit; no CLI available)
- [x] update: [storage CI routing workflow](../../../../../.github/workflows/ci-pkg-storage.yml): replace its pull request trigger and event guard with `pull_request`, grant read-only content and pull request access, use `GITHUB_TOKEN` to classify changed files, and pass `REPOREADING_TOKEN` only on trusted push runs while using `GITHUB_TOKEN` on pull request runs (manual edit; no CLI available)
- [x] update: [Cassandra CI workflow](../../../../../.github/workflows/ci_cas.yml): create failure issues only for trusted push executions so pull request jobs have no write-capable side effects (manual edit; no CLI available)
- [x] update: [DynamoDB CI workflow](../../../../../.github/workflows/ci_amazon.yml): create failure issues only for trusted push executions so pull request jobs have no write-capable side effects (manual edit; no CLI available)
