/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"math"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AppDef_AddData(t *testing.T) {
	require := require.New(t)

	var app IAppDef

	intName := NewQName("test", "int")
	strName := NewQName("test", "string")
	tokenName := NewQName("test", "token")

	t.Run("must be ok to add data types", func(t *testing.T) {
		appDef := New()

		_ = appDef.AddData(intName, DataKind_int64, NullQName)
		_ = appDef.AddData(strName, DataKind_string, NullQName)
		token := appDef.AddData(tokenName, DataKind_string, strName)
		token.AddConstraints(MinLen(1), MaxLen(100), Pattern(`^\w+$`, "only word characters allowed"))

		t.Run("must be ok to build", func(t *testing.T) {
			a, err := appDef.Build()
			require.NoError(err)
			require.NotNil(a)

			app = a
		})
	})

	require.NotNil(app)

	t.Run("must be ok to find builded data type", func(t *testing.T) {
		i := app.Data(intName)
		require.Equal(TypeKind_Data, i.Kind())
		require.Equal(intName, i.QName())
		require.Equal(DataKind_int64, i.DataKind())
		require.False(i.IsSystem())
		require.Equal(app.SysData(DataKind_int64), i.Ancestor())

		s := app.Data(strName)
		require.Equal(TypeKind_Data, s.Kind())
		require.Equal(strName, s.QName())
		require.Equal(DataKind_string, s.DataKind())
		require.Equal(app.SysData(DataKind_string), s.Ancestor())

		tk := app.Data(tokenName)
		require.Equal(TypeKind_Data, tk.Kind())
		require.Equal(tokenName, tk.QName())
		require.Equal(DataKind_string, tk.DataKind())
		require.Equal(s, tk.Ancestor())
		cnt := 0
		tk.Constraints(func(c IConstraint) {
			cnt++
			switch cnt {
			case 1:
				require.Equal(ConstraintKind_MinLen, c.Kind())
				require.EqualValues(1, c.Value())
			case 2:
				require.Equal(ConstraintKind_MaxLen, c.Kind())
				require.EqualValues(100, c.Value())
			case 3:
				require.Equal(ConstraintKind_Pattern, c.Kind())
				require.EqualValues(`^\w+$`, c.Value().(*regexp.Regexp).String())
				require.Equal("only word characters allowed", c.Comment())
			default:
				require.Failf("unexpected constraint", "constraint: %v", c)
			}
		})
		require.Equal(3, cnt)
	})

	t.Run("must be ok to enum data types", func(t *testing.T) {
		cnt := 0
		app.DataTypes(false, func(d IData) {
			cnt++
			require.Equal(TypeKind_Data, d.Kind())
			switch cnt {
			case 1:
				require.Equal(intName, d.QName())
			case 2:
				require.Equal(strName, d.QName())
			case 3:
				require.Equal(tokenName, d.QName())
			default:
				require.Failf("unexpected data type", "data type: %v", d)
			}
		})
		require.Equal(3, cnt)
	})

	t.Run("check nil returns", func(t *testing.T) {
		unknown := NewQName("test", "unknown")
		require.Nil(app.Data(unknown))
	})

	t.Run("panic if name is empty", func(t *testing.T) {
		apb := New()
		require.Panics(func() {
			apb.AddData(NullQName, DataKind_int64, NullQName)
		})
	})

	t.Run("panic if name is invalid", func(t *testing.T) {
		apb := New()
		require.Panics(func() {
			apb.AddData(NewQName("naked", "🔫"), DataKind_QName, NullQName)
		})
	})

	t.Run("panic if type with name already exists", func(t *testing.T) {
		apb := New()
		apb.AddObject(intName)
		require.Panics(func() {
			apb.AddData(intName, DataKind_int64, NullQName)
		})
	})

	t.Run("panic if unknown system ancestor", func(t *testing.T) {
		apb := New()
		require.Panics(func() {
			apb.AddData(intName, DataKind_null, NullQName)
		})
	})

	t.Run("panic if ancestor is not found", func(t *testing.T) {
		apb := New()
		require.Panics(func() {
			apb.AddData(intName, DataKind_int64,
				NewQName("test", "unknown"), // <- error here
			)
		})
	})

	t.Run("panic if ancestor is not data type", func(t *testing.T) {
		objName := NewQName("test", "object")
		apb := New()
		_ = apb.AddObject(objName)
		require.Panics(func() {
			apb.AddData(intName, DataKind_int64,
				objName, // <- error here
			)
		})
	})

	t.Run("panic if ancestor has different kind", func(t *testing.T) {
		apb := New()
		_ = apb.AddData(strName, DataKind_string, NullQName)
		require.Panics(func() {
			apb.AddData(intName, DataKind_int64, strName)
		})
	})

	t.Run("panic if incompatible constraints", func(t *testing.T) {
		apb := New()
		require.Panics(func() { _ = apb.AddData(strName, DataKind_string, NullQName, MinIncl(1)) })
		require.Panics(func() { _ = apb.AddData(intName, DataKind_float64, NullQName, MaxLen(100)) })
		require.Panics(func() { _ = apb.AddData(NewQName("test", "raw"), DataKind_raw, NullQName, Pattern(`^\w+$`)) })
	})
}

