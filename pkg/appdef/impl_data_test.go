/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_AppDef_AddData(t *testing.T) {
	require := require.New(t)

	var app IAppDef

	intName := NewQName("test", "int")
	strName := NewQName("test", "string")
	tokenName := NewQName("test", "token")

	t.Run("must be ok to add data types", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		_ = adb.AddData(intName, DataKind_int64, NullQName)
		_ = adb.AddData(strName, DataKind_string, NullQName)
		token := adb.AddData(tokenName, DataKind_string, strName)
		token.AddConstraints(MinLen(1), MaxLen(100), Pattern(`^\w+$`, "only word characters allowed"))

		t.Run("must be ok to build", func(t *testing.T) {
			a, err := adb.Build()
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

	t.Run("should be ok to enum data types", func(t *testing.T) {
		cnt := 0
		for d := range app.DataTypes {
			if !d.IsSystem() {
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
			}
		}
		require.Equal(3, cnt)
	})

	t.Run("data types range should be breakable", func(t *testing.T) {
		cnt := 0
		for range app.DataTypes {
			cnt++
			break
		}
		require.Equal(1, cnt)
	})

	require.Nil(app.Data(NewQName("test", "unknown")), "check nil returns")

	require.Panics(func() {
		New().AddData(NullQName, DataKind_int64, NullQName)
	}, require.Is(ErrMissedError))

	require.Panics(func() {
		New().AddData(NewQName("naked", "🔫"), DataKind_QName, NullQName)
	}, require.Is(ErrInvalidError), require.Has("naked.🔫"))

	t.Run("panic if type with name already exists", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")
		adb.AddObject(intName)
		require.Panics(func() {
			adb.AddData(intName, DataKind_int64, NullQName)
		}, require.Is(ErrAlreadyExistsError), require.Has(intName.String()))
	})

	require.Panics(func() {
		New().AddData(intName, DataKind_null, NullQName)
	}, require.Is(ErrNotFoundError))

	require.Panics(func() {
		New().AddData(intName, DataKind_int64,
			NewQName("test", "unknown"), // <- error here
		)
	}, require.Is(ErrNotFoundError), require.Has("test.unknown"))

	t.Run("panic if ancestor is not data type", func(t *testing.T) {
		objName := NewQName("test", "object")
		adb := New()
		adb.AddPackage("test", "test.com/test")
		_ = adb.AddObject(objName)
		require.Panics(func() {
			adb.AddData(intName, DataKind_int64,
				objName, // <- error here
			)
		}, require.Is(ErrNotFoundError), require.Has(objName.String()))
	})

	t.Run("panic if ancestor has different kind", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")
		_ = adb.AddData(strName, DataKind_string, NullQName)
		require.Panics(func() {
			adb.AddData(intName, DataKind_int64, strName)
		}, require.Is(ErrInvalidError), require.Has(strName.String()))
	})

	t.Run("panic if incompatible constraints", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")
		require.Panics(func() { _ = adb.AddData(strName, DataKind_string, NullQName, MinIncl(1)) },
			require.Is(ErrIncompatibleError), require.Has("MinIncl"))
		require.Panics(func() { _ = adb.AddData(intName, DataKind_float64, NullQName, MaxLen(100)) },
			require.Is(ErrIncompatibleError), require.Has("MaxLen"))
	})
}

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
			require.True(d.IsSystem())
			require.Equal(k, d.DataKind())
			require.Nil(d.Ancestor())
			require.Empty(d.Constraints(false))
		}
	})
}

