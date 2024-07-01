/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package utils

import "strconv"

func UintToString[T ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](n T) string {
	return strconv.FormatUint(uint64(n), DecimalBase)
}

func IntToString[T ~int | ~int8 | ~int16 | ~int32 | ~int64](n T) string {
	return strconv.FormatInt(int64(n), DecimalBase)
}
