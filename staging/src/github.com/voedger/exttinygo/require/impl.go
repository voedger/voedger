/*
* Copyright (c) 2023-present unTill Pro, Ltd.
*  @author Michael Saigachenko
 */

package assert

import (
	"math"
	"strconv"

	ext "github.com/voedger/exttinygo"
)

func floarToStr(f float64) string {
	return strconv.FormatFloat(f, 'E', -1, 32)
}

func EqualInt32(expected, actual int32) {
	if expected != actual {
		ext.Panic("Int32 not equal. Expected: " + strconv.FormatInt(int64(expected), 10) + " but got " + strconv.FormatInt(int64(actual), 10))
	}
}

func EqualInt64(expected, actual int64) {
	if expected != actual {
		ext.Panic("Int64 not equal. Expected: " + strconv.FormatInt(expected, 10) + " but got " + strconv.FormatInt(actual, 10))
	}
}

func EqualString(expected, actual string) {
	if expected != actual {
		ext.Panic("Strings not equal. Expected: [" + expected + "] but got [" + actual + "]")
	}
}

func EqualBytes(expected, actual []byte) {
	if len(expected) != len(actual) {
		ext.Panic("Byte array lengths not equal")
	}
	for i := 0; i < len(expected); i++ {
		if expected[i] != actual[i] {
			ext.Panic("Byte arrays not equal")
		}
	}
}

func EqualQName(expected, actual ext.QName) {
	if len(expected.Entity) != len(actual.Entity) || len(expected.Pkg) != len(actual.Pkg) {
		ext.Panic("QName not equal. Expected: " + expected.Pkg + "." + expected.Entity + " but got " + actual.Pkg + "." + actual.Entity)
	}
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func EqualBool(expected, actual bool) {
	if expected != actual {
		ext.Panic("Bool not equal. Expected: " + boolToStr(expected) + " but got " + boolToStr(actual))
	}
}

func EqualFloat32(expected float32, actual float32) {
	if math.Abs(float64(expected-actual)) > 1e-5 {
		ext.Panic("Float32 not equal. Expected: " + floarToStr(float64(expected)) + " but got " + floarToStr(float64(actual)))
	}
}

func EqualFloat64(expected float64, actual float64) {
	if math.Abs(expected-actual) > 1e-5 {
		ext.Panic("Float64 not equal. Expected: " + floarToStr(expected) + " but got " + floarToStr(actual))
	}
}
