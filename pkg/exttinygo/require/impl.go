/*
* Copyright (c) 2023-present unTill Pro, Ltd.
*  @author Michael Saigachenko
 */

package assert

import (
	"math"
	"strconv"

	"github.com/voedger/voedger/pkg/coreutils/utils"
	ext "github.com/voedger/voedger/pkg/exttinygo"
)

func floarToStr(f float64) string {
	return strconv.FormatFloat(f, 'E', -1, int32bitsLength)
}

func EqualInt32(expected, actual int32) {
	if expected != actual {
		panic("Int32 not equal. Expected: " + utils.IntToString(expected) + butGotPhrase + utils.IntToString(actual))
	}
}

func EqualInt64(expected, actual int64) {
	if expected != actual {
		panic("Int64 not equal. Expected: " + utils.IntToString(expected) + butGotPhrase + utils.IntToString(actual))
	}
}

func EqualString(expected, actual string) {
	if expected != actual {
		panic("Strings not equal. Expected: [" + expected + "] but got [" + actual + "]")
	}
}

func EqualBytes(expected, actual []byte) {
	if len(expected) != len(actual) {
		panic("Byte array lengths not equal")
	}
	for i := 0; i < len(expected); i++ {
		if expected[i] != actual[i] {
			panic("Byte arrays not equal")
		}
	}
}

func EqualQName(expected, actual ext.QName) {
	if len(expected.Entity) != len(actual.Entity) || len(expected.FullPkgName) != len(actual.FullPkgName) {
		panic("QName not equal. Expected: " + expected.FullPkgName + "." + expected.Entity + butGotPhrase + actual.FullPkgName + "." + actual.Entity)
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
		panic("Bool not equal. Expected: " + boolToStr(expected) + butGotPhrase + boolToStr(actual))
	}
}

func EqualFloat32(expected float32, actual float32) {
	if math.Abs(float64(expected-actual)) > delta {
		panic("Float32 not equal. Expected: " + floarToStr(float64(expected)) + butGotPhrase + floarToStr(float64(actual)))
	}
}

func EqualFloat64(expected float64, actual float64) {
	if math.Abs(expected-actual) > delta {
		panic("Float64 not equal. Expected: " + floarToStr(expected) + butGotPhrase + floarToStr(actual))
	}
}
