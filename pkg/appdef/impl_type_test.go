/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils/utils"
)

func Test_NullType(t *testing.T) {
	require := require.New(t)

	require.Empty(appdef.NullType.Comment())
	require.Empty(slices.Collect(appdef.NullType.CommentLines()))

	require.False(appdef.NullType.HasTag(appdef.NullQName))
	appdef.NullType.Tags()(func(appdef.ITag) bool { require.Fail("Tags() should be empty"); return false })

	require.Nil(appdef.NullType.App())
	require.Nil(appdef.NullType.Workspace())
	require.Equal(appdef.NullQName, appdef.NullType.QName())
	require.Equal(appdef.TypeKind_null, appdef.NullType.Kind())
	require.False(appdef.NullType.IsSystem())

	require.Contains(fmt.Sprint(appdef.NullType), "null type")
}

func Test_AnyType(t *testing.T) {
	require := require.New(t)

	require.Empty(appdef.AnyType.Comment())
	require.Empty(slices.Collect(appdef.AnyType.CommentLines()))

	require.Nil(appdef.AnyType.App())
	require.Nil(appdef.AnyType.Workspace())
	require.Equal(appdef.QNameANY, appdef.AnyType.QName())
	require.Equal(appdef.TypeKind_Any, appdef.AnyType.Kind())
	require.True(appdef.AnyType.IsSystem())

	require.Contains(fmt.Sprint(appdef.AnyType), "ANY type")
}

func Test_TypeKind_Records(t *testing.T) {
	require := require.New(t)

	require.True(appdef.TypeKind_Records.ContainsAll(appdef.TypeKind_Docs.AsArray()...), "should contain all docs")

	var tests = []struct {
		name string
		k    appdef.TypeKind
		want bool
	}{
		{"GDoc", appdef.TypeKind_CDoc, true},
		{"GRecord", appdef.TypeKind_CRecord, true},
		{"CDoc", appdef.TypeKind_CDoc, true},
		{"CRecord", appdef.TypeKind_CRecord, true},
		{"ODoc", appdef.TypeKind_ODoc, true},
		{"ORecord", appdef.TypeKind_ORecord, true},
		{"WDoc", appdef.TypeKind_WDoc, true},
		{"WRecord", appdef.TypeKind_WRecord, true},

		{"Any", appdef.TypeKind_Any, false},
		{"null", appdef.TypeKind_null, false},
		{"count", appdef.TypeKind_count, false},
		{"view", appdef.TypeKind_ViewRecord, false},
		{"command", appdef.TypeKind_Command, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, appdef.TypeKind_Records.Contains(tt.k))
		})
	}

	require.Panics(func() {
		appdef.TypeKind_Records.Set(appdef.TypeKind_ViewRecord)
	}, "should be read-only")
}

func Test_TypeKind_Docs(t *testing.T) {
	require := require.New(t)

	require.True(appdef.TypeKind_Docs.ContainsAll(appdef.TypeKind_GDoc, appdef.TypeKind_CDoc, appdef.TypeKind_ODoc, appdef.TypeKind_WDoc), "should contain all docs")

	var tests = []struct {
		name string
		k    appdef.TypeKind
		want bool
	}{
		{"GDoc", appdef.TypeKind_GDoc, true},
		{"CDoc", appdef.TypeKind_CDoc, true},
		{"ODoc", appdef.TypeKind_ODoc, true},
		{"WDoc", appdef.TypeKind_WDoc, true},

		{"Any", appdef.TypeKind_Any, false},
		{"null", appdef.TypeKind_null, false},
		{"count", appdef.TypeKind_count, false},
		{"view", appdef.TypeKind_ViewRecord, false},
		{"command", appdef.TypeKind_Command, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, appdef.TypeKind_Docs.Contains(tt.k))
		})
	}

	require.Panics(func() {
		appdef.TypeKind_Docs.Set(appdef.TypeKind_ViewRecord)
	}, "should be read-only")
}

func Test_TypeKind_Structures(t *testing.T) {
	require := require.New(t)

	require.True(appdef.TypeKind_Structures.ContainsAll(appdef.TypeKind_Records.AsArray()...), "should contain all records")

	var tests = []struct {
		name string
		k    appdef.TypeKind
		want bool
	}{
		{"Object", appdef.TypeKind_Object, true},

		{"Any", appdef.TypeKind_Any, false},
		{"null", appdef.TypeKind_null, false},
		{"count", appdef.TypeKind_count, false},
		{"view", appdef.TypeKind_ViewRecord, false},
		{"command", appdef.TypeKind_Command, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, appdef.TypeKind_Structures.Contains(tt.k))
		})
	}

	require.Panics(func() {
		appdef.TypeKind_Structures.Set(appdef.TypeKind_ViewRecord)
	}, "should be read-only")
}

