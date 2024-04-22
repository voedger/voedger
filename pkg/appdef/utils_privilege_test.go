/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrivilegeKindsFrom(t *testing.T) {
	tests := []struct {
		name string
		kk   []PrivilegeKind
		want PrivilegeKinds
	}{
		{"empty", []PrivilegeKind{}, PrivilegeKinds{}},
		{"basic", []PrivilegeKind{PrivilegeKind_Insert, PrivilegeKind_Update}, PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update}},
		{"remove dupes", []PrivilegeKind{PrivilegeKind_Insert, PrivilegeKind_Insert}, PrivilegeKinds{PrivilegeKind_Insert}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PrivilegeKindsFrom(tt.kk...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PrivilegeKindsFrom() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("test panics", func(t *testing.T) {
		tests := []struct {
			name string
			kk   []PrivilegeKind
			want error
		}{
			{"null", []PrivilegeKind{PrivilegeKind_null}, ErrInvalidPrivilegeKind},
			{"out of bounds", []PrivilegeKind{PrivilegeKind_count}, ErrInvalidPrivilegeKind},
		}
		require := require.New(t)
		for _, tt := range tests {
			require.Panics(func() { _ = PrivilegeKindsFrom(tt.kk...) }, tt.name)
		}
	})
}

func TestPrivilegeKinds_Contains(t *testing.T) {
	tests := []struct {
		name string
		pk   PrivilegeKinds
		k    PrivilegeKind
		want bool
	}{
		{"empty kinds", PrivilegeKinds{}, PrivilegeKind_Insert, false},
		{"basic contains", PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update}, PrivilegeKind_Insert, true},
		{"basic not contains", PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update}, PrivilegeKind_Select, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pk.Contains(tt.k); got != tt.want {
				t.Errorf("PrivilegeKinds(%v).Contains(%v) = %v, want %v", tt.pk, tt.k, got, tt.want)
			}
		})
	}
}

func TestPrivilegeKinds_ContainsAll(t *testing.T) {
	tests := []struct {
		name string
		pk   PrivilegeKinds
		kk   []PrivilegeKind
		want bool
	}{
		{"empty kinds", PrivilegeKinds{}, []PrivilegeKind{PrivilegeKind_Insert}, false},
		{"empty args", PrivilegeKinds{}, []PrivilegeKind{}, true},
		{"basic contains", PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update}, []PrivilegeKind{PrivilegeKind_Insert, PrivilegeKind_Update}, true},
		{"basic not contains", PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Select}, []PrivilegeKind{PrivilegeKind_Insert, PrivilegeKind_Update}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pk.ContainsAll(tt.kk...); got != tt.want {
				t.Errorf("PrivilegeKinds(%v).ContainsAll(%v) = %v, want %v", tt.pk, tt.kk, got, tt.want)
			}
		})
	}
}

func TestPrivilegeKinds_ContainsAny(t *testing.T) {
	tests := []struct {
		name string
		pk   PrivilegeKinds
		kk   []PrivilegeKind
		want bool
	}{
		{"empty kinds", PrivilegeKinds{}, []PrivilegeKind{PrivilegeKind_Insert}, false},
		{"empty args", PrivilegeKinds{}, []PrivilegeKind{}, true},
		{"basic contains", PrivilegeKinds{PrivilegeKind_Insert}, []PrivilegeKind{PrivilegeKind_Insert, PrivilegeKind_Update}, true},
		{"basic not contains", PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Select}, []PrivilegeKind{PrivilegeKind_Update}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pk.ContainsAny(tt.kk...); got != tt.want {
				t.Errorf("PrivilegeKinds(%v).ContainsAny(%v) = %v, want %v", tt.pk, tt.kk, got, tt.want)
			}
		})
	}
}
