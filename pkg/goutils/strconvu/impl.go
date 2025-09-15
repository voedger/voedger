/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package strconvu

import (
	"strconv"
)

func UintToString[T ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](n T) string {
	return strconv.FormatUint(uint64(n), decimalBase)
}

func IntToString[T ~int | ~int8 | ~int16 | ~int32 | ~int64](n T) string {
	return strconv.FormatInt(int64(n), decimalBase)
}

func ParseUint8(s string) (uint8, error) {
	uint64Val, err := strconv.ParseUint(s, decimalBase, bitSize8)
	return uint8(uint64Val), err
}

func ParseUint64(s string) (uint64, error) {
	return strconv.ParseUint(s, decimalBase, bitSize64)
}

func ParseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, decimalBase, bitSize64)
}
