/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_def_AddContainer(t *testing.T) {
	require := require.New(t)

	appDef := New()
	def := appDef.AddObject(NewQName("test", "object"))
	require.NotNil(def)

	elQName := NewQName("test", "element")
	_ = appDef.AddElement(elQName)

	t.Run("must be ok to add container", func(t *testing.T) {
		def.AddContainer("c1", elQName, 1, Occurs_Unbounded)

		require.Equal(1, def.ContainerCount())
		c := def.Container("c1")
		require.NotNil(c)

		require.Equal("c1", c.Name())
		require.False(c.IsSys())

		require.Equal(elQName, c.Def())
		d := def.ContainerDef("c1")
		require.NotNil(d)
		require.Equal(elQName, d.QName())
		require.Equal(DefKind_Element, d.Kind())

		require.EqualValues(1, c.MinOccurs())
		require.Equal(Occurs_Unbounded, c.MaxOccurs())
	})

	t.Run("test AddContainer(…).QName() helper", func(t *testing.T) {
		d := appDef.AddObject(NewQName("test", "object1"))
		require.Equal(d.QName(), d.AddContainer("c1", elQName, 1, Occurs_Unbounded).QName())
	})

	t.Run("must be panic if empty container name", func(t *testing.T) {
		require.Panics(func() { def.AddContainer("", elQName, 1, Occurs_Unbounded) })
	})

	t.Run("must be panic if invalid container name", func(t *testing.T) {
		require.Panics(func() { def.AddContainer("naked_🔫", elQName, 1, Occurs_Unbounded) })
		t.Run("but ok if system container", func(t *testing.T) {
			require.NotPanics(func() { def.AddContainer(SystemContainer_ViewValue, elQName, 1, Occurs_Unbounded) })
			require.Equal(2, def.ContainerCount())
		})
	})

	t.Run("must be panic if container name dupe", func(t *testing.T) {
		require.Panics(func() { def.AddContainer("c1", elQName, 1, Occurs_Unbounded) })
	})

	t.Run("must be panic if invalid occurrences", func(t *testing.T) {
		require.Panics(func() { def.AddContainer("c2", elQName, 1, 0) })
		require.Panics(func() { def.AddContainer("c3", elQName, 2, 1) })
	})

	t.Run("must be panic if containers are not allowed by definition kind", func(t *testing.T) {
		view := appDef.AddView(NewQName("test", "view"))
		pk := view.PartKey()
		require.Panics(func() { pk.(IContainersBuilder).AddContainer("c1", elQName, 1, Occurs_Unbounded) })
	})

	t.Run("must be panic if container definition is not compatible", func(t *testing.T) {
		require.Panics(func() { def.AddContainer("c2", def.QName(), 1, 1) })

		d := def.ContainerDef("c2")
		require.NotNil(d)
		require.Equal(DefKind_null, d.Kind())
	})

	t.Run("must be panic if too many containers", func(t *testing.T) {
		qn := NewQName("test", "el")
		d := New().AddElement(qn)
		for i := 0; i < MaxDefContainerCount; i++ {
			d.AddContainer(fmt.Sprintf("c_%#x", i), qn, 0, Occurs_Unbounded)
		}
		require.Panics(func() { d.AddContainer("errorContainer", qn, 0, Occurs_Unbounded) })
	})
}

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
