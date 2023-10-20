/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Maxim Geraskin
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strconv"
	"testing"
)

func TestDataConstraintKind_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		k    DataConstraintKind
		want string
	}{
		{
			name: `0 —> "DataConstraintKind_null"`,
			k:    DataConstraintKind_null,
			want: `DataConstraintKind_null`,
		},
		{
			name: `1 —> "DataConstraintKind_MinLen"`,
			k:    DataConstraintKind_MinLen,
			want: `DataConstraintKind_MinLen`,
		},
		{
			name: `DataConstraintKind_Count —> 4`,
			k:    DataConstraintKind_Count,
			want: strconv.FormatUint(uint64(DataConstraintKind_Count), 10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalText()
			if err != nil {
				t.Errorf("DataConstraintKind.MarshalText() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("DataConstraintKind.MarshalText() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("100% cover", func(t *testing.T) {
		const tested = DataConstraintKind_Count + 1
		want := "DataConstraintKind(" + strconv.FormatInt(int64(tested), 10) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(DataConstraintKind_Count + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestDataConstraintKind_TrimString(t *testing.T) {
	tests := []struct {
		name string
		k    DataConstraintKind
		want string
	}{
		{name: "basic", k: DataConstraintKind_MinLen, want: "MinLen"},
		{name: "out of range", k: DataConstraintKind_Count + 1, want: (DataConstraintKind_Count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(%T).TrimString() = %v, want %v", tt.k, tt.k, got, tt.want)
			}
		})
	}
}
