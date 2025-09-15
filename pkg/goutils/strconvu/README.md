# strconvu

Type-safe string conversion utilities for Go integer types using
generics.

## Problem

Standard library string conversions require verbose type casting and
lack compile-time type safety for custom integer types.

<details>
<summary>Without strconvu</summary>

```go
type UserID uint64
type OrderID int64

func processOrder(userStr, orderStr string) error {
    // Verbose casting for custom types - boilerplate here
    userVal, err := strconv.ParseUint(userStr, 10, 64)
    if err != nil {
        return fmt.Errorf("invalid user ID: %w", err)
    }
    userID := UserID(userVal) // Manual casting required

    // Different functions for different sizes - easy to get wrong
    orderVal, err := strconv.ParseInt(orderStr, 10, 64)
    if err != nil {
        return fmt.Errorf("invalid order ID: %w", err)
    }
    orderID := OrderID(orderVal) // More manual casting

    // Converting back requires more casting
    log.Printf("User %s placed order %s",
        strconv.FormatUint(uint64(userID), 10), // Verbose conversion
        strconv.FormatInt(int64(orderID), 10))  // More verbose conversion

    return nil
}
```
</details>

<details>
<summary>With strconvu</summary>

```go
type UserID uint64
type OrderID int32

func processOrder(userStr, orderStr string) error {
    userID, err := ParseUint64(userStr)
    if err != nil {
        return fmt.Errorf("invalid user ID: %w", err)
    }

    orderID, err := ParseInt64(orderStr)
    if err != nil {
        return fmt.Errorf("invalid order ID: %w", err)
    }

    log.Printf("User %s placed order %s",
        UintToString(UserID(userID)), IntToString(OrderID(orderID)))

    return nil
}
```
</details>

## Features

- **[UintToString](impl.go#L14)** - Generic unsigned integer to string
  - [Generic type constraints: impl.go#L14](impl.go#L14)
- **[IntToString](impl.go#L18)** - Generic signed integer to string
  - [Generic type constraints: impl.go#L18](impl.go#L18)
- **[StringToUint8](impl.go#L22)** - String to uint8 with validation
  - [Range validation logic: impl.go#L27](impl.go#L27)
- **[ParseUint64](impl.go#L33)** - String to uint64 parsing
- **[ParseInt64](impl.go#L37)** - String to int64 parsing

## Use

See [example usage](example_test.go)
