/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"encoding/json"
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestJSONUnmarshal(t *testing.T) {
	require := require.New(t)
	j := `{"int":1,"float":1.1}`
	res := map[string]interface{}{}
	require.NoError(JSONUnmarshal([]byte(j), &res))
	require.IsType(json.Number(""), res["int"])
	require.IsType(json.Number(""), res["float"])
}

func TestClarifyJSONNumber(t *testing.T) {
	type numberCase struct {
		name     string
		input    string
		expected any
		err      error
	}
	cases := []struct {
		name    string
		convert func(json.Number) (any, error)
		numbers []numberCase
	}{
		{
			name:    "int8",
			convert: clarifyJSONNumberForTest[int8],
			numbers: []numberCase{
				{"basic", "1", int8(1), nil},
				{"zero", "0", int8(0), nil},
				{"negative zero", "-0", int8(0), nil},
				{"minimum", "-128", int8(math.MinInt8), nil},
				{"maximum", "127", int8(math.MaxInt8), nil},
				{"above maximum", "128", nil, ErrNumberOverflow},
				{"below minimum", "-129", nil, ErrNumberOverflow},
				{"fraction", "1.1", nil, strconv.ErrSyntax},
				{"decimal integer", "1.0", nil, strconv.ErrSyntax},
				{"exponent", "1e0", nil, strconv.ErrSyntax},
				{"parser overflow", "9223372036854775808", nil, strconv.ErrRange},
			},
		},
		{
			name:    "int16",
			convert: clarifyJSONNumberForTest[int16],
			numbers: []numberCase{
				{"basic", "1", int16(1), nil},
				{"zero", "0", int16(0), nil},
				{"negative zero", "-0", int16(0), nil},
				{"minimum", "-32768", int16(math.MinInt16), nil},
				{"maximum", "32767", int16(math.MaxInt16), nil},
				{"above maximum", "32768", nil, ErrNumberOverflow},
				{"below minimum", "-32769", nil, ErrNumberOverflow},
				{"fraction", "1.1", nil, strconv.ErrSyntax},
				{"decimal integer", "1.0", nil, strconv.ErrSyntax},
				{"exponent", "1e0", nil, strconv.ErrSyntax},
				{"parser overflow", "9223372036854775808", nil, strconv.ErrRange},
			},
		},
		{
			name:    "int32",
			convert: clarifyJSONNumberForTest[int32],
			numbers: []numberCase{
				{"basic", "1", int32(1), nil},
				{"zero", "0", int32(0), nil},
				{"negative zero", "-0", int32(0), nil},
				{"minimum", "-2147483648", int32(math.MinInt32), nil},
				{"maximum", "2147483647", int32(math.MaxInt32), nil},
				{"above maximum", "2147483648", nil, ErrNumberOverflow},
				{"below minimum", "-2147483649", nil, ErrNumberOverflow},
				{"fraction", "1.1", nil, strconv.ErrSyntax},
				{"decimal integer", "1.0", nil, strconv.ErrSyntax},
				{"exponent", "1e0", nil, strconv.ErrSyntax},
				{"parser overflow", "9223372036854775808", nil, strconv.ErrRange},
			},
		},
		{
			name:    "int64",
			convert: clarifyJSONNumberForTest[int64],
			numbers: []numberCase{
				{"basic", "1", int64(1), nil},
				{"zero", "0", int64(0), nil},
				{"negative zero", "-0", int64(0), nil},
				{"minimum", "-9223372036854775808", int64(math.MinInt64), nil},
				{"maximum", "9223372036854775807", int64(math.MaxInt64), nil},
				{"above maximum", "9223372036854775808", nil, strconv.ErrRange},
				{"below minimum", "-9223372036854775809", nil, strconv.ErrRange},
				{"fraction", "1.1", nil, strconv.ErrSyntax},
				{"decimal integer", "1.0", nil, strconv.ErrSyntax},
				{"exponent", "1e0", nil, strconv.ErrSyntax},
				{"positive scientific boundary", "9.223372036854776e+18", nil, strconv.ErrSyntax},
				{"negative scientific boundary", "-9.223372036854776e+18", nil, strconv.ErrSyntax},
			},
		},
		{
			name:    "float32",
			convert: clarifyJSONNumberForTest[float32],
			numbers: []numberCase{
				{"basic", "1", float32(1), nil},
				{"zero", "0", float32(0), nil},
				{"negative zero", "-0", float32(math.Copysign(0, -1)), nil},
				{"positive fraction", "1.25", float32(1.25), nil},
				{"negative fraction", "-1.25", float32(-1.25), nil},
				{"exponent", "1.25e2", float32(125), nil},
				{"maximum", "3.4028234663852886e+38", float32(math.MaxFloat32), nil},
				{"minimum", "-3.4028234663852886e+38", float32(-math.MaxFloat32), nil},
				{"above maximum", "3.402823466385289e+38", nil, ErrNumberOverflow},
				{"below minimum", "-3.402823466385289e+38", nil, ErrNumberOverflow},
				{"positive float64 maximum", "1.7976931348623157e+308", nil, ErrNumberOverflow},
				{"negative float64 maximum", "-1.7976931348623157e+308", nil, ErrNumberOverflow},
				{"smallest positive", "1.401298464324817e-45", float32(math.SmallestNonzeroFloat32), nil},
				{"smallest negative", "-1.401298464324817e-45", float32(-math.SmallestNonzeroFloat32), nil},
				{"underflow", "1e-1000", float32(0), nil},
				{"negative underflow", "-1e-1000", float32(math.Copysign(0, -1)), nil},
				{"conversion underflow", "1e-46", float32(0), nil},
				{"negative conversion underflow", "-1e-46", float32(math.Copysign(0, -1)), nil},
				{"integer rounding", "16777217", float32(16777216), nil},
				{"positive parser overflow", TooBigNumberStr, nil, strconv.ErrRange},
				{"negative parser overflow", "-" + TooBigNumberStr, nil, strconv.ErrRange},
			},
		},
		{
			name:    "float64",
			convert: clarifyJSONNumberForTest[float64],
			numbers: []numberCase{
				{"basic", "1", float64(1), nil},
				{"zero", "0", float64(0), nil},
				{"negative zero", "-0", float64(math.Copysign(0, -1)), nil},
				{"positive fraction", "1.25", float64(1.25), nil},
				{"negative fraction", "-1.25", float64(-1.25), nil},
				{"exponent", "1.25e2", float64(125), nil},
				{"maximum", "1.7976931348623157e+308", float64(math.MaxFloat64), nil},
				{"minimum", "-1.7976931348623157e+308", float64(-math.MaxFloat64), nil},
				{"above maximum", "1.7976931348623159e+308", nil, strconv.ErrRange},
				{"below minimum", "-1.7976931348623159e+308", nil, strconv.ErrRange},
				{"smallest positive", "5e-324", float64(math.SmallestNonzeroFloat64), nil},
				{"smallest negative", "-5e-324", float64(-math.SmallestNonzeroFloat64), nil},
				{"underflow", "1e-1000", float64(0), nil},
				{"negative underflow", "-1e-1000", float64(math.Copysign(0, -1)), nil},
				{"integer rounding", "9007199254740993", float64(9007199254740992), nil},
				{"positive parser overflow", TooBigNumberStr, nil, strconv.ErrRange},
				{"negative parser overflow", "-" + TooBigNumberStr, nil, strconv.ErrRange},
			},
		},
		{
			name:    "RecordID",
			convert: clarifyJSONNumberForTest[istructs.RecordID],
			numbers: []numberCase{
				{"basic", "1", istructs.RecordID(1), nil},
				{"zero", "0", istructs.RecordID(0), nil},
				{"negative zero", "-0", istructs.RecordID(0), nil},
				{"maximum", "9223372036854775807", istructs.RecordID(math.MaxInt64), nil},
				{"above maximum", "9223372036854775808", nil, strconv.ErrRange},
				{"below minimum", "-1", nil, ErrNumberOverflow},
				{"fraction", "1.1", nil, strconv.ErrSyntax},
				{"decimal integer", "1.0", nil, strconv.ErrSyntax},
				{"exponent", "1e0", nil, strconv.ErrSyntax},
				{"positive scientific boundary", "9.223372036854776e+18", nil, strconv.ErrSyntax},
				{"negative scientific boundary", "-9.223372036854776e+18", nil, strconv.ErrSyntax},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, input := range []string{"", "a", " 1", "1 ", "1e", "--1"} {
				tc.numbers = append(tc.numbers, numberCase{"invalid/" + input, input, nil, strconv.ErrSyntax})
			}
			for _, number := range tc.numbers {
				t.Run(number.name, func(t *testing.T) {
					require := require.New(t)
					actual, err := tc.convert(json.Number(number.input))
					if number.err != nil {
						require.ErrorIs(err, number.err)
						require.Contains(err.Error(), number.input+" to "+tc.name)
						require.Zero(actual)
						return
					}
					require.NoError(err)
					require.Equal(number.expected, actual)
					switch expected := number.expected.(type) {
					case float32:
						require.Equal(math.Float32bits(expected), math.Float32bits(actual.(float32)))
					case float64:
						require.Equal(math.Float64bits(expected), math.Float64bits(actual.(float64)))
					}
				})
			}
		})
	}
}

