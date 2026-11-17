# Context architecture: devops/dev

Development workflow automation for CI validation and Repository Rules. This overview covers the retained validation entry points and contributor guidance.

## External actors

Roles:

- `@NonWriter`
  - Opens and updates Pull Requests with below-Write repository permission
- `@Writer`
  - Reviews Pull Requests and decides whether changes can be merged

Systems:

- `*GitHub`
  - Hosts the repository, Pull Requests, workflow events, execution results, and Repository Secrets

## Components

### Layers

```text
Contributor guidance
    [Repository rules]

Validation entry points
    [Pull request validation]
    [Main branch validation]
    [Daily validation]
    [Storage validation]

Test execution
    [Shared CI workflows]
    [Storage test workflows]
```

### Contributor guidance

- `[Repository rules]`
  - Versioned guidance for contributors and development tools, retained independently of automated Pull Request reviews
  - impl: [AGENTS.md](../../../../AGENTS.md)
  - impl: [.augment/rules/ar-common-develop.md](../../../../.augment/rules/ar-common-develop.md)
  - impl: [.augment/rules/ar-golang.md](../../../../.augment/rules/ar-golang.md)
  - impl: [.augment/rules/ar-markdown.md](../../../../.augment/rules/ar-markdown.md)

### Validation entry points

- `[Pull request validation]`
  - Uses short tests with the race detector disabled and TinyGo installation enabled; changes confined to `pkg/istorage/**` do not trigger this workflow
  - impl: [.github/workflows/ci_pr.yml](../../../../.github/workflows/ci_pr.yml)
- `[Main branch validation]`
  - Uses the same test options for pushes to `main`, excluding changes confined to `pkg/istorage/**`; successful validation precedes the delivery workflow
  - impl: [.github/workflows/ci.yml](../../../../.github/workflows/ci.yml)
- `[Daily validation]`
  - Requests full tests with the race detector and TinyGo installation enabled; failures trigger issue creation with the workflow run URL
  - impl: [.github/workflows/ci-full.yml](../../../../.github/workflows/ci-full.yml)
- `[Storage validation]`
  - Handles Pull Request and push events affecting storage or elections paths and selects backend test workflows using changed-file outputs and job conditions
  - impl: [.github/workflows/ci-pkg-storage.yml](../../../../.github/workflows/ci-pkg-storage.yml)

### Test execution

- `[Shared CI workflows]`
  - Reusable workflows referenced from `untillpro/ci-action` at `main`; local callers supply test options and credentials
  - impl: [.github/workflows/ci_pr.yml](../../../../.github/workflows/ci_pr.yml)
  - impl: [.github/workflows/ci.yml](../../../../.github/workflows/ci.yml)
  - impl: [.github/workflows/ci-full.yml](../../../../.github/workflows/ci-full.yml)
- `[Storage test workflows]`
  - Run backend and storage tests with the race detector against ScyllaDB or DynamoDB Local services; push-triggered failures invoke issue creation
  - impl: [.github/workflows/ci_cas.yml](../../../../.github/workflows/ci_cas.yml)
  - impl: [.github/workflows/ci_amazon.yml](../../../../.github/workflows/ci_amazon.yml)

## Scenarios

### Validating pull requests

```text
@NonWriter or @Writer opens or updates a Pull Request
  -> *GitHub: pull_request event
      -> [Pull request validation]: when its path filter matches
          -> [Shared CI workflows]: short tests
      -> [Storage validation]: when its path filter matches
          -> [Storage test workflows]: selected backend jobs
```

### Validating main branch changes

```text
*GitHub: push to main
  -> [Main branch validation]: when its path filter matches
      -> [Shared CI workflows]: short tests
  -> [Storage validation]: when its path filter matches
      -> [Storage test workflows]: selected backend jobs
```

### Running full validation

```text
*GitHub: daily schedule at 05:00 UTC or workflow_dispatch
  -> [Daily validation]
      -> [Shared CI workflows]: full tests with race detection
```

## Cross-cutting concerns

### Permissions

- `[Pull request validation]` and `[Storage validation]` declare `contents: read` and `pull-requests: read` permissions
- `[Pull request validation]` passes `GITHUB_TOKEN` to `[Shared CI workflows]`; `[Storage validation]` passes `GITHUB_TOKEN` for Pull Requests and `REPOREADING_TOKEN` for pushes

### Repository scope

- `[Pull request validation]`, `[Main branch validation]`, and `[Daily validation]` invoke `[Shared CI workflows]` only in `voedger/voedger`
