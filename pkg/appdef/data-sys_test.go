/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_SysDataName(t *testing.T) {
	type args struct {
		k DataKind
	}
	tests := []struct {
		name string
		args args
		want QName
	}{
		{"null", args{k: DataKind_null}, NullQName},
		{"int32", args{k: DataKind_int32}, MustParseQName("sys.int32")},
		{"string", args{k: DataKind_string}, MustParseQName("sys.string")},
		{"out of bounds", args{k: DataKind_FakeLast}, NullQName},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SysDataName(tt.args.k); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sysDataTypeName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_appDef_makeSysDataTypes(t *testing.T) {
	require := require.New(t)

	app, err := New().Build()
	require.NoError(err)

	t.Run("must be ok to get system data types", func(t *testing.T) {
		for k := DataKind_null + 1; k < DataKind_FakeLast; k++ {
			d := app.SysData(k)
			require.NotNil(d)
			require.Equal(SysDataName(k), d.QName())
			require.Equal(TypeKind_Data, d.Kind())
			require.Equal(k, d.DataKind())
			require.Nil(d.Ancestor())
			cnt := 0
			d.Constraints(func(c IConstraint) { cnt++ })
			require.Equal(0, cnt, "system data type %v should have no constraints", d)
		}
	})
}
