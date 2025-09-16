# strconvu

Type-safe string conversion utilities with generic support for all
integer types and comprehensive validation.

## Problem

Go's standard library requires verbose type casting and manual
validation when converting between strings and various integer types.

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
    results[3] = strconv.FormatInt(i64, 10)

    return results
}

// String to uint8 with validation - error-prone manual checks
func parseConfig(s string) (uint8, error) {
    value, err := strconv.Atoi(s)
    if err != nil {
        return 0, err
    }
    // Easy to forget range validation or get bounds wrong
    if value < 0 || value > 255 { // common mistake: hardcoded bounds
        return 0, errors.New("out of range")
    }
    return uint8(value), nil
}

// Custom types require even more boilerplate
type Port uint16
func (p Port) String() string {
    return strconv.FormatUint(uint64(p), 10) // repetitive casting
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
        strconvu.UintToString(u8),
        strconvu.UintToString(u16),
        strconvu.IntToString(i32),
        strconvu.IntToString(i64),
    }
}

// Built-in validation with proper error messages
func parseConfig(s string) (uint8, error) {
    return strconvu.StringToUint8(s) // handles range validation automatically
}

// Custom types work seamlessly with generics
type Port uint16
func (p Port) String() string {
    return strconvu.UintToString(p) // no casting needed
}
```
</details>

## Features

- **[UintToString](impl.go#L12)** - Generic
  unsigned integer to string conversion
- **[IntToString](impl.go#L16)** - Generic
  signed integer to string conversion
- **[StringToUint8](impl.go#L20)** - String to
  uint8 with automatic range validation
- **[ParseUint64](impl.go#L30)** - String to
  uint64 parsing with error handling
- **[ParseInt64](impl.go#L33)** - String to int64 parsing with error handling

## Use

See [example](example_test.go)