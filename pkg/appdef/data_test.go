/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"regexp"
	"strconv"
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
		for k, c := range tk.Constraints(false) {
			cnt++
			switch k {
			case ConstraintKind_MinLen:
				require.Equal(ConstraintKind_MinLen, c.Kind())
				require.EqualValues(1, c.Value())
			case ConstraintKind_MaxLen:
				require.Equal(ConstraintKind_MaxLen, c.Kind())
				require.EqualValues(100, c.Value())
			case ConstraintKind_Pattern:
				require.Equal(ConstraintKind_Pattern, c.Kind())
				require.EqualValues(`^\w+$`, c.Value().(*regexp.Regexp).String())
				require.Equal("only word characters allowed", c.Comment())
			default:
				require.Failf("unexpected constraint", "constraint: %v", c)
			}
		}
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
		{"bytes: max length constraint must be ok",
			args{DataKind_bytes, ConstraintKind_MaxLen, uint16(1024)}, false},
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

func TestDataKindType_IsFixed(t *testing.T) {
	type args struct {
		kind DataKind
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "int32 must be fixed",
			args: args{kind: DataKind_int32},
			want: true},
		{name: "string must be variable",
			args: args{kind: DataKind_string},
			want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.kind.IsFixed(); got != tt.want {
				t.Errorf("%v.IsFixed() = %v, want %v", tt.args.kind, got, tt.want)
			}
		})
	}
}

