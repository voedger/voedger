/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_def_AddField(t *testing.T) {
	require := require.New(t)

	def := New().AddStruct(NewQName("test", "object"), DefKind_Object)
	require.NotNil(def)

	t.Run("must be ok to add field", func(t *testing.T) {
		def.AddField("f1", DataKind_int64, true)

		require.Equal(2, def.FieldCount()) // + sys.QName
		f := def.Field("f1")
		require.NotNil(f)
		require.Equal("f1", f.Name())
		require.False(f.IsSys())

		require.Equal(DataKind_int64, f.DataKind())
		require.True(f.IsFixedWidth())
		require.True(f.DataKind().IsFixed())

		require.True(f.Required())
		require.False(f.Verifiable())
	})

	t.Run("must be panic if empty field name", func(t *testing.T) {
		require.Panics(func() { def.AddField("", DataKind_int64, true) })
	})

	t.Run("must be panic if invalid field name", func(t *testing.T) {
		require.Panics(func() { def.AddField("naked_🔫", DataKind_int64, true) })
		t.Run("but ok if system field", func(t *testing.T) {
			require.NotPanics(func() { def.AddField(SystemField_QName, DataKind_QName, true) })
			require.Equal(2, def.FieldCount())
		})
	})

	t.Run("must be panic if field name dupe", func(t *testing.T) {
		require.Panics(func() { def.AddField("f1", DataKind_int64, true) })
		t.Run("but ok if system field", func(t *testing.T) {
			require.NotPanics(func() { def.AddField(SystemField_QName, DataKind_QName, true) })
			require.Equal(2, def.FieldCount())
		})
	})

	t.Run("must be panic if fields are not allowed by definition kind", func(t *testing.T) {
		view := New().AddView(NewQName("test", "view"))
		def := view.Def()
		require.Panics(func() { def.AddField("f1", DataKind_string, true) })
	})

	t.Run("must be panic if field data kind is not allowed by definition kind", func(t *testing.T) {
		view := New().AddView(NewQName("test", "view"))
		require.Panics(func() { view.AddPartField("f1", DataKind_string) })
	})
}

func Test_def_AddVerifiedField(t *testing.T) {
	require := require.New(t)

	def := New().AddStruct(NewQName("test", "object"), DefKind_Object)
	require.NotNil(def)

	t.Run("must be ok to add verified field", func(t *testing.T) {
		def.AddVerifiedField("f1", DataKind_int64, true, VerificationKind_Phone)
		def.AddVerifiedField("f2", DataKind_int64, true, VerificationKind_Any...)

		require.Equal(3, def.FieldCount()) // + sys.QName
		f1 := def.Field("f1")
		require.NotNil(f1)

		require.True(f1.Verifiable())
		require.False(f1.VerificationKind(VerificationKind_EMail))
		require.True(f1.VerificationKind(VerificationKind_Phone))
		require.False(f1.VerificationKind(VerificationKind_FakeLast))

		f2 := def.Field("f2")
		require.NotNil(f2)

		require.True(f2.Verifiable())
		require.True(f2.VerificationKind(VerificationKind_EMail))
		require.True(f2.VerificationKind(VerificationKind_Phone))
		require.False(f2.VerificationKind(VerificationKind_FakeLast))
	})

	t.Run("must be panic if no verification kinds", func(t *testing.T) {
		require.Panics(func() { def.AddVerifiedField("f2", DataKind_int64, true) })
	})
}

func Test_def_AddContainer(t *testing.T) {
	require := require.New(t)

	appDef := New()
	def := appDef.AddStruct(NewQName("test", "object"), DefKind_Object)
	require.NotNil(def)

	elQName := NewQName("test", "element")
	_ = appDef.AddStruct(elQName, DefKind_Element)

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
		pk := view.PartKeyDef()
		require.Panics(func() { pk.AddContainer("c1", elQName, 1, Occurs_Unbounded) })
	})

	t.Run("must be panic if container definition is not compatable", func(t *testing.T) {
		require.Panics(func() { def.AddContainer("c2", def.QName(), 1, 1) })

		d := def.ContainerDef("c2")
		require.NotNil(d)
		require.Equal(DefKind_null, d.Kind())
	})
}

func Test_def_Singleton(t *testing.T) {
	require := require.New(t)

	appDef := New()

	t.Run("must be ok to create singleton definition", func(t *testing.T) {
		def := appDef.AddStruct(NewQName("test", "singleton"), DefKind_CDoc)
		require.NotNil(def)

		def.SetSingleton()
		require.True(def.Singleton())
	})

	t.Run("must be panic if not CDoc definition", func(t *testing.T) {
		def := appDef.AddStruct(NewQName("test", "wdoc"), DefKind_WDoc)
		require.NotNil(def)

		require.Panics(func() { def.SetSingleton() })

		require.False(def.Singleton())
	})
}
