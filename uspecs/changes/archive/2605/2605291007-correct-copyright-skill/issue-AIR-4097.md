# implement skill that will force to produce the correct copyright

- URL: https://untill.atlassian.net/browse/AIR-4097
- ID: AIR-4097
- State: in-progress
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov

## Why

AI generates wrong copyright for new files like:

```text
/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */
```

## What

implement  skill that will force to produce the correct header comment for new files:

```text
/*
 * Copyright (c) <current year>-present unTill Software Development Group B.V.
 * @author <current git user>
 */
```

same for sql\vsql
