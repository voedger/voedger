/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strconv"
	"testing"
)

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
		{"bytes: Enum", DataKind_bytes, args{ConstraintKind_Enum}, true},
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
				t.Errorf("DataKind.IsSupportedConstraint() = %v, want %v", got, tt.want)
			}
		})
	}
}
