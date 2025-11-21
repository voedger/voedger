# Package strconvu

Type-safe string conversion utilities with generic support for all
integer types and comprehensive validation.

## Problem

Go's standard library requires verbose type casting and magic numbers
when converting between strings and various integer types.

<details>
<summary>Without strconvu</summary>

```go
// Converting different integer types to strings - boilerplate everywhere
func processNumbers(u8 uint8, u16 uint16, i32 int32, i64 int64) []string {
    results := make([]string, 4)

    // Each type needs explicit casting to uint64/int64
    results[0] = strconv.FormatUint(uint64(u8), 10)  // boilerplate cast
    results[1] = strconv.FormatUint(uint64(u16), 10) // boilerplate cast
    results[2] = strconv.FormatInt(int64(i32), 10)   // boilerplate cast
    results[3] = strconv.FormatInt(i64, 10)          // magic number 10

    return results
}

// String to uint8 with validation - error-prone manual checks
func parseConfig(s string) (uint8, error) {
    value, err := strconv.ParseUint(s, 10, 8) // magic numbers everywhere
    if err != nil {
        return 0, err
    }
    // Easy to forget proper casting or get bit sizes wrong
    return uint8(value), nil // potential overflow if bitSize wrong
}

// Custom types require even more boilerplate
type Port uint16
func (p Port) String() string {
    return strconv.FormatUint(uint64(p), 10) // repetitive casting
}

// Multiple parsing functions with repeated validation logic
func parseMultiple(s1, s2, s3 string) (uint8, uint64, int64, error) {
    v1, err := strconv.ParseUint(s1, 10, 8)  // repeated base/bitSize
    if err != nil {
        return 0, 0, 0, err
    }
    v2, err := strconv.ParseUint(s2, 10, 64) // repeated base/bitSize
    if err != nil {
        return 0, 0, 0, err
    }
    v3, err := strconv.ParseInt(s3, 10, 64)  // repeated base/bitSize
    if err != nil {
        return 0, 0, 0, err
    }
    return uint8(v1), v2, v3, nil // manual casting
}
```

</details>

<details>
<summary>With strconvu</summary>

```go
import "github.com/voedger/voedger/pkg/goutils/strconvu"

// Clean, type-safe conversions for any integer type
func processNumbers(u8 uint8, u16 uint16, i32 int32, i64 int64) []string {
    return []string{
        strconvu.UintToString(u8),   // no casting needed
        strconvu.UintToString(u16),  // works with any uint type
        strconvu.IntToString(i32),   // works with any int type
        strconvu.IntToString(i64),   // consistent API
    }
}

// Built-in validation with proper error messages
func parseConfig(s string) (uint8, error) {
    return strconvu.ParseUint8(s) // handles validation automatically
}

// Custom types work seamlessly with generics
type Port uint16
func (p Port) String() string {
    return strconvu.UintToString(p) // no casting needed
}

// Clean parsing with consistent error handling
func parseMultiple(s1, s2, s3 string) (uint8, uint64, int64, error) {
    v1, err := strconvu.ParseUint8(s1)  // type-specific validation
    if err != nil {
        return 0, 0, 0, err
    }
    v2, err := strconvu.ParseUint64(s2) // consistent API
    if err != nil {
        return 0, 0, 0, err
    }
    v3, err := strconvu.ParseInt64(s3)  // no magic numbers
    if err != nil {
        return 0, 0, 0, err
    }
    return v1, v2, v3, nil // no manual casting
}
```

</details>

## Features

- **[UintToString](impl.go#L12)** - Generic unsigned integer to
  string conversion
- **[IntToString](impl.go#L17)** - Generic signed integer to string
  conversion
- **[ParseUint8](impl.go#L22)** - String to uint8 parsing with
  automatic range validation
- **[ParseUint64](impl.go#L31)** - String to uint64 parsing with
  error handling
- **[ParseInt32](impl.go#L36)** - String to int32 parsing with automatic range validation
- **[ParseInt64](impl.go#L45)** - String to int64 parsing with error
  handling

## Use

See [example](example_test.go)