func TestDataKindType_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		k    DataKind
		want string
	}{
		{
			name: `0 —> "DataKind_null"`,
			k:    DataKind_null,
			want: `DataKind_null`,
		},
		{
			name: `1 —> "DataKind_int32"`,
			k:    DataKind_int32,
			want: `DataKind_int32`,
		},
		{
			name: `DataKind_FakeLast —> 12`,
			k:    DataKind_FakeLast,
			want: strconv.FormatUint(uint64(DataKind_FakeLast), 10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalText()
			if err != nil {
				t.Errorf("DataKind.MarshalText() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("DataKind.MarshalText() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("100% cover", func(t *testing.T) {
		const tested = DataKind_FakeLast + 1
		want := "DataKind(" + strconv.FormatInt(int64(tested), 10) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(DataKind_FakeLast + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestDataKind_TrimString(t *testing.T) {
	tests := []struct {
		name string
		k    DataKind
		want string
	}{
		{name: "basic", k: DataKind_int32, want: "int32"},
		{name: "out of range", k: DataKind_FakeLast + 1, want: (DataKind_FakeLast + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(DataKind).TrimString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}

func TestDataKind_IsSupportedConstraint(t *testing.T) {
	type args struct {
		c ConstraintKind
	}
	tests := []struct {
		name string
		k    DataKind
		args args
		want bool
	}{
		{"string: MinLen", DataKind_string, args{ConstraintKind_MinLen}, true},
		{"string: MaxLen", DataKind_string, args{ConstraintKind_MaxLen}, true},
		{"string: Pattern", DataKind_string, args{ConstraintKind_Pattern}, true},
		{"string: MinIncl", DataKind_string, args{ConstraintKind_MinIncl}, false},
		{"string: MinExcl", DataKind_string, args{ConstraintKind_MinExcl}, false},
		{"string: MaxIncl", DataKind_string, args{ConstraintKind_MaxIncl}, false},
		{"string: MaxExcl", DataKind_string, args{ConstraintKind_MaxExcl}, false},
		{"string: Enum", DataKind_string, args{ConstraintKind_Enum}, true},
		//-
		{"bytes: MinLen", DataKind_bytes, args{ConstraintKind_MinLen}, true},
		{"bytes: MaxLen", DataKind_bytes, args{ConstraintKind_MaxLen}, true},
		{"bytes: Pattern", DataKind_bytes, args{ConstraintKind_Pattern}, true},
		{"bytes: MinIncl", DataKind_bytes, args{ConstraintKind_MinIncl}, false},
		{"bytes: MinExcl", DataKind_bytes, args{ConstraintKind_MinExcl}, false},
		{"bytes: MaxIncl", DataKind_bytes, args{ConstraintKind_MaxIncl}, false},
		{"bytes: MaxExcl", DataKind_bytes, args{ConstraintKind_MaxExcl}, false},
		{"bytes: Enum", DataKind_bytes, args{ConstraintKind_Enum}, false},
		//-
		{"int32: MinLen", DataKind_int32, args{ConstraintKind_MinLen}, false},
		{"int32: MaxLen", DataKind_int32, args{ConstraintKind_MaxLen}, false},
		{"int32: Pattern", DataKind_int32, args{ConstraintKind_Pattern}, false},
		{"int32: MinIncl", DataKind_int32, args{ConstraintKind_MinIncl}, true},
		{"int32: MinExcl", DataKind_int32, args{ConstraintKind_MinExcl}, true},
		{"int32: MaxIncl", DataKind_int32, args{ConstraintKind_MaxIncl}, true},
		{"int32: MaxExcl", DataKind_int32, args{ConstraintKind_MaxExcl}, true},
		{"int32: Enum", DataKind_int32, args{ConstraintKind_Enum}, true},
		//-
		{"int64: MinLen", DataKind_int64, args{ConstraintKind_MinLen}, false},
		{"int64: MaxLen", DataKind_int64, args{ConstraintKind_MaxLen}, false},
		{"int64: Pattern", DataKind_int64, args{ConstraintKind_Pattern}, false},
		{"int64: MinIncl", DataKind_int64, args{ConstraintKind_MinIncl}, true},
		{"int64: MinExcl", DataKind_int64, args{ConstraintKind_MinExcl}, true},
		{"int64: MaxIncl", DataKind_int64, args{ConstraintKind_MaxIncl}, true},
		{"int64: MaxExcl", DataKind_int64, args{ConstraintKind_MaxExcl}, true},
		{"int64: Enum", DataKind_int64, args{ConstraintKind_Enum}, true},
		//-
		{"float32: MinLen", DataKind_float32, args{ConstraintKind_MinLen}, false},
		{"float32: MaxLen", DataKind_float32, args{ConstraintKind_MaxLen}, false},
		{"float32: Pattern", DataKind_float32, args{ConstraintKind_Pattern}, false},
		{"float32: MinIncl", DataKind_float32, args{ConstraintKind_MinIncl}, true},
		{"float32: MinExcl", DataKind_float32, args{ConstraintKind_MinExcl}, true},
		{"float32: MaxIncl", DataKind_float32, args{ConstraintKind_MaxIncl}, true},
		{"float32: MaxExcl", DataKind_float32, args{ConstraintKind_MaxExcl}, true},
		{"float32: Enum", DataKind_float32, args{ConstraintKind_Enum}, true},
		//-
		{"float64: MinLen", DataKind_float64, args{ConstraintKind_MinLen}, false},
		{"float64: MaxLen", DataKind_float64, args{ConstraintKind_MaxLen}, false},
		{"float64: Pattern", DataKind_float64, args{ConstraintKind_Pattern}, false},
		{"float64: MinIncl", DataKind_float64, args{ConstraintKind_MinIncl}, true},
		{"float64: MinExcl", DataKind_float64, args{ConstraintKind_MinExcl}, true},
		{"float64: MaxIncl", DataKind_float64, args{ConstraintKind_MaxIncl}, true},
		{"float64: MaxExcl", DataKind_float64, args{ConstraintKind_MaxExcl}, true},
		{"float64: Enum", DataKind_float64, args{ConstraintKind_Enum}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.IsSupportedConstraint(tt.args.c); got != tt.want {
				t.Errorf("%v.IsSupportedConstraint(%v) = %v, want %v", tt.k.TrimString(), tt.args.c.TrimString(), got, tt.want)
			}
		})
	}
}
