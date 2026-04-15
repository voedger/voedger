# AIR-3578: httpu: Fix body loss on retry in HTTPClient

- **Type:** Task
- **Status:** In Progress
- **Assignee:** d.gribanov@dev.untill.com

## Description

**Why**

Case:

- Want to execute a request using `IHTTPClient` with some retry policy
- Request body is provided as `BodyReader`
- Retry occurs
- On next iteration the `BodyReader` is read out already
- Result → the further retries are done with empty body

**What**

- Implement a test that reproduces the problem
- Prevent the body loss on retries
