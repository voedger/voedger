/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"
	"maps"
	"math"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_AppDef_AddData(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")
	intName := appdef.NewQName("test", "int")
	strName := appdef.NewQName("test", "string")
	tokenName := appdef.NewQName("test", "token")

	t.Run("should be ok to add data types", func(t *testing.T) {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)

		_ = ws.AddData(intName, appdef.DataKind_int64, appdef.NullQName)
		_ = ws.AddData(strName, appdef.DataKind_string, appdef.NullQName)
		token := ws.AddData(tokenName, appdef.DataKind_string, strName)
		token.AddConstraints(appdef.MinLen(1), appdef.MaxLen(100), appdef.Pattern(`^\w+$`, "only word characters allowed"))

		t.Run("should be ok to build", func(t *testing.T) {
			a, err := adb.Build()
			require.NoError(err)
			require.NotNil(a)

			app = a
		})
	})

	require.NotNil(app)

	testWith := func(tested testedTypes) {
		t.Run("should be ok to find builded data type", func(t *testing.T) {
			i := appdef.Data(tested.Type, intName)
			require.Equal(appdef.TypeKind_Data, i.Kind())
			require.Equal(intName, i.QName())
			require.Equal(appdef.DataKind_int64, i.DataKind())
			require.False(i.IsSystem())
			require.Equal(appdef.SysData(tested.Type, appdef.DataKind_int64), i.Ancestor())

			s := appdef.Data(tested.Type, strName)
			require.Equal(appdef.TypeKind_Data, s.Kind())
			require.Equal(strName, s.QName())
			require.Equal(appdef.DataKind_string, s.DataKind())
			require.Equal(appdef.SysData(tested.Type, appdef.DataKind_string), s.Ancestor())

			tk := appdef.Data(tested.Type, tokenName)
			require.Equal(appdef.TypeKind_Data, tk.Kind())
			require.Equal(tokenName, tk.QName())
			require.Equal(appdef.DataKind_string, tk.DataKind())
			require.Equal(s, tk.Ancestor())
			cnt := 0
			for k, c := range tk.Constraints(false) {
				cnt++
				switch k {
				case appdef.ConstraintKind_MinLen:
					require.Equal(appdef.ConstraintKind_MinLen, c.Kind())
					require.EqualValues(1, c.Value())
				case appdef.ConstraintKind_MaxLen:
					require.Equal(appdef.ConstraintKind_MaxLen, c.Kind())
					require.EqualValues(100, c.Value())
				case appdef.ConstraintKind_Pattern:
					require.Equal(appdef.ConstraintKind_Pattern, c.Kind())
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
			for d := range appdef.DataTypes(tested.Types()) {
				if !d.IsSystem() {
					cnt++
					require.Equal(appdef.TypeKind_Data, d.Kind())
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

		require.Nil(appdef.Data(tested.Type, appdef.NewQName("test", "unknown")), "check nil returns")
	}

	testWith(app)
	testWith(app.Workspace(wsName))

	t.Run("should be panics", func(t *testing.T) {

		t.Run("if data type name missed", func(t *testing.T) {
			wsb := appdef.New().AddWorkspace(wsName)
			require.Panics(func() {
				wsb.AddData(appdef.NullQName, appdef.DataKind_int64, appdef.NullQName)
			}, require.Is(appdef.ErrMissedError))
		})

		t.Run("if invalid data type name", func(t *testing.T) {
			wsb := appdef.New().AddWorkspace(wsName)
			require.Panics(func() {
				wsb.AddData(appdef.NewQName("naked", "🔫"), appdef.DataKind_QName, appdef.NullQName)
			}, require.Is(appdef.ErrInvalidError), require.Has("naked.🔫"))
		})

		t.Run("if type with name already exists", func(t *testing.T) {
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			wsb.AddObject(intName)
			require.Panics(func() {
				wsb.AddData(intName, appdef.DataKind_int64, appdef.NullQName)
			}, require.Is(appdef.ErrAlreadyExistsError), require.Has(intName.String()))
		})

		t.Run("if sys data to inherits from not found", func(t *testing.T) {
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			require.Panics(func() {
				wsb.AddData(strName, appdef.DataKind_null, appdef.NullQName)
			}, require.Is(appdef.ErrNotFoundError), require.Has("null"))
		})

		t.Run("if ancestor not found", func(t *testing.T) {
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			require.Panics(func() {
				wsb.AddData(strName, appdef.DataKind_string,
					appdef.NewQName("test", "unknown"), // <- error here
				)
			}, require.Is(appdef.ErrNotFoundError), require.Has("test.unknown"))
		})

		t.Run("if ancestor is not data type", func(t *testing.T) {
			objName := appdef.NewQName("test", "object")
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			_ = wsb.AddObject(objName)
			require.Panics(func() {
				wsb.AddData(intName, appdef.DataKind_int64,
					objName, // <- error here
				)
			}, require.Is(appdef.ErrNotFoundError), require.Has(objName.String()))
		})

		t.Run("if ancestor has different kind", func(t *testing.T) {
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			_ = wsb.AddData(strName, appdef.DataKind_string, appdef.NullQName)
			require.Panics(func() {
				wsb.AddData(intName, appdef.DataKind_int64, strName)
			}, require.Is(appdef.ErrInvalidError), require.Has(strName.String()))
		})

		t.Run("if incompatible constraints", func(t *testing.T) {
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			require.Panics(func() { _ = wsb.AddData(strName, appdef.DataKind_string, appdef.NullQName, appdef.MinIncl(1)) },
				require.Is(appdef.ErrIncompatibleError), require.Has("MinIncl"))
			require.Panics(func() { _ = wsb.AddData(intName, appdef.DataKind_float64, appdef.NullQName, appdef.MaxLen(100)) },
				require.Is(appdef.ErrIncompatibleError), require.Has("MaxLen"))
		})
	})
}

func Test_SysDataName(t *testing.T) {
	type args struct {
		k appdef.DataKind
	}
	tests := []struct {
		name string
		args args
		want appdef.QName
	}{
		{"null", args{k: appdef.DataKind_null}, appdef.NullQName},
		{"int32", args{k: appdef.DataKind_int32}, appdef.MustParseQName("sys.int32")},
		{"string", args{k: appdef.DataKind_string}, appdef.MustParseQName("sys.string")},
		{"out of bounds", args{k: appdef.DataKind_FakeLast}, appdef.NullQName},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appdef.SysDataName(tt.args.k); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sysDataTypeName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_appDef_makeSysDataTypes(t *testing.T) {
	require := require.New(t)

	app, err := appdef.New().Build()
	require.NoError(err)

	t.Run("must be ok to get system data types", func(t *testing.T) {
		sysWS := app.Workspace(appdef.SysWorkspaceQName)
		for k := appdef.DataKind_null + 1; k < appdef.DataKind_FakeLast; k++ {
			d := appdef.SysData(app.Type, k)
			require.NotNil(d)
			require.Equal(appdef.SysDataName(k), d.QName())
			require.Equal(appdef.TypeKind_Data, d.Kind())
			require.Equal(d.Workspace(), sysWS)
			require.True(d.IsSystem())
			require.Equal(k, d.DataKind())
			require.Nil(d.Ancestor())
			require.Empty(maps.Collect(d.Constraints(false)))
		}
	})
}

func TestNewConstraint(t *testing.T) {
	type args struct {
		kind  appdef.ConstraintKind
		value any
		c     []string
	}
	tests := []struct {
		name string
		args args
		want args
	}{
		{"Min length",
			args{appdef.ConstraintKind_MinLen, uint16(1), []string{"test min length"}},
			args{appdef.ConstraintKind_MinLen, 1, []string{"test min length"}},
		},
		{"Max length",
			args{appdef.ConstraintKind_MaxLen, uint16(100), []string{"test max length"}},
			args{appdef.ConstraintKind_MaxLen, 100, []string{"test max length"}},
		},
		{"Pattern",
			args{appdef.ConstraintKind_Pattern, "^/w+$", []string{"test pattern"}},
			args{appdef.ConstraintKind_Pattern, regexp.MustCompile("^/w+$"), []string{"test pattern"}},
		},
		{"Min inclusive",
			args{appdef.ConstraintKind_MinIncl, float64(1), []string{"test min inclusive"}},
			args{appdef.ConstraintKind_MinIncl, 1, []string{"test min inclusive"}},
		},
		{"Min exclusive",
			args{appdef.ConstraintKind_MinExcl, float64(1), []string{"test min exclusive"}},
			args{appdef.ConstraintKind_MinExcl, 1, []string{"test min exclusive"}},
		},
		{"Max inclusive",
			args{appdef.ConstraintKind_MaxIncl, float64(1), []string{"test max inclusive"}},
			args{appdef.ConstraintKind_MaxIncl, 1, []string{"test max inclusive"}},
		},
		{"Max exclusive",
			args{appdef.ConstraintKind_MaxExcl, float64(1), []string{"test max exclusive"}},
			args{appdef.ConstraintKind_MaxExcl, 1, []string{"test max exclusive"}},
		},
		{"string enumeration",
			args{appdef.ConstraintKind_Enum, []string{"c", "b", "a", "b"}, []string{"test string enum"}},
			args{appdef.ConstraintKind_Enum, []string{"a", "b", "c"}, []string{"test string enum"}},
		},
		{"int32 enumeration",
			args{appdef.ConstraintKind_Enum, []int32{3, 2, 1, 3}, []string{"test int32 enum"}},
			args{appdef.ConstraintKind_Enum, []int32{1, 2, 3}, []string{"test int32 enum"}},
		},
		{"int64 enumeration",
			args{appdef.ConstraintKind_Enum, []int64{3, 2, 1, 2}, nil},
			args{appdef.ConstraintKind_Enum, []int64{1, 2, 3}, nil},
		},
		{"float32 enumeration",
			args{appdef.ConstraintKind_Enum, []float32{1, 3, 2, 1}, []string{"test", "float32", "enum"}},
			args{appdef.ConstraintKind_Enum, []float32{1, 2, 3}, []string{"test", "float32", "enum"}},
		},
		{"float64 enumeration",
			args{appdef.ConstraintKind_Enum, []float64{3, 1, 2, 2, 3}, []string{"test float64 enum"}},
			args{appdef.ConstraintKind_Enum, []float64{1, 2, 3}, []string{"test float64 enum"}},
		},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := appdef.NewConstraint(tt.args.kind, tt.args.value, tt.args.c...)
			require.NotNil(c)
			require.Equal(tt.want.kind, c.Kind())
			require.EqualValues(tt.want.value, c.Value())
			require.EqualValues(tt.want.c, slices.Collect(c.CommentLines()))
		})
	}
}

func TestNewConstraintPanics(t *testing.T) {
	type args struct {
		kind  appdef.ConstraintKind
		value any
	}
	tests := []struct {
		name string
		args args
		e    error
	}{
		{"MaxLen(0)",
			args{appdef.ConstraintKind_MaxLen, uint16(0)}, appdef.ErrOutOfBoundsError,
		},
		{"Pattern(`^[error$`)",
			args{appdef.ConstraintKind_Pattern, `^[error$`}, nil,
		},
		{"MinIncl(+∞)",
			args{appdef.ConstraintKind_MinIncl, math.NaN()}, appdef.ErrInvalidError,
		},
		{"MinIncl(+∞)",
			args{appdef.ConstraintKind_MinIncl, math.Inf(+1)}, appdef.ErrOutOfBoundsError,
		},
		{"MinExcl(NaN)",
			args{appdef.ConstraintKind_MinExcl, math.NaN()}, appdef.ErrInvalidError,
		},
		{"MinExcl(+∞)",
			args{appdef.ConstraintKind_MinExcl, math.Inf(+1)}, appdef.ErrOutOfBoundsError,
		},
		{"MaxIncl(NaN)",
			args{appdef.ConstraintKind_MaxIncl, math.NaN()}, appdef.ErrInvalidError,
		},
		{"MaxIncl(-∞)",
			args{appdef.ConstraintKind_MaxIncl, math.Inf(-1)}, appdef.ErrOutOfBoundsError,
		},
		{"MaxExcl(NaN)",
			args{appdef.ConstraintKind_MaxExcl, math.NaN()}, appdef.ErrInvalidError,
		},
		{"MaxExcl(-∞)",
			args{appdef.ConstraintKind_MaxExcl, math.Inf(-1)}, appdef.ErrOutOfBoundsError,
		},
		{"Enum([]string{})",
			args{appdef.ConstraintKind_Enum, []string{}}, appdef.ErrMissedError,
		},
		{"Enum([]bool)",
			args{appdef.ConstraintKind_Enum, []bool{true, false}}, appdef.ErrUnsupportedError,
		},
		{"Enum([][]byte)",
			args{appdef.ConstraintKind_Enum, [][]byte{{1, 2, 3}, {4, 5, 6}}}, appdef.ErrUnsupportedError,
		},
		{"???(0)",
			args{appdef.ConstraintKind_count, 0}, appdef.ErrUnsupportedError,
		},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.e == nil {
				require.Panics(func() { _ = appdef.NewConstraint(tt.args.kind, tt.args.value) })
			} else {
				require.Panics(func() { _ = appdef.NewConstraint(tt.args.kind, tt.args.value) },
					require.Is(tt.e))
			}
		})
	}
}

func Test_dataConstraint_String(t *testing.T) {
	tests := []struct {
		name  string
		c     appdef.IConstraint
		wantS string
	}{
		{"MinLen", appdef.MinLen(1), "MinLen: 1"},
		{"MaxLen", appdef.MaxLen(100), "MaxLen: 100"},
		{"Pattern", appdef.Pattern(`^\d+$`), "Pattern: `^\\d+$`"},
		{"MinIncl", appdef.MinIncl(1), "MinIncl: 1"},
		{"MinExcl", appdef.MinExcl(0), "MinExcl: 0"},
		{"MinExcl(-∞)", appdef.MinExcl(math.Inf(-1)), "MinExcl: -Inf"},
		{"MaxIncl", appdef.MaxIncl(100), "MaxIncl: 100"},
		{"MaxExcl", appdef.MaxExcl(100), "MaxExcl: 100"},
		{"MaxExcl(+∞)", appdef.MaxExcl(math.Inf(+1)), "MaxExcl: +Inf"},
		{"Enum(string)", appdef.Enum("c", "d", "a", "a", "b", "c"), "Enum: [a b c d]"},
		{"Enum(float64)", appdef.Enum(float64(1), 2, 3, 4, math.Round(100*math.Pi)/100, math.Inf(-1)), "Enum: [-Inf 1 2 3 3.14 4]"},
		{"Enum(long case)", appdef.Enum("b", "d", "a", strings.Repeat("c", 100)), "Enum: [a b cccccccccccccccccccccccccccccccccccccccccccccccccccc…"},
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
		k    appdef.ConstraintKind
		want string
	}{
		{
			name: `0 —> "ConstraintKind_null"`,
			k:    appdef.ConstraintKind_null,
			want: `ConstraintKind_null`,
		},
		{
			name: `1 —> "ConstraintKind_MinLen"`,
			k:    appdef.ConstraintKind_MinLen,
			want: `ConstraintKind_MinLen`,
		},
		{
			name: `ConstraintKind_count —> 4`,
			k:    appdef.ConstraintKind_count,
			want: utils.UintToString(appdef.ConstraintKind_count),
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
		const tested = appdef.ConstraintKind_count + 1
		want := "ConstraintKind(" + utils.UintToString(tested) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(ConstraintKind_count + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestConstraintKind_TrimString(t *testing.T) {
	tests := []struct {
		name string
		k    appdef.ConstraintKind
		want string
	}{
		{name: "basic", k: appdef.ConstraintKind_MinLen, want: "MinLen"},
		{name: "out of range", k: appdef.ConstraintKind_count + 1, want: (appdef.ConstraintKind_count + 1).String()},
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
		da appdef.DataKind
		ck appdef.ConstraintKind
		cv any
	}
	tests := []struct {
		name      string
		args      args
		wantPanic bool
		e         error
	}{
		//- appdef.MaxLen
		{"string: max length constraint should be ok",
			args{appdef.DataKind_string, appdef.ConstraintKind_MaxLen, uint16(100)}, false, nil},
		{"bytes: max length constraint should be ok",
			args{appdef.DataKind_bytes, appdef.ConstraintKind_MaxLen, uint16(1024)}, false, nil},
		//- Enum
		{"int32: enum constraint should be ok",
			args{appdef.DataKind_int32, appdef.ConstraintKind_Enum, []int32{1, 2, 3}}, false, nil},
		{"int32: enum constraint should fail if incompatible enum type",
			args{appdef.DataKind_int32, appdef.ConstraintKind_Enum, []int64{1, 2, 3}}, true, appdef.ErrIncompatibleError},
		{"int64: enum constraint should be ok",
			args{appdef.DataKind_int64, appdef.ConstraintKind_Enum, []int64{1, 2, 3}}, false, nil},
		{"int64: enum constraint should fail if incompatible appdef.ErrIncompatibleError type",
			args{appdef.DataKind_int64, appdef.ConstraintKind_Enum, []string{"1", "2", "3"}}, true, appdef.ErrIncompatibleError},
		{"float32: enum constraint should be ok",
			args{appdef.DataKind_float32, appdef.ConstraintKind_Enum, []float32{1.0, 2.0, 3.0}}, false, nil},
		{"float32: enum constraint should fail if incompatible enum type",
			args{appdef.DataKind_float32, appdef.ConstraintKind_Enum, []float64{1.0, 2.0, 3.0}}, true, appdef.ErrIncompatibleError},
		{"float64: enum constraint should be ok",
			args{appdef.DataKind_float64, appdef.ConstraintKind_Enum, []float64{1.0, 2.0, 3.0}}, false, nil},
		{"float64: enum constraint should fail if incompatible enum type",
			args{appdef.DataKind_float64, appdef.ConstraintKind_Enum, []int32{1, 2, 3}}, true, appdef.ErrIncompatibleError},
		{"string: enum constraint should be ok",
			args{appdef.DataKind_string, appdef.ConstraintKind_Enum, []string{"a", "b", "c"}}, false, nil},
		{"string: enum constraint should fail if incompatible enum type",
			args{appdef.DataKind_float64, appdef.ConstraintKind_Enum, []int32{1, 2, 3}}, true, appdef.ErrIncompatibleError},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
			d := wsb.AddData(appdef.NewQName("test", "test"), tt.args.da, appdef.NullQName)
			if tt.wantPanic {
				if tt.e == nil {
					require.Panics(func() { d.AddConstraints(appdef.NewConstraint(tt.args.ck, tt.args.cv)) })
				} else {
					require.Panics(func() { d.AddConstraints(appdef.NewConstraint(tt.args.ck, tt.args.cv)) },
						require.Is(tt.e))
				}
			} else {
				require.NotPanics(func() { d.AddConstraints(appdef.NewConstraint(tt.args.ck, tt.args.cv)) })
			}
		})
	}
}

func TestDataKindType_IsFixed(t *testing.T) {
	type args struct {
		kind appdef.DataKind
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "int32 must be fixed",
			args: args{kind: appdef.DataKind_int32},
			want: true},
		{name: "string must be variable",
			args: args{kind: appdef.DataKind_string},
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
		k    appdef.DataKind
		want string
	}{
		{
			name: `0 —> "DataKind_null"`,
			k:    appdef.DataKind_null,
			want: `DataKind_null`,
		},
		{
			name: `1 —> "DataKind_int32"`,
			k:    appdef.DataKind_int32,
			want: `DataKind_int32`,
		},
		{
			name: `DataKind_FakeLast —> 12`,
			k:    appdef.DataKind_FakeLast,
			want: utils.UintToString(appdef.DataKind_FakeLast),
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
		const tested = appdef.DataKind_FakeLast + 1
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
		k    appdef.DataKind
		want string
	}{
		{name: "basic", k: appdef.DataKind_int32, want: "int32"},
		{name: "out of range", k: appdef.DataKind_FakeLast + 1, want: (appdef.DataKind_FakeLast + 1).String()},
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
		c appdef.ConstraintKind
	}
	tests := []struct {
		name string
		k    appdef.DataKind
		args args
		want bool
	}{
		{"string: MinLen", appdef.DataKind_string, args{appdef.ConstraintKind_MinLen}, true},
		{"string: MaxLen", appdef.DataKind_string, args{appdef.ConstraintKind_MaxLen}, true},
		{"string: Pattern", appdef.DataKind_string, args{appdef.ConstraintKind_Pattern}, true},
		{"string: MinIncl", appdef.DataKind_string, args{appdef.ConstraintKind_MinIncl}, false},
		{"string: MinExcl", appdef.DataKind_string, args{appdef.ConstraintKind_MinExcl}, false},
		{"string: MaxIncl", appdef.DataKind_string, args{appdef.ConstraintKind_MaxIncl}, false},
		{"string: MaxExcl", appdef.DataKind_string, args{appdef.ConstraintKind_MaxExcl}, false},
		{"string: Enum", appdef.DataKind_string, args{appdef.ConstraintKind_Enum}, true},
		//-
		{"bytes: MinLen", appdef.DataKind_bytes, args{appdef.ConstraintKind_MinLen}, true},
		{"bytes: MaxLen", appdef.DataKind_bytes, args{appdef.ConstraintKind_MaxLen}, true},
		{"bytes: Pattern", appdef.DataKind_bytes, args{appdef.ConstraintKind_Pattern}, true},
		{"bytes: MinIncl", appdef.DataKind_bytes, args{appdef.ConstraintKind_MinIncl}, false},
		{"bytes: MinExcl", appdef.DataKind_bytes, args{appdef.ConstraintKind_MinExcl}, false},
		{"bytes: MaxIncl", appdef.DataKind_bytes, args{appdef.ConstraintKind_MaxIncl}, false},
		{"bytes: MaxExcl", appdef.DataKind_bytes, args{appdef.ConstraintKind_MaxExcl}, false},
		{"bytes: Enum", appdef.DataKind_bytes, args{appdef.ConstraintKind_Enum}, false},
		//-
		{"int32: MinLen", appdef.DataKind_int32, args{appdef.ConstraintKind_MinLen}, false},
		{"int32: MaxLen", appdef.DataKind_int32, args{appdef.ConstraintKind_MaxLen}, false},
		{"int32: Pattern", appdef.DataKind_int32, args{appdef.ConstraintKind_Pattern}, false},
		{"int32: MinIncl", appdef.DataKind_int32, args{appdef.ConstraintKind_MinIncl}, true},
		{"int32: MinExcl", appdef.DataKind_int32, args{appdef.ConstraintKind_MinExcl}, true},
		{"int32: MaxIncl", appdef.DataKind_int32, args{appdef.ConstraintKind_MaxIncl}, true},
		{"int32: MaxExcl", appdef.DataKind_int32, args{appdef.ConstraintKind_MaxExcl}, true},
		{"int32: Enum", appdef.DataKind_int32, args{appdef.ConstraintKind_Enum}, true},
		//-
		{"int64: MinLen", appdef.DataKind_int64, args{appdef.ConstraintKind_MinLen}, false},
		{"int64: MaxLen", appdef.DataKind_int64, args{appdef.ConstraintKind_MaxLen}, false},
		{"int64: Pattern", appdef.DataKind_int64, args{appdef.ConstraintKind_Pattern}, false},
		{"int64: MinIncl", appdef.DataKind_int64, args{appdef.ConstraintKind_MinIncl}, true},
		{"int64: MinExcl", appdef.DataKind_int64, args{appdef.ConstraintKind_MinExcl}, true},
		{"int64: MaxIncl", appdef.DataKind_int64, args{appdef.ConstraintKind_MaxIncl}, true},
		{"int64: MaxExcl", appdef.DataKind_int64, args{appdef.ConstraintKind_MaxExcl}, true},
		{"int64: Enum", appdef.DataKind_int64, args{appdef.ConstraintKind_Enum}, true},
		//-
		{"float32: appdef.MinLen", appdef.DataKind_float32, args{appdef.ConstraintKind_MinLen}, false},
		{"float32: appdef.MaxLen", appdef.DataKind_float32, args{appdef.ConstraintKind_MaxLen}, false},
		{"float32: appdef.Pattern", appdef.DataKind_float32, args{appdef.ConstraintKind_Pattern}, false},
		{"float32: appdef.MinIncl", appdef.DataKind_float32, args{appdef.ConstraintKind_MinIncl}, true},
		{"float32: MinExcl", appdef.DataKind_float32, args{appdef.ConstraintKind_MinExcl}, true},
		{"float32: MaxIncl", appdef.DataKind_float32, args{appdef.ConstraintKind_MaxIncl}, true},
		{"float32: MaxExcl", appdef.DataKind_float32, args{appdef.ConstraintKind_MaxExcl}, true},
		{"float32: Enum", appdef.DataKind_float32, args{appdef.ConstraintKind_Enum}, true},
		//-
		{"float64: appdef.MinLen", appdef.DataKind_float64, args{appdef.ConstraintKind_MinLen}, false},
		{"float64: appdef.MaxLen", appdef.DataKind_float64, args{appdef.ConstraintKind_MaxLen}, false},
		{"float64: appdef.Pattern", appdef.DataKind_float64, args{appdef.ConstraintKind_Pattern}, false},
		{"float64: appdef.MinIncl", appdef.DataKind_float64, args{appdef.ConstraintKind_MinIncl}, true},
		{"float64: MinExcl", appdef.DataKind_float64, args{appdef.ConstraintKind_MinExcl}, true},
		{"float64: MaxIncl", appdef.DataKind_float64, args{appdef.ConstraintKind_MaxIncl}, true},
		{"float64: MaxExcl", appdef.DataKind_float64, args{appdef.ConstraintKind_MaxExcl}, true},
		{"float64: Enum", appdef.DataKind_float64, args{appdef.ConstraintKind_Enum}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.IsCompatibleWithConstraint(tt.args.c); got != tt.want {
				t.Errorf("%v.IsCompatibleWithConstraint(%v) = %v, want %v", tt.k.TrimString(), tt.args.c.TrimString(), got, tt.want)
			}
		})
	}
}
