# voedger: accomodate .nolint files according to exclusions

- URL: https://untill.atlassian.net/browse/AIR-4322
- ID: AIR-4322
- State: in-progress
- Author: Denis Gribanov
- Labels: none

## Description

### Why
CI action in Voedger now provides exclusions for ci-action to be used in lint-all.sh. Need to change the approach to “convetion over configuration”

### What
create .nolint empty files in dirs to be excluded according to paths provided to ci-action
