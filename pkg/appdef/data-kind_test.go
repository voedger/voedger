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
