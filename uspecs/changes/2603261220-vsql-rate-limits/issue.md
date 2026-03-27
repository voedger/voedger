# AIR-3410: Implement rate limits using VSQL

- **Source**: [AIR-3410](https://untill.atlassian.net/browse/AIR-3410)

## Summary

Implement rate limits using VSQL instead of the current appconfig method.

## Context

The application currently uses appconfig to apply rate limits. There is a need to improve this by declaring limits using VSQL.

## Acceptance criteria

- Rate limits must be implemented using VSQL
- The existing appconfig method should be replaced
- Existing integration tests should not be changed

