/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"math/bits"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func _() {
	// An "invalid array index" compiler error signifies that the count of elements in TypeKind_××× enumeration exceeds 64.
	// This means what you need to use another impolementation for TypeKinds set.
	var x [bits.UintSize]any
	_ = x[TypeKind_count]
}

func TestTypeKindsFrom(t *testing.T) {
	tests := []struct {
		name  string
		kinds []TypeKind
		want  string
	}{
		{"empty", []TypeKind{}, "[]"},
		{"one", []TypeKind{TypeKind_Any}, "[Any]"},
		{"two", []TypeKind{TypeKind_Any, TypeKind_Data}, "[Any Data]"},
		{"three", []TypeKind{TypeKind_Any, TypeKind_Data, TypeKind_Role}, "[Any Data Role]"},
		{"should shrink duplicates", []TypeKind{TypeKind_Command, TypeKind_Command}, "[Command]"},
		{"should accept out of bounds", []TypeKind{TypeKind_count + 1}, fmt.Sprintf("[%v]", TypeKind_count+1)},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTt := TypeKindsFrom(tt.kinds...)
			require.Equal(tt.want, gotTt.String(), "TypeKindsFrom(%v) = %v, want %v", tt.kinds, gotTt, tt.want)
		})
	}
}

func TestTypeKinds_AsArray(t *testing.T) {
	tests := []struct {
		name  string
		kinds TypeKinds
		want  []TypeKind
	}{
		{"empty", TypeKinds{}, []TypeKind{}},
		{"one", TypeKindsFrom(TypeKind_CDoc), []TypeKind{TypeKind_CDoc}},
		{"two", TypeKindsFrom(TypeKind_CDoc, TypeKind_ODoc), []TypeKind{TypeKind_CDoc, TypeKind_ODoc}},
		{"out of bounds", TypeKindsFrom(TypeKind_CDoc, TypeKind_count+1), []TypeKind{TypeKind_CDoc, TypeKind_count + 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.kinds.AsArray(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TypeKinds.AsArray(%v) = %v, want %v", tt.kinds, got, tt.want)
			}
		})
	}
}

func TestTypeKinds_Clear(t *testing.T) {
	require := require.New(t)

	t.Run("should be ok to clear one kind", func(t *testing.T) {
		tt := TypeKindsFrom(TypeKind_CDoc, TypeKind_ODoc)
		tt.Clear(TypeKind_CDoc)
		require.Equal("[ODoc]", tt.String())
	})

	t.Run("should be ok to clear a few kinds", func(t *testing.T) {
		tt := TypeKindsFrom(TypeKind_CDoc, TypeKind_ODoc, TypeKind_Command)
		tt.Clear(TypeKind_CDoc, TypeKind_ODoc)
		require.Equal("[Command]", tt.String())
	})

	t.Run("should be safe to clear already cleared", func(t *testing.T) {
		tt := TypeKinds{}
		tt.Clear(TypeKind_CDoc, TypeKind_ODoc)
		require.Equal("[]", tt.String())
	})
}

func TestTypeKinds_ClearAll(t *testing.T) {
	tt := TypeKindsFrom(TypeKind_CDoc, TypeKind_ODoc)
	tt.ClearAll()
	require.Equal(t, "[]", tt.String())
}

func TestTypeKinds_Contains(t *testing.T) {
	tests := []struct {
		name string
		tt   TypeKinds
		k    TypeKind
		want bool
	}{
		{"empty", TypeKinds{}, TypeKind_CDoc, false},
		{"one", TypeKindsFrom(TypeKind_CDoc), TypeKind_CDoc, true},
		{"two", TypeKindsFrom(TypeKind_CDoc, TypeKind_ODoc), TypeKind_ODoc, true},
		{"negative", TypeKindsFrom(TypeKind_CDoc, TypeKind_ODoc), TypeKind_Command, false},
		{"out of bounds", TypeKindsFrom(TypeKind_CDoc, TypeKind_ODoc, TypeKind_count+1), TypeKind_count + 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tt.Contains(tt.k); got != tt.want {
				t.Errorf("TypeKinds(%v).Contains(%v) = %v, want %v", tt.tt, tt.k, got, tt.want)
			}
		})
	}
}