func Test_TypeKind_Singletons(t *testing.T) {
	require := require.New(t)

	var tests = []struct {
		name string
		k    appdef.TypeKind
		want bool
	}{
		{"CDoc", appdef.TypeKind_CDoc, true},
		{"WDoc", appdef.TypeKind_WDoc, true},

		{"Any", appdef.TypeKind_Any, false},
		{"null", appdef.TypeKind_null, false},
		{"ODoc", appdef.TypeKind_ODoc, false},
		{"view", appdef.TypeKind_ViewRecord, false},
		{"command", appdef.TypeKind_Command, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, appdef.TypeKind_Singletons.Contains(tt.k))
		})
	}

	require.Panics(func() {
		appdef.TypeKind_Singletons.Clear(appdef.TypeKind_CDoc)
	}, "should be read-only")
}

func Test_TypeKind_Functions(t *testing.T) {
	require := require.New(t)

	var tests = []struct {
		name string
		k    appdef.TypeKind
		want bool
	}{
		{"Query", appdef.TypeKind_Query, true},
		{"Command", appdef.TypeKind_Command, true},

		{"Any", appdef.TypeKind_Any, false},
		{"null", appdef.TypeKind_null, false},
		{"count", appdef.TypeKind_count, false},
		{"CDoc", appdef.TypeKind_CDoc, false},
		{"Projector", appdef.TypeKind_Projector, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, appdef.TypeKind_Functions.Contains(tt.k))
		})
	}

	require.Panics(func() {
		appdef.TypeKind_Functions.Set(appdef.TypeKind_Job)
	}, "should be read-only")
}

func Test_TypeKind_Extensions(t *testing.T) {
	require := require.New(t)

	var tests = []struct {
		name string
		k    appdef.TypeKind
		want bool
	}{
		{"Query", appdef.TypeKind_Query, true},
		{"Command", appdef.TypeKind_Command, true},
		{"Projector", appdef.TypeKind_Projector, true},
		{"Job", appdef.TypeKind_Job, true},

		{"Any", appdef.TypeKind_Any, false},
		{"null", appdef.TypeKind_null, false},
		{"count", appdef.TypeKind_count, false},
		{"CDoc", appdef.TypeKind_CDoc, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, appdef.TypeKind_Extensions.Contains(tt.k))
		})
	}

	require.Panics(func() {
		appdef.TypeKind_Extensions.ClearAll()
	}, "should be read-only")
}

func Test_TypeKind_Limitables(t *testing.T) {
	require := require.New(t)

	var tests = []struct {
		name string
		k    appdef.TypeKind
		want bool
	}{
		{"Query", appdef.TypeKind_Query, true},
		{"Command", appdef.TypeKind_Command, true},
		{"CDoc", appdef.TypeKind_CDoc, true},
		{"ORecord", appdef.TypeKind_ORecord, true},

		{"Any", appdef.TypeKind_Any, false},
		{"null", appdef.TypeKind_null, false},
		{"count", appdef.TypeKind_count, false},
		{"Job", appdef.TypeKind_Job, false},
		{"Object", appdef.TypeKind_Object, false},
		{"Role", appdef.TypeKind_Role, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, appdef.TypeKind_Limitables.Contains(tt.k))
		})
	}

	require.Panics(func() {
		appdef.TypeKind_Limitables.ClearAll()
	}, "should be read-only")
}

func TestTypeKind_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		k    appdef.TypeKind
		want string
	}{
		{name: `0 —> "appdef.TypeKind_null"`,
			k:    appdef.TypeKind_null,
			want: `TypeKind_null`,
		},
		{name: `2 —> "TypeKind_Data"`,
			k:    appdef.TypeKind_Data,
			want: `TypeKind_Data`,
		},
		{name: `3 —> "appdef.TypeKind_GDoc"`,
			k:    appdef.TypeKind_GDoc,
			want: `TypeKind_GDoc`,
		},
		{name: `TypeKind_FakeLast —> <number>`,
			k:    appdef.TypeKind_count,
			want: utils.UintToString(appdef.TypeKind_count),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalText()
			if err != nil {
				t.Errorf("appdef.TypeKind.MarshalText() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("appdef.TypeKind.MarshalText() = %s, want %v", got, tt.want)
			}
		})
	}

	t.Run("100% cover appdef.TypeKind.String()", func(t *testing.T) {
		const tested = appdef.TypeKind_count + 1
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
		k    appdef.TypeKind
		want string
	}{
		{name: "null", k: appdef.TypeKind_null, want: "null"},
		{name: "basic", k: appdef.TypeKind_CDoc, want: "CDoc"},
		{name: "out of range", k: appdef.TypeKind_count + 1, want: (appdef.TypeKind_count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(appdef.TypeKind).TrimString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}
