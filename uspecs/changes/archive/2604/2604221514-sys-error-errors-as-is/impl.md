# Implementation plan: Support errors.As and errors.Is by coreutils.SysError

## Construction

- [x] update: [pkg/coreutils/syserror.go](../../../../../pkg/coreutils/syserror.go)
  - add: `Is(target error) bool` method on `SysError`
  - add: `As(target any) bool` method on `SysError`
- [x] update: [pkg/coreutils/syserror_test.go](../../../../../pkg/coreutils/syserror_test.go)
  - add: tests for `errors.Is` between equal and non-equal `SysError` values
  - add: tests for `errors.As` into `SysError` target
