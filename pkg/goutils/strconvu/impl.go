/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package strconvu

import (
	"errors"
	"math"
	"strconv"
)

func UintToString[T ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](n T) string {
	return strconv.FormatUint(uint64(n), decimalBase)
}

func IntToString[T ~int | ~int8 | ~int16 | ~int32 | ~int64](n T) string {
	return strconv.FormatInt(int64(n), decimalBase)
}

func StringToUint8(s string) (uint8, error) {
	value, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if value < 0 || value > math.MaxUint8 {
		return 0, errors.New("out of range: value must be between 0 and 255")
	}
	return uint8(value), nil
}

// ParseUint64 parses a string into a uint64 value treating it as a regular decimal number
func ParseUint64(s string) (uint64, error) {
	return strconv.ParseUint(s, decimalBase, bitSize64)
}

// ParseInt64 parses a string into an int64 value treating it as a regular decimal number
func ParseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, decimalBase, bitSize64)
}
