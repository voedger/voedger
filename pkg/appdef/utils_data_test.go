/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"reflect"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils/utils"
)

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
