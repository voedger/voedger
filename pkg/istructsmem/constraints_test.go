/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/constraints"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_checkConstraints(t *testing.T) {
	require := require.New(t)

	obj := func() appdef.IObject {
		name := appdef.NewQName("test", "obj")
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		obj := wsb.AddObject(name)
		obj.
			AddField("str", appdef.DataKind_string, false).
			AddField("bytes", appdef.DataKind_bytes, false).
			// char fields to test constraints: exactly four digits
			AddField("str4", appdef.DataKind_string, false,
				constraints.MinLen(4),
				constraints.MaxLen(4),
				constraints.Pattern(`^\d+$`),
				constraints.Enum(`1111`, `2222`, `3333`, `4444`)).
			AddField("bytes4", appdef.DataKind_bytes, false,
				constraints.MinLen(4),
				constraints.MaxLen(4),
				constraints.Pattern(`^\d+$`)).
			// numeric fields to test inclusive constraints: closed range [1, 8]
			AddField("int8_i", appdef.DataKind_int8, false, // #3434 [small integers]
										constraints.MinIncl(1),
										constraints.MaxIncl(8),
										constraints.Enum(int8(2), 4, 6, 8)).
			AddField("int16_i", appdef.DataKind_int16, false, // #3434 [small integers]
				constraints.MinIncl(1),
				constraints.MaxIncl(8),
				constraints.Enum(int16(2), 4, 6, 8)).
			AddField("int32_i", appdef.DataKind_int32, false,
				constraints.MinIncl(1),
				constraints.MaxIncl(8),
				constraints.Enum(int32(2), 4, 6, 8)).
			AddField("int64_i", appdef.DataKind_int64, false,
				constraints.MinIncl(1),
				constraints.MaxIncl(8),
				constraints.Enum(int64(7), 5, 3, 1)).
			AddField("float32_i", appdef.DataKind_float32, false,
				constraints.MinIncl(1),
				constraints.MaxIncl(8),
				constraints.Enum(float32(7.77), 5.55, 3.33, 1.11, 1.11, 1.11)).
			AddField("float64_i", appdef.DataKind_float64, false,
				constraints.MinIncl(1),
				constraints.MaxIncl(8),
				constraints.Enum(math.Pi, 2*math.Pi, 1)).
			// numeric fields to test exclusive constraints: open range (0, 9)
			AddField("int8_e", appdef.DataKind_int8, false, // #3434 [small integers]
										constraints.MinExcl(0),
										constraints.MaxExcl(9)).
			AddField("int16_e", appdef.DataKind_int16, false, // #3434 [small integers]
				constraints.MinExcl(0),
				constraints.MaxExcl(9)).
			AddField("int32_e", appdef.DataKind_int32, false,
				constraints.MinExcl(0),
				constraints.MaxExcl(9)).
			AddField("int64_e", appdef.DataKind_int64, false,
				constraints.MinExcl(0),
				constraints.MaxExcl(9)).
			AddField("float32_e", appdef.DataKind_float32, false,
				constraints.MinExcl(0),
				constraints.MaxExcl(9)).
			AddField("float64_e", appdef.DataKind_float64, false,
				constraints.MinExcl(0),
				constraints.MaxExcl(9))

		app, err := adb.Build()
		require.NoError(err)
		require.NotNil(app)

		return appdef.Object(app.Type, name)
	}()

	type args struct {
		fld   string
		value interface{}
	}
	tests := []struct {
		name string
		args args
		err  string
	}{
		{"string: default max len", args{"str", strings.Repeat("7", int(appdef.DefaultFieldMaxLength)+1)}, "default MaxLen:"},
		{"string: min len", args{"str4", "123"}, "MinLen: 4"},
		{"string: max len", args{"str4", "12345"}, "MaxLen: 4"},
		{"string: pattern", args{"str4", "abcd"}, "Pattern: `^\\d+$`"},
		{"string: enum", args{"str4", "0000"}, "Enum: [1111 2222 3333 4444]"},
		{"string: ok", args{"str4", "2222"}, ""},
		//-
		{"[]byte: default max len", args{"bytes", bytes.Repeat([]byte("7"), int(appdef.DefaultFieldMaxLength)+1)}, "default MaxLen:"},
		{"[]byte: min len", args{"bytes4", []byte("123")}, "MinLen: 4"},
		{"[]byte: max len", args{"bytes4", []byte("12345")}, "MaxLen: 4"},
		{"[]byte: pattern", args{"bytes4", []byte("abcd")}, "Pattern: `^\\d+$`"},
		{"[]byte: ok", args{"bytes4", []byte("1234")}, ""},
		//- // #3434 [small integers]
		{"int8_i: min inclusive", args{"int8_i", int8(0)}, "MinIncl: 1"},
		{"int8_i: max inclusive", args{"int8_i", int8(9)}, "MaxIncl: 8"},
		{"int8_e: min exclusive", args{"int8_e", int8(0)}, "MinExcl: 0"},
		{"int8_e: max exclusive", args{"int8_e", int8(9)}, "MaxExcl: 9"},
		{"int8_i: enum", args{"int8_i", int8(5)}, "Enum: [2 4 6 8]"},
		{"int8_i: ok", args{"int8_i", int8(4)}, ""},
		//- // #3434 [small integers]
		{"int16_i: min inclusive", args{"int16_i", int16(0)}, "MinIncl: 1"},
		{"int16_i: max inclusive", args{"int16_i", int16(9)}, "MaxIncl: 8"},
		{"int16_e: min exclusive", args{"int16_e", int16(0)}, "MinExcl: 0"},
		{"int16_e: max exclusive", args{"int16_e", int16(9)}, "MaxExcl: 9"},
		{"int16_i: enum", args{"int16_i", int16(5)}, "Enum: [2 4 6 8]"},
		{"int16_i: ok", args{"int16_i", int16(4)}, ""},
		//-
		{"int32_i: min inclusive", args{"int32_i", int32(0)}, "MinIncl: 1"},
		{"int32_i: max inclusive", args{"int32_i", int32(9)}, "MaxIncl: 8"},
		{"int32_e: min exclusive", args{"int32_e", int32(0)}, "MinExcl: 0"},
		{"int32_e: max exclusive", args{"int32_e", int32(9)}, "MaxExcl: 9"},
		{"int32_i: enum", args{"int32_i", int32(5)}, "Enum: [2 4 6 8]"},
		{"int32_i: ok", args{"int32_i", int32(4)}, ""},
		//-
		{"int64_i: min inclusive", args{"int64_i", int64(0)}, "MinIncl: 1"},
		{"int64_i: max inclusive", args{"int64_i", int64(9)}, "MaxIncl: 8"},
		{"int64_e: min exclusive", args{"int64_e", int64(0)}, "MinExcl: 0"},
		{"int64_e: max exclusive", args{"int64_e", int64(9)}, "MaxExcl: 9"},
		{"int64_i: enum", args{"int64_i", int64(4)}, "Enum: [1 3 5 7]"},
		{"int64_i: ok", args{"int64_i", int64(7)}, ""},
		//-
		{"float32_i: min inclusive", args{"float32_i", float32(0)}, "MinIncl: 1"},
		{"float32_i: max inclusive", args{"float32_i", float32(9)}, "MaxIncl: 8"},
		{"float32_e: min exclusive", args{"float32_e", float32(0)}, "MinExcl: 0"},
		{"float32_e: max exclusive", args{"float32_e", float32(9)}, "MaxExcl: 9"},
		{"float32_i: enum", args{"float32_i", float32(5.5)}, "Enum: [1.11 3.33 5.55 7.77]"},
		{"float32_i: ok", args{"float32_i", float32(5.55)}, ""},
		//-
		{"float64_i: min inclusive", args{"float64_i", float64(0)}, "MinIncl: 1"},
		{"float64_i: max inclusive", args{"float64_i", float64(9)}, "MaxIncl: 8"},
		{"float64_e: min exclusive", args{"float64_e", float64(0)}, "MinExcl: 0"},
		{"float64_e: max exclusive", args{"float64_e", float64(9)}, "MaxExcl: 9"},
		{"float64_i: enum", args{"float64_i", math.E}, "Enum: [1 3.14159265358"},
		{"float64_i: ok", args{"float64_i", math.Pi}, ""},
		//-
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fld := obj.Field(tt.args.fld)
			err := checkConstraints(fld, tt.args.value)
			if len(tt.err) > 0 {
				require.Error(err, require.Is(ErrDataConstraintViolationError),
					require.HasAll(fld, tt.err))
			} else {
				require.NoError(err)
			}
		})
	}

}
