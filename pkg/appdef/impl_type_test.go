/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils/utils"
)

type testedTypes interface {
	Type(QName) IType
	Types(func(IType) bool)
}

func Test_NullType(t *testing.T) {
	require := require.New(t)

	require.Empty(NullType.Comment())
	require.Empty(NullType.CommentLines())

	require.Nil(NullType.App())
	require.Nil(NullType.Workspace())
	require.Equal(NullQName, NullType.QName())
	require.Equal(TypeKind_null, NullType.Kind())
	require.False(NullType.IsSystem())

	require.Contains(fmt.Sprint(NullType), "null type")
}

func Test_AnyTypes(t *testing.T) {
	require := require.New(t)

	require.Empty(AnyType.Comment())
	require.Empty(AnyType.CommentLines())

	require.Nil(AnyType.App())
	require.Nil(AnyType.Workspace())
	require.Equal(QNameANY, AnyType.QName())
	require.Equal(TypeKind_Any, AnyType.Kind())
	require.True(AnyType.IsSystem())

	require.Contains(fmt.Sprint(AnyType), "ANY type")

	for n, t := range anyTypes {
		require.Empty(t.Comment())
		require.Nil(t.App())
		require.Equal(n, t.QName())
		require.Equal(TypeKind_Any, t.Kind())
		require.True(t.IsSystem())
	}
}

func Test_TypeKind_Records(t *testing.T) {
	require := require.New(t)

	require.True(TypeKind_Records.ContainsAll(TypeKind_Docs.AsArray()...), "should contain all docs")

	var tests = []struct {
		name string
		k    TypeKind
		want bool
	}{
		{"GDoc", TypeKind_CDoc, true},
		{"GRecord", TypeKind_CRecord, true},
		{"CDoc", TypeKind_CDoc, true},
		{"CRecord", TypeKind_CRecord, true},
		{"ODoc", TypeKind_ODoc, true},
		{"ORecord", TypeKind_ORecord, true},
		{"WDoc", TypeKind_WDoc, true},
		{"WRecord", TypeKind_WRecord, true},

		{"Any", TypeKind_Any, false},
		{"null", TypeKind_null, false},
		{"count", TypeKind_count, false},
		{"view", TypeKind_ViewRecord, false},
		{"command", TypeKind_Command, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, TypeKind_Records.Contains(tt.k))
		})
	}

	require.Panics(func() {
		TypeKind_Records.Set(TypeKind_ViewRecord)
	}, "should be read-only")
}

func Test_TypeKind_Docs(t *testing.T) {
	require := require.New(t)

	require.True(TypeKind_Docs.ContainsAll(TypeKind_GDoc, TypeKind_CDoc, TypeKind_ODoc, TypeKind_WDoc), "should contain all docs")

	var tests = []struct {
		name string
		k    TypeKind
		want bool
	}{
		{"GDoc", TypeKind_GDoc, true},
		{"CDoc", TypeKind_CDoc, true},
		{"ODoc", TypeKind_ODoc, true},
		{"WDoc", TypeKind_WDoc, true},

		{"Any", TypeKind_Any, false},
		{"null", TypeKind_null, false},
		{"count", TypeKind_count, false},
		{"view", TypeKind_ViewRecord, false},
		{"command", TypeKind_Command, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, TypeKind_Docs.Contains(tt.k))
		})
	}

	require.Panics(func() {
		TypeKind_Docs.Set(TypeKind_ViewRecord)
	}, "should be read-only")
}

func Test_TypeKind_Structures(t *testing.T) {
	require := require.New(t)

	require.True(TypeKind_Structures.ContainsAll(TypeKind_Records.AsArray()...), "should contain all records")

	var tests = []struct {
		name string
		k    TypeKind
		want bool
	}{
		{"Object", TypeKind_Object, true},

		{"Any", TypeKind_Any, false},
		{"null", TypeKind_null, false},
		{"count", TypeKind_count, false},
		{"view", TypeKind_ViewRecord, false},
		{"command", TypeKind_Command, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, TypeKind_Structures.Contains(tt.k))
		})
	}

	require.Panics(func() {
		TypeKind_Structures.Set(TypeKind_ViewRecord)
	}, "should be read-only")
}

