/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package strconvu

import (
	"strconv"
)

// UintToString converts an unsigned integer to its decimal string representation.
func UintToString[T ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](n T) string {
	return strconv.FormatUint(uint64(n), decimalBase)
}

// IntToString converts a signed integer to its decimal string representation.
func IntToString[T ~int | ~int8 | ~int16 | ~int32 | ~int64](n T) string {
	return strconv.FormatInt(int64(n), decimalBase)
}

// ParseUint8 parses a string into a uint8 value treating it as a regular decimal number
func ParseUint8(s string) (uint8, error) {
	value, err := strconv.ParseUint(s, decimalBase, bitSize8)
	if err != nil {
		return 0, err
	}
	return uint8(value), err
}

// ParseUint64 parses a string into a uint64 value treating it as a regular decimal number
func ParseUint64(s string) (uint64, error) {
	return strconv.ParseUint(s, decimalBase, bitSize64)
}

// ParseInt32 parses a string into an int32 value treating it as a regular decimal number
func ParseInt32(s string) (int32, error) {
	value, err := strconv.ParseInt(s, decimalBase, bitSize32)
	if err != nil {
		return 0, err
	}
	return int32(value), err
}

// ParseInt64 parses a string into an int64 value treating it as a regular decimal number
func ParseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, decimalBase, bitSize64)
}
