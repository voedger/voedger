/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_schema_AddField(t *testing.T) {
	require := require.New(t)

	bld := NewSchemaCache().Add(NewQName("test", "object"), SchemaKind_Object)
	require.NotNil(bld)

	t.Run("must be ok to add field", func(t *testing.T) {
		bld.AddField("f1", DataKind_int64, true)

		require.Equal(2, bld.FieldCount()) // + sys.QName
		f := bld.Field("f1")
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
		require.Panics(func() { bld.AddField("", DataKind_int64, true) })
	})

	t.Run("must be panic if invalid field name", func(t *testing.T) {
		require.Panics(func() { bld.AddField("naked_🔫", DataKind_int64, true) })
		t.Run("but ok if system field", func(t *testing.T) {
			require.NotPanics(func() { bld.AddField(SystemField_QName, DataKind_QName, true) })
			require.Equal(2, bld.FieldCount())
		})
	})

	t.Run("must be panic if field name dupe", func(t *testing.T) {
		require.Panics(func() { bld.AddField("f1", DataKind_int64, true) })
		t.Run("but ok if system field", func(t *testing.T) {
			require.NotPanics(func() { bld.AddField(SystemField_QName, DataKind_QName, true) })
			require.Equal(2, bld.FieldCount())
		})
	})

	t.Run("must be panic if fields are not allowed by schema kind", func(t *testing.T) {
		bld := NewSchemaCache().Add(NewQName("test", "test"), SchemaKind_ViewRecord)
		require.Panics(func() { bld.AddField("f1", DataKind_int64, true) })
	})

	t.Run("must be panic if field data kind is not allowed by schema kind", func(t *testing.T) {
		bld := NewSchemaCache().Add(NewQName("test", "test"), SchemaKind_ViewRecord_PartitionKey)
		require.Panics(func() { bld.AddField("f1", DataKind_string, true) })
	})
}

func Test_schema_AddVerifiedField(t *testing.T) {
	require := require.New(t)

	bld := NewSchemaCache().Add(NewQName("test", "object"), SchemaKind_Object)
	require.NotNil(bld)

	t.Run("must be ok to add verified field", func(t *testing.T) {
		bld.AddVerifiedField("f1", DataKind_int64, true, VerificationKind_Phone)
		bld.AddVerifiedField("f2", DataKind_int64, true, VerificationKind_Any...)

		require.Equal(3, bld.FieldCount()) // + sys.QName
		f1 := bld.Field("f1")
		require.NotNil(f1)

		require.True(f1.Verifiable())
		require.False(f1.VerificationKind(VerificationKind_EMail))
		require.True(f1.VerificationKind(VerificationKind_Phone))
		require.False(f1.VerificationKind(VerificationKind_FakeLast))

		f2 := bld.Field("f2")
		require.NotNil(f2)

		require.True(f2.Verifiable())
		require.True(f2.VerificationKind(VerificationKind_EMail))
		require.True(f2.VerificationKind(VerificationKind_Phone))
		require.False(f2.VerificationKind(VerificationKind_FakeLast))
	})

	t.Run("must be panic if no verification kinds", func(t *testing.T) {
		require.Panics(func() { bld.AddVerifiedField("f2", DataKind_int64, true) })
	})
}

func Test_schema_AddContainer(t *testing.T) {
	require := require.New(t)

	cache := NewSchemaCache()
	bld := cache.Add(NewQName("test", "object"), SchemaKind_Object)
	require.NotNil(bld)

	elQName := NewQName("test", "element")
	_ = cache.Add(elQName, SchemaKind_Element)

	t.Run("must be ok to add container", func(t *testing.T) {
		bld.AddContainer("c1", elQName, 1, Occurs_Unbounded)

		require.Equal(1, bld.ContainerCount())
		c := bld.Container("c1")
		require.NotNil(c)

		require.Equal("c1", c.Name())
		require.False(c.IsSys())

		require.Equal(elQName, c.Schema())
		s := bld.ContainerSchema("c1")
		require.NotNil(s)
		require.Equal(elQName, s.QName())

		require.EqualValues(1, c.MinOccurs())
		require.Equal(Occurs_Unbounded, c.MaxOccurs())
	})

	t.Run("must be panic if empty container name", func(t *testing.T) {
		require.Panics(func() { bld.AddContainer("", elQName, 1, Occurs_Unbounded) })
	})

	t.Run("must be panic if invalid container name", func(t *testing.T) {
		require.Panics(func() { bld.AddContainer("naked_🔫", elQName, 1, Occurs_Unbounded) })
		t.Run("but ok if system container", func(t *testing.T) {
			require.NotPanics(func() { bld.AddContainer(SystemContainer_ViewValue, elQName, 1, Occurs_Unbounded) })
			require.Equal(2, bld.ContainerCount())
		})
	})

	t.Run("must be panic if container name dupe", func(t *testing.T) {
		require.Panics(func() { bld.AddContainer("c1", elQName, 1, Occurs_Unbounded) })
	})

	t.Run("must be panic if invalid occurrences", func(t *testing.T) {
		require.Panics(func() { bld.AddContainer("c2", elQName, 1, 0) })
		require.Panics(func() { bld.AddContainer("c3", elQName, 2, 1) })
	})

	pkQName := NewQName("test", "pk")

	t.Run("must be panic if containers are not allowed by schema kind", func(t *testing.T) {
		bld := cache.Add(pkQName, SchemaKind_ViewRecord_PartitionKey)
		require.Panics(func() { bld.AddContainer("c1", elQName, 1, Occurs_Unbounded) })
	})

	t.Run("must be panic if container schema is not compatable", func(t *testing.T) {
		require.Panics(func() { bld.AddContainer("c2", pkQName, 1, 1) })

		require.Nil(bld.ContainerSchema("c2"))
	})
}

func Test_schema_Singleton(t *testing.T) {
	require := require.New(t)

	cache := NewSchemaCache()

	t.Run("must be ok to create singleton schema", func(t *testing.T) {
		bld := cache.Add(NewQName("test", "singleton"), SchemaKind_CDoc)
		require.NotNil(bld)

		bld.SetSingleton()
		require.True(bld.Singleton())
	})

	t.Run("must be panic if not CDoc schema", func(t *testing.T) {
		bld := cache.Add(NewQName("test", "wdoc"), SchemaKind_WDoc)
		require.NotNil(bld)

		require.Panics(func() { bld.SetSingleton() })

		require.False(bld.Singleton())
	})
}
