# strconvu

Type-safe string conversion utilities for Go integer types with
generic support and validation.

## Problem

Standard library string conversion requires explicit type casting
and lacks validation for specific numeric ranges like uint8.

## Features

- **Generic conversion** - Convert any integer type to string
  - [Generic type constraints: impl.go#L14](pkg/goutils/strconvu/impl.go#L14)
  - [Generic type constraints: impl.go#L18](pkg/goutils/strconvu/impl.go#L18)
- **Validated parsing** - Parse strings to uint8 with range checking
  - [Range validation logic: impl.go#L27](pkg/goutils/strconvu/impl.go#L27)
  - [Error handling strategy: impl.go#L28](pkg/goutils/strconvu/impl.go#L28)

## Use

See [example usage](example_test.go)
