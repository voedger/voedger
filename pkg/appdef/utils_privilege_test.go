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
			{"null", []PrivilegeKind{PrivilegeKind_null}, ErrOutOfBoundsError},
			{"out of bounds", []PrivilegeKind{PrivilegeKind_count}, ErrOutOfBoundsError},
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

func TestAllPrivilegesOnType(t *testing.T) {
	tests := []struct {
		name   string
		tk     TypeKind
		wantPk PrivilegeKinds
	}{
		{"null", TypeKind_null, nil},
		{"GRecord", TypeKind_GRecord, PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select}},
		{"CDoc", TypeKind_CDoc, PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select}},
		{"View", TypeKind_ViewRecord, PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select}},
		{"Command", TypeKind_Command, PrivilegeKinds{PrivilegeKind_Execute}},
		{"Workspace", TypeKind_Workspace, PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select, PrivilegeKind_Execute}},
		{"Role", TypeKind_Role, PrivilegeKinds{PrivilegeKind_Inherits}},
		{"Projector", TypeKind_Projector, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotPk := AllPrivilegesOnType(tt.tk); !reflect.DeepEqual(gotPk, tt.wantPk) {
				t.Errorf("AllPrivilegesOnType(%s) = %v, want %v", tt.tk.TrimString(), gotPk, tt.wantPk)
			}
		})
	}
}

func TestPrivilegeAccessControlString(t *testing.T) {
	tests := []struct {
		name  string
		grant bool
		want  string
	}{
		{"granted", true, "grant"},
		{"revoked", false, "revoke"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PrivilegeAccessControlString(tt.grant); got != tt.want {
				t.Errorf("PrivilegeAccessControlString(%v) = %v, want %v", tt.grant, got, tt.want)
			}
		})
	}
}
