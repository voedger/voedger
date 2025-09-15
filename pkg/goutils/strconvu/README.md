# strconvu

Type-safe string conversion utilities for Go integer types using generics.

## Problem

Standard library string conversions require explicit type casting and
lack compile-time type safety for custom integer types.

## Features

- **[UintToString](impl.go#L12)** - Generic unsigned integer to string conversion
  - [Generic type constraints: impl.go#L12](impl.go#L12)
  - [Decimal base formatting: impl.go#L13](impl.go#L13)
- **[IntToString](impl.go#L16)** - Generic signed integer to string conversion
  - [Generic type constraints: impl.go#L16](impl.go#L16)
  - [Decimal base formatting: impl.go#L17](impl.go#L17)
- **[ParseUint8](impl.go#L20)** - String to uint8 parsing
  - [Decimal base parsing: impl.go#L21](impl.go#L21)
  - [8-bit size constraint: impl.go#L21](impl.go#L21)
- **[ParseUint64](impl.go#L25)** - String to uint64 parsing
  - [Decimal base parsing: impl.go#L26](impl.go#L26)
  - [64-bit size constraint: impl.go#L26](impl.go#L26)
- **[ParseInt64](impl.go#L29)** - String to int64 parsing
  - [Decimal base parsing: impl.go#L30](impl.go#L30)
  - [64-bit size constraint: impl.go#L30](impl.go#L30)

## Use

See [example usage](example_test.go)
