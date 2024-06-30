/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package utils

import "strconv"

func UIntToString[T ~uint | ~uint8 | ~uint16 | ~uint32 | uint64](n T) string {
	return strconv.FormatUint(uint64(n), DecimalBase)
}
