# strconvu

Type-safe string conversion utilities for Go integer types using generics.

## Problem

Standard library string conversions require explicit type casting and 
lack compile-time type safety for custom integer types.

## Features

- **[UintToString](impl.go#L14)** - Generic unsigned integer to string conversion
  - [Generic type constraints: impl.go#L14](impl.go#L14)
  - [Decimal base formatting: impl.go#L15](impl.go#L15)
- **[IntToString](impl.go#L18)** - Generic signed integer to string conversion
  - [Generic type constraints: impl.go#L18](impl.go#L18)
  - [Decimal base formatting: impl.go#L19](impl.go#L19)
- **[StringToUint8](impl.go#L22)** - String to uint8 with range validation
  - [Range validation logic: impl.go#L27](impl.go#L27)
  - [Error handling strategy: impl.go#L28](impl.go#L28)

## Use

See [example usage](example_test.go)
