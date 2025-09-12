/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package strconvu

import (
	"fmt"
)

// ExampleUintToString demonstrates converting various unsigned integer types to strings
func ExampleUintToString() {
	// Basic unsigned integer types
	var u8 uint8 = 255
	var u16 uint16 = 65535
	var u32 uint32 = 4294967295
	var u64 uint64 = 18446744073709551615
	var u uint = 42

	fmt.Println("uint8:", UintToString(u8))
	fmt.Println("uint16:", UintToString(u16))
	fmt.Println("uint32:", UintToString(u32))
	fmt.Println("uint64:", UintToString(u64))
	fmt.Println("uint:", UintToString(u))

	// Custom types based on unsigned integers
	type CustomUint32 uint32
	type CustomUint64 uint64

	var custom32 CustomUint32 = 1000
	var custom64 CustomUint64 = 2000

	fmt.Println("CustomUint32:", UintToString(custom32))
	fmt.Println("CustomUint64:", UintToString(custom64))

	// Output:
	// uint8: 255
	// uint16: 65535
	// uint32: 4294967295
	// uint64: 18446744073709551615
	// uint: 42
	// CustomUint32: 1000
	// CustomUint64: 2000
}

// ExampleIntToString demonstrates converting various signed integer types to strings
func ExampleIntToString() {
	// Basic signed integer types
	var i8 int8 = -128
	var i16 int16 = -32768
	var i32 int32 = -2147483648
	var i64 int64 = -9223372036854775808
	var i int = -42

	fmt.Println("int8:", IntToString(i8))
	fmt.Println("int16:", IntToString(i16))
	fmt.Println("int32:", IntToString(i32))
	fmt.Println("int64:", IntToString(i64))
	fmt.Println("int:", IntToString(i))

	// Positive values
	var pos8 int8 = 127
	var pos16 int16 = 32767
	var pos32 int32 = 2147483647
	var pos64 int64 = 9223372036854775807

	fmt.Println("positive int8:", IntToString(pos8))
	fmt.Println("positive int16:", IntToString(pos16))
	fmt.Println("positive int32:", IntToString(pos32))
	fmt.Println("positive int64:", IntToString(pos64))

	// Custom types based on signed integers
	type CustomInt32 int32
	type CustomInt64 int64

	var custom32 CustomInt32 = -1000
	var custom64 CustomInt64 = 2000

	fmt.Println("CustomInt32:", IntToString(custom32))
	fmt.Println("CustomInt64:", IntToString(custom64))

	// Output:
	// int8: -128
	// int16: -32768
	// int32: -2147483648
	// int64: -9223372036854775808
	// int: -42
	// positive int8: 127
	// positive int16: 32767
	// positive int32: 2147483647
	// positive int64: 9223372036854775807
	// CustomInt32: -1000
	// CustomInt64: 2000
}

// ExampleStringToUint8 demonstrates converting strings to uint8 with validation
func ExampleStringToUint8() {
	// Valid conversions
	validStrings := []string{"0", "1", "127", "255"}

	for _, s := range validStrings {
		if value, err := StringToUint8(s); err == nil {
			fmt.Printf("'%s' -> %d\n", s, value)
		}
	}

	// Invalid conversions - negative numbers
	if _, err := StringToUint8("-1"); err != nil {
		fmt.Println("Error for '-1':", err)
	}

	// Invalid conversions - out of range
	if _, err := StringToUint8("256"); err != nil {
		fmt.Println("Error for '256':", err)
	}

	// Invalid conversions - non-numeric
	if _, err := StringToUint8("abc"); err != nil {
		fmt.Println("Error for 'abc':", err)
	}

	// Output:
	// '0' -> 0
	// '1' -> 1
	// '127' -> 127
	// '255' -> 255
	// Error for '-1': out of range: value must be between 0 and 255
	// Error for '256': out of range: value must be between 0 and 255
	// Error for 'abc': strconv.Atoi: parsing "abc": invalid syntax
}
