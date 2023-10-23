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

func Test_sysDataTypeName(t *testing.T) {
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
			if got := sysDataTypeName(tt.args.k); !reflect.DeepEqual(got, tt.want) {
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
			require.Equal(sysDataTypeName(k), d.QName())
			require.Equal(TypeKind_Data, d.Kind())
			require.Equal(k, d.DataKind())
			require.Nil(d.Ancestor())
			switch k {
			case DataKind_string, DataKind_bytes:
				cnt := 0
				d.Constraints(func(c IConstraint) {
					cnt++
					switch cnt {
					case 1:
						require.Equal(ConstraintKind_MaxLen, c.Kind())
						require.EqualValues(DefaultFieldMaxLength, c.Value())
					default:
						require.Fail("unexpected constraint", "constraint: %v", c)
					}
				})
				require.Equal(1, cnt)
			}
		}
	})
}