func TestNewConstraint(t *testing.T) {
	type args struct {
		kind  ConstraintKind
		value any
		c     []string
	}
	tests := []struct {
		name string
		args args
		want args
	}{
		{"Min length",
			args{ConstraintKind_MinLen, uint16(1), []string{"test min length"}},
			args{ConstraintKind_MinLen, 1, []string{"test min length"}},
		},
		{"Max length",
			args{ConstraintKind_MaxLen, uint16(100), []string{"test max length"}},
			args{ConstraintKind_MaxLen, 100, []string{"test max length"}},
		},
		{"Pattern",
			args{ConstraintKind_Pattern, "^/w+$", []string{"test pattern"}},
			args{ConstraintKind_Pattern, regexp.MustCompile("^/w+$"), []string{"test pattern"}},
		},
		{"Min inclusive",
			args{ConstraintKind_MinIncl, float64(1), []string{"test min inclusive"}},
			args{ConstraintKind_MinIncl, 1, []string{"test min inclusive"}},
		},
		{"Min exclusive",
			args{ConstraintKind_MinExcl, float64(1), []string{"test min exclusive"}},
			args{ConstraintKind_MinExcl, 1, []string{"test min exclusive"}},
		},
		{"Max inclusive",
			args{ConstraintKind_MaxIncl, float64(1), []string{"test max inclusive"}},
			args{ConstraintKind_MaxIncl, 1, []string{"test max inclusive"}},
		},
		{"Max exclusive",
			args{ConstraintKind_MaxExcl, float64(1), []string{"test max exclusive"}},
			args{ConstraintKind_MaxExcl, 1, []string{"test max exclusive"}},
		},
		{"string enumeration",
			args{ConstraintKind_Enum, []string{"c", "b", "a", "b"}, []string{"test string enum"}},
			args{ConstraintKind_Enum, []string{"a", "b", "c"}, []string{"test string enum"}},
		},
		{"int32 enumeration",
			args{ConstraintKind_Enum, []int32{3, 2, 1, 3}, []string{"test int32 enum"}},
			args{ConstraintKind_Enum, []int32{1, 2, 3}, []string{"test int32 enum"}},
		},
		{"int64 enumeration",
			args{ConstraintKind_Enum, []int64{3, 2, 1, 2}, []string{}},
			args{ConstraintKind_Enum, []int64{1, 2, 3}, []string{}},
		},
		{"float32 enumeration",
			args{ConstraintKind_Enum, []float32{1, 3, 2, 1}, []string{"test", "float32", "enum"}},
			args{ConstraintKind_Enum, []float32{1, 2, 3}, []string{"test", "float32", "enum"}},
		},
		{"float64 enumeration",
			args{ConstraintKind_Enum, []float64{3, 1, 2, 2, 3}, []string{"test float64 enum"}},
			args{ConstraintKind_Enum, []float64{1, 2, 3}, []string{"test float64 enum"}},
		},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConstraint(tt.args.kind, tt.args.value, tt.args.c...)
			require.NotNil(c)
			require.Equal(tt.want.kind, c.Kind())
			require.EqualValues(tt.want.value, c.Value())
			require.EqualValues(tt.want.c, c.CommentLines())
		})
	}
}

func TestNewConstraintPanics(t *testing.T) {
	type args struct {
		kind  ConstraintKind
		value any
	}
	tests := []struct {
		name string
		args args
		e    error
	}{
		{"MaxLen(0)",
			args{ConstraintKind_MaxLen, uint16(0)}, ErrOutOfBoundsError,
		},
		{"Pattern(`^[error$`)",
			args{ConstraintKind_Pattern, `^[error$`}, nil,
		},
		{"MinIncl(+∞)",
			args{ConstraintKind_MinIncl, math.NaN()}, ErrInvalidError,
		},
		{"MinIncl(+∞)",
			args{ConstraintKind_MinIncl, math.Inf(+1)}, ErrOutOfBoundsError,
		},
		{"MinExcl(NaN)",
			args{ConstraintKind_MinExcl, math.NaN()}, ErrInvalidError,
		},
		{"MinExcl(+∞)",
			args{ConstraintKind_MinExcl, math.Inf(+1)}, ErrOutOfBoundsError,
		},
		{"MaxIncl(NaN)",
			args{ConstraintKind_MaxIncl, math.NaN()}, ErrInvalidError,
		},
		{"MaxIncl(-∞)",
			args{ConstraintKind_MaxIncl, math.Inf(-1)}, ErrOutOfBoundsError,
		},
		{"MaxExcl(NaN)",
			args{ConstraintKind_MaxExcl, math.NaN()}, ErrInvalidError,
		},
		{"MaxExcl(-∞)",
			args{ConstraintKind_MaxExcl, math.Inf(-1)}, ErrOutOfBoundsError,
		},
		{"Enum([]string{})",
			args{ConstraintKind_Enum, []string{}}, ErrMissedError,
		},
		{"Enum([]bool)",
			args{ConstraintKind_Enum, []bool{true, false}}, ErrUnsupportedError,
		},
		{"Enum([][]byte)",
			args{ConstraintKind_Enum, [][]byte{{1, 2, 3}, {4, 5, 6}}}, ErrUnsupportedError,
		},
		{"???(0)",
			args{ConstraintKind_Count, 0}, ErrUnsupportedError,
		},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.e == nil {
				require.Panics(func() { _ = NewConstraint(tt.args.kind, tt.args.value) })
			} else {
				require.Panics(func() { _ = NewConstraint(tt.args.kind, tt.args.value) },
					require.Is(tt.e))
			}
		})
	}
}

func Test_dataConstraint_String(t *testing.T) {
	tests := []struct {
		name  string
		c     IConstraint
		wantS string
	}{
		{"MinLen", MinLen(1), "MinLen: 1"},
		{"MaxLen", MaxLen(100), "MaxLen: 100"},
		{"Pattern", Pattern(`^\d+$`), "Pattern: `^\\d+$`"},
		{"MinIncl", MinIncl(1), "MinIncl: 1"},
		{"MinExcl", MinExcl(0), "MinExcl: 0"},
		{"MinExcl(-∞)", MinExcl(math.Inf(-1)), "MinExcl: -Inf"},
		{"MaxIncl", MaxIncl(100), "MaxIncl: 100"},
		{"MaxExcl", MaxExcl(100), "MaxExcl: 100"},
		{"MaxExcl(+∞)", MaxExcl(math.Inf(+1)), "MaxExcl: +Inf"},
		{"Enum(string)", Enum("c", "d", "a", "a", "b", "c"), "Enum: [a b c d]"},
		{"Enum(float64)", Enum(float64(1), 2, 3, 4, math.Round(100*math.Pi)/100, math.Inf(-1)), "Enum: [-Inf 1 2 3 3.14 4]"},
		{"Enum(long case)", Enum("b", "d", "a", strings.Repeat("c", 100)), "Enum: [a b cccccccccccccccccccccccccccccccccccccccccccccccccccc…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotS := fmt.Sprint(tt.c); gotS != tt.wantS {
				t.Errorf("dataConstraint.String() = %v, want %v", gotS, tt.wantS)
			}
		})
	}
}

