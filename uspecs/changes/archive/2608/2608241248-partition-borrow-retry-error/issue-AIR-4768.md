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

At this point, `err` is always `nil`. Consequently, `ErrNotAvailableEngines` is never recognized and borrowing is retried indefinitely. This can cause processors to hang when an engine pool is exhausted and can amplify re-entrant scheduler failures.

## What

Update the retry handler to inspect `opErr`:

```
if !errors.Is(opErr, ErrNotAvailableEngines)
```

Add a regression test verifying that:

* `ErrNotAvailableEngines` aborts immediately and is returned to the caller.
* Other transient borrow errors continue to be retried.
* The behavior applies to all processor engine pools, including schedulers.

