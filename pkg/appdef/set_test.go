/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetFrom(t *testing.T) {
	tests := []struct {
		name string
		set  Set[TypeKind]
		want string
	}{
		{"empty", SetFrom[TypeKind](), "[]"},
		{"one", SetFrom(TypeKind_Any), "[Any]"},
		{"two", SetFrom(TypeKind_Any, TypeKind_Data), "[Any Data]"},
		{"three", SetFrom(TypeKind_Any, TypeKind_Data, TypeKind_Role), "[Any Data Role]"},
		{"should shrink duplicates", SetFrom(TypeKind_Command, TypeKind_Command), "[Command]"},
		{"should accept out of bounds", SetFrom(TypeKind_count + 1), fmt.Sprintf("[%v]", TypeKind_count+1)},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, tt.set.String(), "SetFrom(%v).String() = %v, want %v", tt.set, tt.set.String(), tt.want)
		})
	}
}

func TestSet_AsArray(t *testing.T) {
	tests := []struct {
		name string
		set  Set[TypeKind]
		want []TypeKind
	}{
		{"empty", SetFrom[TypeKind](), nil},
		{"one", SetFrom(TypeKind_CDoc), []TypeKind{TypeKind_CDoc}},
		{"two", SetFrom(TypeKind_CDoc, TypeKind_ODoc), []TypeKind{TypeKind_CDoc, TypeKind_ODoc}},
		{"out of bounds", SetFrom(TypeKind_CDoc, TypeKind_count+1), []TypeKind{TypeKind_CDoc, TypeKind_count + 1}},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.set.AsArray()
			require.EqualValues(got, tt.want, "SetFrom(%v).AsArray() = %v, want %v", tt.set, got, tt.want)
		})
	}
}

func TestSet_Clear(t *testing.T) {
	require := require.New(t)

	t.Run("should be ok to clear one value", func(t *testing.T) {
		set := SetFrom(TypeKind_CDoc, TypeKind_ODoc)
		set.Clear(TypeKind_CDoc)
		require.Equal("[ODoc]", set.String())
		require.Equal(1, set.Len())
		require.EqualValues([]TypeKind{TypeKind_ODoc}, set.AsArray())
	})

	t.Run("should be ok to clear a few values", func(t *testing.T) {
		set := SetFrom(TypeKind_CDoc, TypeKind_ODoc, TypeKind_Command)
		set.Clear(TypeKind_CDoc, TypeKind_ODoc)
		require.Equal("[Command]", set.String())
	})

	t.Run("should be safe to clear already cleared values", func(t *testing.T) {
		set := Set[TypeKind]{}
		set.Clear(TypeKind_CDoc, TypeKind_ODoc)
		require.Equal("[]", set.String())
	})
}

func TestSet_ClearAll(t *testing.T) {
	set := SetFrom(TypeKind_CDoc, TypeKind_ODoc)
	set.ClearAll()
	require.Equal(t, "[]", set.String())
	require.Zero(t, set.Len())
	require.Empty(t, set.AsArray())
}

func TestSet_Contains(t *testing.T) {
	tests := []struct {
		name string
		set  Set[TypeKind]
		v    TypeKind
		want bool
	}{
		{"empty", Set[TypeKind]{}, TypeKind_CDoc, false},
		{"one", SetFrom(TypeKind_CDoc), TypeKind_CDoc, true},
		{"two", SetFrom(TypeKind_CDoc, TypeKind_ODoc), TypeKind_ODoc, true},
		{"negative", SetFrom(TypeKind_CDoc, TypeKind_ODoc), TypeKind_Command, false},
		{"out of bounds", SetFrom(TypeKind_CDoc, TypeKind_ODoc, TypeKind_count+1), TypeKind_count + 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.set.Contains(tt.v); got != tt.want {
				t.Errorf("Set(%v).Contains(%v) = %v, want %v", tt.set, tt.v, got, tt.want)
			}
		})
	}
}

func TestSet_ContainsAll(t *testing.T) {
	tests := []struct {
		name   string
		set    Set[TypeKind]
		values []TypeKind
		want   bool
	}{
		{"nil in empty", Set[TypeKind]{}, nil, true},
		{"empty in empty", Set[TypeKind]{}, []TypeKind{}, true},
		{"cdoc in empty", Set[TypeKind]{}, []TypeKind{TypeKind_CDoc}, false},
		{"cdoc in cdoc", SetFrom(TypeKind_CDoc), []TypeKind{TypeKind_CDoc}, true},
		{"cdoc + odoc in cdoc", SetFrom(TypeKind_CDoc), []TypeKind{TypeKind_CDoc, TypeKind_ODoc}, false},
		{"cdoc + odoc in cdoc + odoc", SetFrom(TypeKind_CDoc, TypeKind_ODoc), []TypeKind{TypeKind_CDoc, TypeKind_ODoc}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.set.ContainsAll(tt.values...); got != tt.want {
				t.Errorf("Set(%v).ContainsAll(%v) = %v, want %v", tt.set, tt.values, got, tt.want)
			}
		})
	}
}