func Test_data_AddConstraint(t *testing.T) {
	type args struct {
		da DataKind
		ck ConstraintKind
		cv any
	}
	tests := []struct {
		name      string
		args      args
		wantPanic bool
	}{
		//- MaxLen
		{"string: max length constraint must be ok",
			args{DataKind_string, ConstraintKind_MaxLen, uint16(100)}, false},
		{"string: max length constraint must fail if exceeds 1024",
			args{DataKind_string, ConstraintKind_MaxLen, MaxFieldLength + 1}, true},
		{"bytes: max length constraint must be ok",
			args{DataKind_bytes, ConstraintKind_MaxLen, uint16(1024)}, false},
		{"bytes: max length constraint must fail if exceeds 1024",
			args{DataKind_bytes, ConstraintKind_MaxLen, MaxFieldLength + 1}, true},
		{"raw: max length constraint must be ok",
			args{DataKind_raw, ConstraintKind_MaxLen, uint16(100)}, false},
		{"raw: max length constraint must be ok up to 64K",
			args{DataKind_raw, ConstraintKind_MaxLen, uint16(math.MaxUint16)}, false},
		//- Enum
		{"int32: enum constraint must be ok",
			args{DataKind_int32, ConstraintKind_Enum, []int32{1, 2, 3}}, false},
		{"int32: enum constraint must fail if wrong enum type",
			args{DataKind_int32, ConstraintKind_Enum, []int64{1, 2, 3}}, true},
		{"int64: enum constraint must be ok",
			args{DataKind_int64, ConstraintKind_Enum, []int64{1, 2, 3}}, false},
		{"int64: enum constraint must fail if wrong enum type",
			args{DataKind_int64, ConstraintKind_Enum, []string{"1", "2", "3"}}, true},
		{"float32: enum constraint must be ok",
			args{DataKind_float32, ConstraintKind_Enum, []float32{1.0, 2.0, 3.0}}, false},
		{"float32: enum constraint must fail if wrong enum type",
			args{DataKind_float32, ConstraintKind_Enum, []float64{1.0, 2.0, 3.0}}, true},
		{"float64: enum constraint must be ok",
			args{DataKind_float64, ConstraintKind_Enum, []float64{1.0, 2.0, 3.0}}, false},
		{"float64: enum constraint must fail if wrong enum type",
			args{DataKind_float64, ConstraintKind_Enum, []int32{1, 2, 3}}, true},
		{"string: enum constraint must be ok",
			args{DataKind_string, ConstraintKind_Enum, []string{"a", "b", "c"}}, false},
		{"string: enum constraint must fail if wrong enum type",
			args{DataKind_float64, ConstraintKind_Enum, []int32{1, 2, 3}}, true},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adb := New()
			d := adb.AddData(NewQName("test", "test"), tt.args.da, NullQName)
			if tt.wantPanic {
				require.Panics(func() { d.AddConstraints(NewConstraint(tt.args.ck, tt.args.cv)) })
			} else {
				require.NotPanics(func() { d.AddConstraints(NewConstraint(tt.args.ck, tt.args.cv)) })
			}
		})
	}
}