func clarifyJSONNumberForTest[T int8 | int16 | int32 | int64 | float32 | float64 | istructs.RecordID](value json.Number) (any, error) {
	return ClarifyJSONNumber[T](value)
}

func TestJSONUnmarshalDisallowUnknownFields(t *testing.T) {
	require := require.New(t)
	type payload struct {
		A int    `json:"a"`
		B string `json:"b"`
	}

	t.Run("basic", func(t *testing.T) {
		var p payload
		require.NoError(JSONUnmarshalDisallowUnknownFields([]byte(`{"a":1,"b":"x"}`), &p))
		require.Equal(1, p.A)
		require.Equal("x", p.B)
	})

	t.Run("error on unknown field", func(t *testing.T) {
		var p payload
		err := JSONUnmarshalDisallowUnknownFields([]byte(`{"a":1,"b":"x","c":2}`), &p)
		require.Error(err)
		require.Contains(err.Error(), `unknown field "c"`)
	})

	t.Run("error on invalid JSON", func(t *testing.T) {
		var p payload
		require.Error(JSONUnmarshalDisallowUnknownFields([]byte(`{"a":1,"b":"x"`), &p))
	})
}

func TestClarifyJSONWSID(t *testing.T) {
	testCases := []struct {
		name    string
		input   string
		expects istructs.WSID
		wantErr bool
	}{
		{"valid", "123", 123, false},
		{"negative", "-1", 0, true},
		{"above_max", strconv.FormatUint(istructs.MaxAllowedWSID+1, 10), 0, true},
		{"invalid", "notanumber", 0, true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wsid, err := ClarifyJSONWSID(json.Number(tc.input))
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expects, wsid)
			}
		})
	}
}