func TestSet_ContainsAny(t *testing.T) {
	tests := []struct {
		name   string
		set    Set[TypeKind]
		values []TypeKind
		want   bool
	}{
		{"nil in empty", Set[TypeKind]{}, nil, true},
		{"empty in empty", Set[TypeKind]{}, []TypeKind{}, true},
		{"cdoc in empty", Set[TypeKind]{}, []TypeKind{TypeKind_CDoc}, false},
		{"cdoc in cdoc", SetFrom(TypeKind_CDoc), []TypeKind{TypeKind_CDoc}, true},
		{"cdoc + odoc in cdoc", SetFrom(TypeKind_CDoc), []TypeKind{TypeKind_CDoc, TypeKind_ODoc}, true},
		{"cdoc + odoc in wdoc + gdoc", SetFrom(TypeKind_WDoc, TypeKind_GDoc), []TypeKind{TypeKind_CDoc, TypeKind_ODoc}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.set.ContainsAny(tt.values...); got != tt.want {
				t.Errorf("Set(%v).ContainsAny(%v) = %v, want %v", tt.set, tt.values, got, tt.want)
			}
		})
	}
}

func TestSet_First(t *testing.T) {
	tests := []struct {
		name      string
		set       Set[TypeKind]
		want      bool
		wantValue TypeKind
	}{
		{"empty", Set[TypeKind]{}, false, TypeKind_null},
		{"one", SetFrom(TypeKind_Workspace), true, TypeKind_Workspace},
		{"two", SetFrom(TypeKind_Workspace, TypeKind_GDoc), true, TypeKind_GDoc},
		{"out of bounds", SetFrom(TypeKind_count + 1), true, TypeKind_count + 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotV := tt.set.First()
			if got != tt.want {
				t.Errorf("Set(%v).First() got = %v, want %v", tt.set, got, tt.want)
			}
			if !reflect.DeepEqual(gotV, tt.wantValue) {
				t.Errorf("Set(%v).First() gotV = %v, want %v", tt.set, gotV, tt.wantValue)
			}
		})
	}
}

func TestSet_Len(t *testing.T) {
	tests := []struct {
		name string
		set  Set[TypeKind]
		want int
	}{
		{"empty", Set[TypeKind]{}, 0},
		{"one", SetFrom(TypeKind_CDoc), 1},
		{"two", SetFrom(TypeKind_CDoc, TypeKind_Any), 2},
		{"two + out of bounds", SetFrom(TypeKind_CDoc, TypeKind_Role, TypeKind_count+1), 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.set.Len(); got != tt.want {
				t.Errorf("Set(%v).Len() = %v, want %v", tt.set, got, tt.want)
			}
		})
	}
}

func TestSet_SetRange(t *testing.T) {
	type args struct {
		start TypeKind
		end   TypeKind
	}
	tests := []struct {
		name string
		set  Set[TypeKind]
		args args
		want string
	}{
		{"empty", Set[TypeKind]{}, args{TypeKind_CDoc, TypeKind_CDoc}, "[]"},
		{"one", Set[TypeKind]{}, args{TypeKind_CDoc, TypeKind_CDoc + 1}, "[CDoc]"},
		{"two", Set[TypeKind]{}, args{TypeKind_CDoc, TypeKind_CDoc + 2}, "[CDoc ODoc]"},
		{"three", Set[TypeKind]{}, args{TypeKind_CDoc, TypeKind_CDoc + 3}, "[CDoc ODoc WDoc]"},
		{"one + range", SetFrom(TypeKind_null), args{TypeKind_CDoc, TypeKind_CDoc + 3}, "[null CDoc ODoc WDoc]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.set.SetRange(tt.args.start, tt.args.end)
			if got := tt.set.String(); got != tt.want {
				t.Errorf("Set.SetRange(%v, %v).String() = %v, want %v", tt.args.start, tt.args.end, got, tt.want)
			}
		})
	}
}

func TestSet_String(t *testing.T) {
	tests := []struct {
		name string
		set  Set[TypeKind]
		want string
	}{
		{"empty", Set[TypeKind]{}, "[]"},
		{"one", SetFrom(TypeKind_CDoc), "[CDoc]"},
		{"two", SetFrom(TypeKind_CDoc, TypeKind_Limit), "[CDoc Limit]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.set.String(); got != tt.want {
				t.Errorf("Set(%v).String() = %v, want %v", tt.set, got, tt.want)
			}
		})
	}
}
