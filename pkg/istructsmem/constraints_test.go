/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
)

func Test_checkConstraints(t *testing.T) {
	require := require.New(t)

	obj := func() appdef.IObject {
		name := appdef.NewQName("test", "obj")
		adb := appdef.New()

		obj := adb.AddObject(name)
		obj.
			AddField("str", appdef.DataKind_string, false).
			AddField("str4", appdef.DataKind_string, false, appdef.MinLen(4), appdef.MaxLen(4), appdef.Pattern(`^\d+$`)).
			//—
			AddField("bytes", appdef.DataKind_bytes, false).
			AddField("bytes4", appdef.DataKind_bytes, false, appdef.MinLen(4), appdef.MaxLen(4), appdef.Pattern(`^\d+$`)).
			//—
			AddField("int32_i", appdef.DataKind_int32, false, appdef.MinIncl(1)).
			AddField("int64_i", appdef.DataKind_int64, false, appdef.MinIncl(1)).
			AddField("float32_i", appdef.DataKind_float32, false, appdef.MinIncl(1)).
			AddField("float64_i", appdef.DataKind_float64, false, appdef.MinIncl(1)).
			//—
			AddField("int32_e", appdef.DataKind_int32, false, appdef.MinExcl(0)).
			AddField("int64_e", appdef.DataKind_int64, false, appdef.MinExcl(0)).
			AddField("float32_e", appdef.DataKind_float32, false, appdef.MinExcl(0)).
			AddField("float64_e", appdef.DataKind_float64, false, appdef.MinExcl(0))

		app, err := adb.Build()
		require.NoError(err)
		require.NotNil(app)

		return app.Object(name)
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
		{"string: default max len", args{"str", strings.Repeat("7", int(appdef.DefaultFieldMaxLength)+1)}, "string-field «str» data constraint «default MaxLen:"},
		{"string: min len", args{"str4", "123"}, "string-field «str4» data constraint «MinLen: 4» violated"},
		{"string: max len", args{"str4", "12345"}, "string-field «str4» data constraint «MaxLen: 4» violated"},
		{"string: pattern", args{"str4", "abcd"}, "string-field «str4» data constraint «Pattern: `^\\d+$`» violated"},
		//-
		{"[]byte: default max len", args{"bytes", bytes.Repeat([]byte("7"), int(appdef.DefaultFieldMaxLength)+1)}, "bytes-field «bytes» data constraint «default MaxLen:"},
		{"[]byte: min len", args{"bytes4", []byte("123")}, "bytes-field «bytes4» data constraint «MinLen: 4» violated"},
		{"[]byte: max len", args{"bytes4", []byte("12345")}, "bytes-field «bytes4» data constraint «MaxLen: 4» violated"},
		{"[]byte: pattern", args{"bytes4", []byte("abcd")}, "bytes-field «bytes4» data constraint «Pattern: `^\\d+$`» violated"},
		//-
		{"int32_i: min inclusive", args{"int32_i", int32(0)}, "int32-field «int32_i» data constraint «MinIncl: 1» violated"},
		{"int64_i: min inclusive", args{"int64_i", int64(0)}, "int64-field «int64_i» data constraint «MinIncl: 1» violated"},
		{"float32_i: min inclusive", args{"float32_i", float32(0)}, "float32-field «float32_i» data constraint «MinIncl: 1» violated"},
		{"float64_i: min inclusive", args{"float64_i", float64(0)}, "float64-field «float64_i» data constraint «MinIncl: 1» violated"},
		//-
		{"int32_e: min exclusive", args{"int32_e", int32(0)}, "int32-field «int32_e» data constraint «MinExcl: 0» violated"},
		{"int64_e: min exclusive", args{"int64_e", int64(0)}, "int64-field «int64_e» data constraint «MinExcl: 0» violated"},
		{"float32_e: min exclusive", args{"float32_e", float32(0)}, "float32-field «float32_e» data constraint «MinExcl: 0» violated"},
		{"float64_e: min exclusive", args{"float64_e", float64(0)}, "float64-field «float64_e» data constraint «MinExcl: 0» violated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkConstraints(obj.Field(tt.args.fld), tt.args.value)
			require.ErrorIs(err, ErrDataConstraintViolation)
			require.ErrorContains(err, tt.err)
		})
	}

}
