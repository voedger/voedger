/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
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
	require := require.New(t)
	cases := []struct {
		val         json.Number
		kind        appdef.DataKind
		expectedVal interface{}
	}{
		{json.Number("1"), appdef.DataKind_int8, int8(1)},   // #3434 [small integers]
		{json.Number("1"), appdef.DataKind_int16, int16(1)}, // #3434 [small integers]
		{json.Number("1"), appdef.DataKind_int32, int32(1)},
		{json.Number("1"), appdef.DataKind_int64, int64(1)},
		{json.Number("1"), appdef.DataKind_float32, float32(1)},
		{json.Number("1"), appdef.DataKind_float64, float64(1)},
		{json.Number("1"), appdef.DataKind_RecordID, istructs.RecordID(1)},
	}

	for _, c := range cases {
		actual, err := ClarifyJSONNumber(c.val, c.kind)
		require.NoError(err)
		require.Equal(c.expectedVal, actual)
	}
}

func TestClarifyJSONNumberErrors(t *testing.T) {
	require := require.New(t)
	cases := []struct {
		val  json.Number
		kind appdef.DataKind
	}{
		{val: json.Number("1.1"), kind: appdef.DataKind_int8},  // #3434 [small integers]
		{val: json.Number("1.1"), kind: appdef.DataKind_int16}, // #3434 [small integers]
		{val: json.Number("1.1"), kind: appdef.DataKind_int32},
		{val: json.Number("1.1"), kind: appdef.DataKind_int64},
		{val: json.Number("1.1"), kind: appdef.DataKind_RecordID},
		{val: json.Number(strconv.Itoa(math.MaxInt8 + 1)), kind: appdef.DataKind_int8},   // #3434 [small integers]
		{val: json.Number(strconv.Itoa(math.MinInt8 - 1)), kind: appdef.DataKind_int8},   // #3434 [small integers]
		{val: json.Number(strconv.Itoa(math.MaxInt16 + 1)), kind: appdef.DataKind_int16}, // #3434 [small integers]
		{val: json.Number(strconv.Itoa(math.MinInt16 - 1)), kind: appdef.DataKind_int16}, // #3434 [small integers]
		{val: json.Number(strconv.Itoa(math.MaxInt32 + 1)), kind: appdef.DataKind_int32},
		{val: json.Number(strconv.Itoa(math.MinInt32 - 1)), kind: appdef.DataKind_int32},
		{val: json.Number(fmt.Sprint(math.MaxInt64 + (float64(1)))), kind: appdef.DataKind_int64},
		{val: json.Number(fmt.Sprint(math.MinInt64 - (float64(1)))), kind: appdef.DataKind_int64},
		{val: json.Number(fmt.Sprint(math.MaxFloat64)), kind: appdef.DataKind_float32},
		{val: json.Number(fmt.Sprint(-math.MaxFloat64)), kind: appdef.DataKind_float32},
		{val: json.Number("a"), kind: appdef.DataKind_float32},
		{val: json.Number("a"), kind: appdef.DataKind_float64},
		{val: json.Number("a"), kind: appdef.DataKind_int32},
		{val: json.Number("a"), kind: appdef.DataKind_int64},
		{val: json.Number("a"), kind: appdef.DataKind_RecordID},
		{val: json.Number(TooBigNumberStr), kind: appdef.DataKind_float64},
		{val: json.Number("-" + TooBigNumberStr), kind: appdef.DataKind_float64},
		{val: json.Number("-1"), kind: appdef.DataKind_RecordID},
	}

	for _, c := range cases {
		actual, err := ClarifyJSONNumber(c.val, c.kind)
		require.Error(err)
		require.Nil(actual)
	}

	require.Panics(func() {
		ClarifyJSONNumber(json.Number("1"), appdef.DataKind_string)
	})
}
