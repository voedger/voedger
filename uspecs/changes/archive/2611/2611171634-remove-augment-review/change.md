---
change_id: 2611171403-remove-augment-review
type: ci
issue_url: https://untill.atlassian.net/browse/AIR-5286
domains: [devops]
scope: [dev]
breaking: true
---

# Change request: Removal of Augment pull request reviews

Refs:

- [AIR-5286: devops: eliminate augment review step on pull request](./issue-AIR-5286.md)

## Why

The repository currently invokes Augment as part of its Pull Request review automation. Not required anymore.

## What

- Pull Request events no longer trigger Augment reviews through repository automation
- Comment-triggered Review Commands no longer invoke Augment or report an Augment review as queued or running
- Contributor-facing workflow documentation and specifications reflect the removal of the Augment review integration

## How

Decisions:

- Remove the dedicated review workflow as a unit, including its command handling, permission checks, and status comments, because it contains no independent CI responsibilities
- Retire Augment-specific provider relationships and review-command contracts in active specifications while retaining the broader engineering operations domain and development context
- Preserve archived change requests as historical records and retain reusable repository rules used by development tools
- Validate removal through workflow and reference checks; verify that unrelated CI workflows retain their existing triggers and jobs

Assumptions:

- GitHub branch protection and rulesets do not require the retiring review jobs as merge checks; this must be verified before rollout

Out of scope:

- Introducing a replacement automated review provider
- Deleting hosted repository secrets or changing external Augment account settings
- Removing local Augment tooling or its repository rules

References:

- [Retired review automation and its triggers](https://github.com/voedger/voedger/blob/c7d135815a66507f60d32fa09df926ffd7292bdc/.github/workflows/pr-review.yml)
- [Contributor workflow documentation](../../../../../.github/workflows/README.md)
- [Engineering operations scope and provider relationships](../../../../specs/devops/domain.md)
- [Retired automatic and manual review contract](https://github.com/voedger/voedger/blob/c7d135815a66507f60d32fa09df926ffd7292bdc/uspecs/specs/devops/dev/pr-reviews.feature)
- [Development context architecture](../../../../specs/devops/dev/arch.md)

## Domain design

- [x] update: [devops/domain.md](../../../../specs/devops/domain.md)
  - Remove the ReviewProvider external system and its review-request relationships from the domain model
  - Update NonWriter and Writer role descriptions to retain their repository participation and review responsibilities without automated review request permissions
  - Remove the Review Command concept and update Repository Rule terminology to describe retained development guidance
  - Update the dev context description and relationship diagram to remove automatic review triggers, requested reruns, and review publication by the retired integration while preserving the context and broader engineering operations scope

## Functional design

- [x] remove: [dev/pr-reviews.feature](https://github.com/voedger/voedger/blob/c7d135815a66507f60d32fa09df926ffd7292bdc/uspecs/specs/devops/dev/pr-reviews.feature): retire the feature specification for automatic Augment reviews, comment-triggered requests, permission feedback, and review status comments

## Provisioning and configuration

### Pull request review automation

- [x] Verify: Inspect main branch protection and applicable GitHub rulesets through read-only GitHub API calls for required checks supplied by the retiring review workflow, and record any dependency or access limitation before rollout
  - Checked on 2026-11-17: effective main branch rules and the active repository ruleset contain no required status checks or Augment workflow dependencies; the existing Copilot review rule remains unchanged
  - Verification limit: legacy branch protection and required-status-check endpoints both return HTTP 404 with the current non-admin credentials, so legacy protection could not be independently confirmed

- [x] remove: [workflows/pr-review.yml](https://github.com/voedger/voedger/blob/c7d135815a66507f60d32fa09df926ffd7292bdc/.github/workflows/pr-review.yml)
  - Remove the dedicated workflow using `git rm -- .github/workflows/pr-review.yml`, including both review jobs and their comment handling
  - Verify that remaining workflow configurations contain no Augment review action invocations or references to its session secret
  - Verify that other workflow configurations and local Augment rules remain unchanged

## Technical design

- [x] update: [dev/arch.md](../../../../specs/devops/dev/arch.md)
  - Retain the Context Architecture for the existing dev Context
  - Remove the retired Augment workflow components, review-command flows, provider integration, and associated security and concurrency rules
  - Describe retained CI validation workflows and Repository Rules using existing repository files as implementation references

## Construction

- [x] update: [.github/workflows/README.md](../../../../../.github/workflows/README.md)
  - Remove the retired `pr-review.yml` workflow from the overview and remove its detailed review automation section
  - Remove descriptions of automatic Augment reviews, `/review` commands, permission feedback, status comments, provider credentials, and review concurrency
  - Preserve documentation for unrelated workflows and retain local Repository Rules
  - Verify that the workflow documentation contains no active references to the retired workflow, Augment review action, or session secret
