/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_IsSysContainer(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "true if sys.pkey",
			args: args{SystemContainer_ViewPartitionKey},
			want: true,
		},
		{
			name: "true if sys.ccols",
			args: args{SystemContainer_ViewClusteringCols},
			want: true,
		},
		{
			name: "true if sys.key",
			args: args{SystemContainer_ViewKey},
			want: true,
		},
		{
			name: "true if sys.val",
			args: args{SystemContainer_ViewValue},
			want: true,
		},
		{
			name: "false if empty",
			args: args{""},
			want: false,
		},
		{
			name: "false if basic user",
			args: args{"userContainer"},
			want: false,
		},
		{
			name: "false if curious user",
			args: args{"sys.user"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSysContainer(tt.args.name); got != tt.want {
				t.Errorf("IsSysContainer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_type_AddContainer(t *testing.T) {
	require := require.New(t)

	appDef := New()
	obj := appDef.AddObject(NewQName("test", "object"))
	require.NotNil(obj)

	elQName := NewQName("test", "element")
	_ = appDef.AddElement(elQName)

	t.Run("must be ok to add container", func(t *testing.T) {
		obj.AddContainer("c1", elQName, 1, Occurs_Unbounded)

		require.Equal(1, obj.ContainerCount())
		c := obj.Container("c1")
		require.NotNil(c)

		require.Equal("c1", c.Name())
		require.False(c.IsSys())

		require.Equal(elQName, c.QName())
		typ := c.Type()
		require.NotNil(typ)
		require.Equal(elQName, typ.QName())
		require.Equal(TypeKind_Element, typ.Kind())

		require.EqualValues(1, c.MinOccurs())
		require.Equal(Occurs_Unbounded, c.MaxOccurs())
	})

	t.Run("chain notation is ok to add containers", func(t *testing.T) {
		obj := New().AddObject(NewQName("test", "obj"))
		n := obj.AddContainer("c1", elQName, 1, Occurs_Unbounded).
			AddContainer("c2", elQName, 1, Occurs_Unbounded).
			AddContainer("c3", elQName, 1, Occurs_Unbounded).(IType).QName()
		require.Equal(obj.QName(), n)
		require.Equal(3, obj.ContainerCount())
	})

	t.Run("must be panic if empty container name", func(t *testing.T) {
		require.Panics(func() { obj.AddContainer("", elQName, 1, Occurs_Unbounded) })
	})

	t.Run("must be panic if invalid container name", func(t *testing.T) {
		require.Panics(func() { obj.AddContainer("naked_🔫", elQName, 1, Occurs_Unbounded) })
		t.Run("but ok if system container", func(t *testing.T) {
			require.NotPanics(func() { obj.AddContainer(SystemContainer_ViewValue, elQName, 1, Occurs_Unbounded) })
			require.Equal(2, obj.ContainerCount())
		})
	})

	t.Run("must be panic if container name dupe", func(t *testing.T) {
		require.Panics(func() { obj.AddContainer("c1", elQName, 1, Occurs_Unbounded) })
	})

	t.Run("must be panic if container type name missed", func(t *testing.T) {
		require.Panics(func() { obj.AddContainer("c2", NullQName, 1, Occurs_Unbounded) })
	})

	t.Run("must be panic if invalid occurrences", func(t *testing.T) {
		require.Panics(func() { obj.AddContainer("c2", elQName, 1, 0) })
		require.Panics(func() { obj.AddContainer("c3", elQName, 2, 1) })
	})

	t.Run("must be panic if container type is incompatible", func(t *testing.T) {
		require.Panics(func() { obj.AddContainer("c2", obj.QName(), 1, 1) })
		require.Nil(obj.Container("c2"))
	})

	t.Run("must be panic if too many containers", func(t *testing.T) {
		qn := NewQName("test", "el")
		el := New().AddElement(qn)
		for i := 0; i < MaxTypeContainerCount; i++ {
			el.AddContainer(fmt.Sprintf("c_%#x", i), qn, 0, Occurs_Unbounded)
		}
		require.Panics(func() { el.AddContainer("errorContainer", qn, 0, Occurs_Unbounded) })
	})
}

func TestValidateContainer(t *testing.T) {
	require := require.New(t)

	app := New()
	doc := app.AddCDoc(NewQName("test", "doc"))
	doc.AddContainer("rec", NewQName("test", "rec"), 0, Occurs_Unbounded)

	t.Run("must be error if container type not found", func(t *testing.T) {
		_, err := app.Build()
		require.ErrorIs(err, ErrNameNotFound)
		require.ErrorContains(err, "unknown type «test.rec»")
	})

	rec := app.AddCRecord(NewQName("test", "rec"))
	_, err := app.Build()
	require.NoError(err)

	t.Run("must be ok container recurse", func(t *testing.T) {
		rec.AddContainer("rec", NewQName("test", "rec"), 0, Occurs_Unbounded)
		_, err := app.Build()
		require.NoError(err)
	})

	t.Run("must be ok container sub recurse", func(t *testing.T) {
		rec.AddContainer("rec1", NewQName("test", "rec1"), 0, Occurs_Unbounded)
		rec1 := app.AddCRecord(NewQName("test", "rec1"))
		rec1.AddContainer("rec", NewQName("test", "rec"), 0, Occurs_Unbounded)
		_, err := app.Build()
		require.NoError(err)
	})

	t.Run("must be error if container kind is incompatible", func(t *testing.T) {
		doc.AddContainer("obj", NewQName("test", "obj"), 0, 1)
		_ = app.AddObject(NewQName("test", "obj"))
		_, err := app.Build()
		require.ErrorIs(err, ErrInvalidTypeKind)
		require.ErrorContains(err, "«CDoc» can`t contain «Object»")
	})
}
