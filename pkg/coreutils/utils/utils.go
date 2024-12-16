/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package utils

import (
	"errors"
	"math"
	"strconv"
)

func UintToString[T ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](n T) string {
	return strconv.FormatUint(uint64(n), DecimalBase)
}

func IntToString[T ~int | ~int8 | ~int16 | ~int32 | ~int64](n T) string {
	return strconv.FormatInt(int64(n), DecimalBase)
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
