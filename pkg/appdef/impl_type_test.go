/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/utils/utils"
)

func Test_NullType(t *testing.T) {
	require := require.New(t)

	require.Empty(NullType.Comment())
	require.Empty(NullType.CommentLines())

	require.Nil(NullType.App())
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