func Test_TypeKind_Singletons(t *testing.T) {
	require := require.New(t)

	var tests = []struct {
		name string
		k    TypeKind
		want bool
	}{
		{"CDoc", TypeKind_CDoc, true},
		{"WDoc", TypeKind_WDoc, true},

		{"Any", TypeKind_Any, false},
		{"null", TypeKind_null, false},
		{"ODoc", TypeKind_ODoc, false},
		{"view", TypeKind_ViewRecord, false},
		{"command", TypeKind_Command, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, TypeKind_Singletons.Contains(tt.k))
		})
	}

	require.Panics(func() {
		TypeKind_Singletons.Clear(TypeKind_CDoc)
	}, "should be read-only")
}

func Test_TypeKind_Functions(t *testing.T) {
	require := require.New(t)

	var tests = []struct {
		name string
		k    TypeKind
		want bool
	}{
		{"Query", TypeKind_Query, true},
		{"Command", TypeKind_Command, true},

		{"Any", TypeKind_Any, false},
		{"null", TypeKind_null, false},
		{"count", TypeKind_count, false},
		{"CDoc", TypeKind_CDoc, false},
		{"Projector", TypeKind_Projector, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, TypeKind_Functions.Contains(tt.k))
		})
	}

	require.Panics(func() {
		TypeKind_Functions.Set(TypeKind_Job)
	}, "should be read-only")
}

func Test_TypeKind_Extensions(t *testing.T) {
	require := require.New(t)

	var tests = []struct {
		name string
		k    TypeKind
		want bool
	}{
		{"Query", TypeKind_Query, true},
		{"Command", TypeKind_Command, true},
		{"Projector", TypeKind_Projector, true},
		{"Job", TypeKind_Job, true},

		{"Any", TypeKind_Any, false},
		{"null", TypeKind_null, false},
		{"count", TypeKind_count, false},
		{"CDoc", TypeKind_CDoc, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, TypeKind_Extensions.Contains(tt.k))
		})
	}

	require.Panics(func() {
		TypeKind_Extensions.ClearAll()
	}, "should be read-only")
}

func TestTypeKind_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		k    TypeKind
		want string
	}{
		{name: `0 —> "TypeKind_null"`,
			k:    TypeKind_null,
			want: `TypeKind_null`,
		},
		{name: `1 —> "TypeKind_Data"`,
			k:    TypeKind_Data,
			want: `TypeKind_Data`,
		},
		{name: `2 —> "TypeKind_GDoc"`,
			k:    TypeKind_GDoc,
			want: `TypeKind_GDoc`,
		},
		{name: `TypeKind_FakeLast —> <number>`,
			k:    TypeKind_count,
			want: utils.UintToString(TypeKind_count),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalText()
			if err != nil {
				t.Errorf("TypeKind.MarshalText() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("TypeKind.MarshalText() = %s, want %v", got, tt.want)
			}
		})
	}

	t.Run("100% cover TypeKind.String()", func(t *testing.T) {
		const tested = TypeKind_count + 1
		want := "TypeKind(" + utils.UintToString(tested) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(TypeKind_FakeLast + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestTypeKindTrimString(t *testing.T) {
	tests := []struct {
		name string
		k    TypeKind
		want string
	}{
		{name: "null", k: TypeKind_null, want: "null"},
		{name: "basic", k: TypeKind_CDoc, want: "CDoc"},
		{name: "out of range", k: TypeKind_count + 1, want: (TypeKind_count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(TypeKind).TrimString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}

type mockType struct {
	IType
	kind TypeKind
	name QName
}

func (m mockType) Kind() TypeKind { return m.kind }
func (m mockType) QName() QName   { return m.name }