func TestTypeKinds_ContainsAll(t *testing.T) {
	tests := []struct {
		name string
		tt   TypeKinds
		t    []TypeKind
		want bool
	}{
		{"nil in empty", TypeKinds{}, nil, true},
		{"empty in empty", TypeKinds{}, []TypeKind{}, true},
		{"cdoc in empty", TypeKinds{}, []TypeKind{TypeKind_CDoc}, false},
		{"cdoc in cdoc", TypeKindsFrom(TypeKind_CDoc), []TypeKind{TypeKind_CDoc}, true},
		{"cdoc + odoc in cdoc", TypeKindsFrom(TypeKind_CDoc), []TypeKind{TypeKind_CDoc, TypeKind_ODoc}, false},
		{"cdoc + odoc in cdoc + odoc", TypeKindsFrom(TypeKind_CDoc, TypeKind_ODoc), []TypeKind{TypeKind_CDoc, TypeKind_ODoc}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tt.ContainsAll(tt.t...); got != tt.want {
				t.Errorf("TypeKinds(%v).ContainsAll(%v) = %v, want %v", tt.tt, tt.t, got, tt.want)
			}
		})
	}
}

func TestTypeKinds_ContainsAny(t *testing.T) {
	tests := []struct {
		name string
		tt   TypeKinds
		t    []TypeKind
		want bool
	}{
		{"nil in empty", TypeKinds{}, nil, true},
		{"empty in empty", TypeKinds{}, []TypeKind{}, true},
		{"cdoc in empty", TypeKinds{}, []TypeKind{TypeKind_CDoc}, false},
		{"cdoc in cdoc", TypeKindsFrom(TypeKind_CDoc), []TypeKind{TypeKind_CDoc}, true},
		{"cdoc + odoc in cdoc", TypeKindsFrom(TypeKind_CDoc), []TypeKind{TypeKind_CDoc, TypeKind_ODoc}, true},
		{"cdoc + odoc in wdoc + gdoc", TypeKindsFrom(TypeKind_WDoc, TypeKind_GDoc), []TypeKind{TypeKind_CDoc, TypeKind_ODoc}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tt.ContainsAny(tt.t...); got != tt.want {
				t.Errorf("TypeKinds(%v).ContainsAny(%v) = %v, want %v", tt.tt, tt.t, got, tt.want)
			}
		})
	}
}

func TestTypeKinds_First(t *testing.T) {
	tests := []struct {
		name  string
		tt    TypeKinds
		want  bool
		wantT TypeKind
	}{
		{"empty", TypeKinds{}, false, TypeKind_count},
		{"one", TypeKindsFrom(TypeKind_Workspace), true, TypeKind_Workspace},
		{"two", TypeKindsFrom(TypeKind_Workspace, TypeKind_GDoc), true, TypeKind_GDoc},
		{"out of bounds", TypeKindsFrom(TypeKind_count + 1), true, TypeKind_count + 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.tt.First()
			if got != tt.want {
				t.Errorf("TypeKinds.First() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.wantT) {
				t.Errorf("TypeKinds.First() got1 = %v, want %v", got1, tt.wantT)
			}
		})
	}
}

func TestTypeKinds_Len(t *testing.T) {
	tests := []struct {
		name string
		tt   TypeKinds
		want int
	}{
		{"empty", TypeKinds{}, 0},
		{"one", TypeKindsFrom(TypeKind_CDoc), 1},
		{"two", TypeKindsFrom(TypeKind_CDoc, TypeKind_Any), 2},
		{"two + out of bounds", TypeKindsFrom(TypeKind_CDoc, TypeKind_Role, TypeKind_count+1), 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tt.Len(); got != tt.want {
				t.Errorf("TypeKinds.Len(%v) = %v, want %v", tt.tt, got, tt.want)
			}
		})
	}
}

func TestTypeKinds_SetRange(t *testing.T) {
	type args struct {
		start TypeKind
		end   TypeKind
	}
	tests := []struct {
		name string
		tt   TypeKinds
		args args
		want string
	}{
		{"empty", TypeKinds{}, args{TypeKind_CDoc, TypeKind_CDoc}, "[]"},
		{"one", TypeKinds{}, args{TypeKind_CDoc, TypeKind_CDoc + 1}, "[CDoc]"},
		{"two", TypeKinds{}, args{TypeKind_CDoc, TypeKind_CDoc + 2}, "[CDoc ODoc]"},
		{"three", TypeKinds{}, args{TypeKind_CDoc, TypeKind_CDoc + 3}, "[CDoc ODoc WDoc]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.tt.SetRange(tt.args.start, tt.args.end)
			if got := tt.tt.String(); got != tt.want {
				t.Errorf("TypeKinds.SetRange(%v, %v).String() = %v, want %v", tt.args.start, tt.args.end, got, tt.want)
			}
		})
	}
}
