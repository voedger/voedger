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
			AddField("int32", appdef.DataKind_int32, false, appdef.MinIncl(1)).
			AddField("int64", appdef.DataKind_int64, false, appdef.MinIncl(1)).
			AddField("float32", appdef.DataKind_float32, false, appdef.MinIncl(1)).
			AddField("float64", appdef.DataKind_float64, false, appdef.MinIncl(1))

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
		{"int32: min inclusive", args{"int32", int32(0)}, "int32-field «int32» data constraint «MinIncl: 1» violated"},
		{"int64: min inclusive", args{"int64", int64(0)}, "int64-field «int64» data constraint «MinIncl: 1» violated"},
		{"float32: min inclusive", args{"float32", float32(0)}, "float32-field «float32» data constraint «MinIncl: 1» violated"},
		{"float64: min inclusive", args{"float64", float64(0)}, "float64-field «float64» data constraint «MinIncl: 1» violated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkConstraints(obj.Field(tt.args.fld), tt.args.value)
			require.ErrorIs(err, ErrDataConstraintViolation)
			require.ErrorContains(err, tt.err)
		})
	}

}