func TestConstraintKind_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		k    ConstraintKind
		want string
	}{
		{
			name: `0 —> "ConstraintKind_null"`,
			k:    ConstraintKind_null,
			want: `ConstraintKind_null`,
		},
		{
			name: `1 —> "ConstraintKind_MinLen"`,
			k:    ConstraintKind_MinLen,
			want: `ConstraintKind_MinLen`,
		},
		{
			name: `ConstraintKind_Count —> 4`,
			k:    ConstraintKind_Count,
			want: utils.UintToString(ConstraintKind_Count),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalText()
			if err != nil {
				t.Errorf("%T.MarshalText() unexpected error %v", tt.k, err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("%T.MarshalText() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}

	t.Run("100% cover", func(t *testing.T) {
		const tested = ConstraintKind_Count + 1
		want := "ConstraintKind(" + utils.UintToString(tested) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(ConstraintKind_Count + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestConstraintKind_TrimString(t *testing.T) {
	tests := []struct {
		name string
		k    ConstraintKind
		want string
	}{
		{name: "basic", k: ConstraintKind_MinLen, want: "MinLen"},
		{name: "out of range", k: ConstraintKind_Count + 1, want: (ConstraintKind_Count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(%T).TrimString() = %v, want %v", tt.k, tt.k, got, tt.want)
			}
		})
	}
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
		e         error
	}{
		//- MaxLen
		{"string: max length constraint should be ok",
			args{DataKind_string, ConstraintKind_MaxLen, uint16(100)}, false, nil},
		{"bytes: max length constraint should be ok",
			args{DataKind_bytes, ConstraintKind_MaxLen, uint16(1024)}, false, nil},
		//- Enum
		{"int32: enum constraint should be ok",
			args{DataKind_int32, ConstraintKind_Enum, []int32{1, 2, 3}}, false, nil},
		{"int32: enum constraint should fail if incompatible enum type",
			args{DataKind_int32, ConstraintKind_Enum, []int64{1, 2, 3}}, true, ErrIncompatibleError},
		{"int64: enum constraint should be ok",
			args{DataKind_int64, ConstraintKind_Enum, []int64{1, 2, 3}}, false, nil},
		{"int64: enum constraint should fail if incompatible ErrIncompatibleError type",
			args{DataKind_int64, ConstraintKind_Enum, []string{"1", "2", "3"}}, true, ErrIncompatibleError},
		{"float32: enum constraint should be ok",
			args{DataKind_float32, ConstraintKind_Enum, []float32{1.0, 2.0, 3.0}}, false, nil},
		{"float32: enum constraint should fail if incompatible enum type",
			args{DataKind_float32, ConstraintKind_Enum, []float64{1.0, 2.0, 3.0}}, true, ErrIncompatibleError},
		{"float64: enum constraint should be ok",
			args{DataKind_float64, ConstraintKind_Enum, []float64{1.0, 2.0, 3.0}}, false, nil},
		{"float64: enum constraint should fail if incompatible enum type",
			args{DataKind_float64, ConstraintKind_Enum, []int32{1, 2, 3}}, true, ErrIncompatibleError},
		{"string: enum constraint should be ok",
			args{DataKind_string, ConstraintKind_Enum, []string{"a", "b", "c"}}, false, nil},
		{"string: enum constraint should fail if incompatible enum type",
			args{DataKind_float64, ConstraintKind_Enum, []int32{1, 2, 3}}, true, ErrIncompatibleError},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			d := adb.AddData(NewQName("test", "test"), tt.args.da, NullQName)
			if tt.wantPanic {
				if tt.e == nil {
					require.Panics(func() { d.AddConstraints(NewConstraint(tt.args.ck, tt.args.cv)) })
				} else {
					require.Panics(func() { d.AddConstraints(NewConstraint(tt.args.ck, tt.args.cv)) },
						require.Is(tt.e))
				}
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
			want: utils.UintToString(DataKind_FakeLast),
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
		want := "DataKind(" + utils.UintToString(tested) + ")"
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
			if got := tt.k.IsCompatibleWithConstraint(tt.args.c); got != tt.want {
				t.Errorf("%v.IsCompatibleWithConstraint(%v) = %v, want %v", tt.k.TrimString(), tt.args.c.TrimString(), got, tt.want)
			}
		})
	}
}
