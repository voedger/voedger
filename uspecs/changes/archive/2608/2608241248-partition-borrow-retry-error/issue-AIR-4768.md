# voedger, appparts: fix wrong error checked in partition borrow retry handler

- URL: https://untill.atlassian.net/browse/AIR-4768
- ID: AIR-4768
- State: in-progress
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov

## Why

The `WaitForBorrow` retry handler checks the named return variable `err` instead of the received operation error `opErr` [here](https://github.com/voedger/voedger/blob/0c964a0d0ce3ee1f2b493615c7a2670ff018a19b/pkg/appparts/impl.go#L63):

```
if !errors.Is(err, ErrNotAvailableEngines)
```

At this point, `err` is always `nil`, so the condition is always true and every borrow error is retried. Retrying `ErrNotAvailableEngines` is intentional because an engine can become available after another processor releases it. The defect is that permanent errors, such as an unknown application or partition, are also suppressed and retried until the caller's context is cancelled.

## What

Update the retry handler to inspect `opErr` and retry only engine-availability errors:

```
if errors.Is(opErr, ErrNotAvailableEngines) {
    return true, nil
}
return false, opErr
```

Add a regression test verifying that:

* `ErrNotAvailableEngines` continues to be retried.
* Every other borrow error aborts immediately and is returned to the caller.
* The behavior applies to all processor engine pools, including schedulers.
