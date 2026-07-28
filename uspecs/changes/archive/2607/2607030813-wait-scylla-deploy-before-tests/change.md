---
change_id: 2607030808-wait-scylla-deploy-before-tests
type: ci
issue_url: https://untill.atlassian.net/browse/AIR-4407
domains: [devops]
---

# Change request: Wait for Scylla readiness in Cassandra tests

Refs:

- [AIR-4407: voedger: wait for scylla deploy before tests](./issue-AIR-4407.md)

## Why

The Cassandra test workflow can start Go tests before the Scylla service is ready to accept CQL sessions. This creates misleading TCK failures where storage-not-found expectations are masked by transient gocql protocol-discovery errors.

## What

Improve the CI validation workflow for contributors and reviewers:

- Cassandra implementation tests run only after the Scylla service is ready for CQL traffic.
- Startup-related Scylla failures are reported as infrastructure readiness problems instead of storage behavior regressions.
- Pull Request feedback from GitHub Actions becomes more deterministic for Scylla-backed storage checks.

## How

Decisions:

- Add an explicit Scylla CQL readiness wait to the Cassandra GitHub Actions workflow before any Go tests run.
- Use the existing workflow service container as the readiness target instead of changing Cassandra test code.
- Keep both Cassandra implementation tests and Cassandra-backed VVM storage tests behind the same readiness gate.

Out of scope:

- Changing Scylla image versions or Cassandra driver behavior.
- Changing storage TCK expectations or Cassandra storage semantics.

References:

- [Cassandra test workflow](../../../../../.github/workflows/ci_cas.yml)
- [Cassandra implementation tests](../../../../../pkg/istorage/cas/impl_test.go)
- [Cassandra-backed VVM storage tests](../../../../../pkg/vvm/storage/impl_elections_test.go)

## Provisioning and configuration

- [x] update: [ci_cas.yml](../../../../../.github/workflows/ci_cas.yml): add a Scylla CQL readiness wait before Cassandra-backed Go tests (manual edit - no CLI available)
  - use the existing `scylladb` service container as the readiness target
  - fail with Scylla container logs when readiness is not reached within the configured wait window
  - run both Cassandra implementation tests and Cassandra-backed VVM storage tests only after readiness succeeds
