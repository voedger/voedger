/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AppDef_AddGDoc(t *testing.T) {
	require := require.New(t)

	app := newAppDef()

	t.Run("must be ok to add document", func(t *testing.T) {
		doc := app.AddGDoc(NewQName("test", "doc"))
		require.Equal(DefKind_GDoc, doc.Kind())
		require.Equal(doc, app.GDoc(doc.QName()))

		t.Run("must be ok to add doc fields", func(t *testing.T) {
			doc.
				AddField("f1", DataKind_int64, true).
				AddField("f2", DataKind_string, false)
			require.Equal(2, doc.UserFieldCount())
		})

		t.Run("must be ok to add child", func(t *testing.T) {
			rec := app.AddGRecord(NewQName("test", "rec"))
			require.Equal(DefKind_GRecord, rec.Kind())
			require.Equal(rec, app.GRecord(rec.QName()))

			doc.AddContainer("rec", rec.QName(), 0, Occurs_Unbounded)
			require.Equal(1, doc.ContainerCount())
			require.Equal(rec, doc.ContainerDef("rec"))

			t.Run("must be ok to add rec fields", func(t *testing.T) {
				rec.
					AddField("f1", DataKind_int64, true).
					AddField("f2", DataKind_string, false)
				require.Equal(2, rec.UserFieldCount())
			})
		})

		require.Equal(2, app.DefCount())
	})

	t.Run("must be ok to build", func(t *testing.T) {
		require.True(app.HasChanges())
		_, err := app.Build()
		require.NoError(err)
		require.False(app.HasChanges())
	})

	t.Run("check nil returns", func(t *testing.T) {
		unknown := NewQName("test", "unknown")
		require.Equal(NullDef, app.Def(unknown))
		require.Nil(app.GDoc(unknown))
		require.Nil(app.GRecord(unknown))
		require.Nil(app.CDoc(unknown))
		require.Nil(app.CRecord(unknown))
		require.Nil(app.WDoc(unknown))
		require.Nil(app.WRecord(unknown))
		require.Nil(app.ODoc(unknown))
		require.Nil(app.ORecord(unknown))
		require.Nil(app.Object(unknown))
		require.Nil(app.Element(unknown))
	})

	t.Run("panic if name is empty", func(t *testing.T) {
		require.Panics(func() {
			app.AddGDoc(NullQName)
		})
	})

	t.Run("panic if name is invalid", func(t *testing.T) {
		require.Panics(func() {
			app.AddGDoc(NewQName("naked", "🔫"))
		})
	})

	t.Run("panic if definition with name already exists", func(t *testing.T) {
		testName := NewQName("test", "dupe")
		app.AddGDoc(testName)
		require.Panics(func() {
			app.AddGDoc(testName)
		})
	})
}

func Test_AppDef_AddCDoc(t *testing.T) {
	require := require.New(t)

	app := newAppDef()

	t.Run("must be ok to add singleton", func(t *testing.T) {
		doc := app.AddSingleton(NewQName("test", "doc"))
		require.Equal(DefKind_CDoc, doc.Kind())
		require.Equal(doc, app.CDoc(doc.QName()))
		require.True(doc.Singleton())

		t.Run("must be ok to add child", func(t *testing.T) {
			rec := app.AddCRecord(NewQName("test", "rec"))
			require.Equal(DefKind_CRecord, rec.Kind())
			require.Equal(rec, app.CRecord(rec.QName()))

			doc.AddContainer("rec", rec.QName(), 0, Occurs_Unbounded)
			require.Equal(1, doc.ContainerCount())
			require.Equal(rec, doc.ContainerDef("rec"))
		})
	})
}

func Test_AppDef_AddWDoc(t *testing.T) {
	require := require.New(t)

	app := newAppDef()

	t.Run("must be ok to add document", func(t *testing.T) {
		doc := app.AddWDoc(NewQName("test", "doc"))
		require.Equal(DefKind_WDoc, doc.Kind())
		require.Equal(doc, app.WDoc(doc.QName()))

		t.Run("must be ok to add child", func(t *testing.T) {
			rec := app.AddWRecord(NewQName("test", "rec"))
			require.Equal(DefKind_WRecord, rec.Kind())
			require.Equal(rec, app.WRecord(rec.QName()))

			doc.AddContainer("rec", rec.QName(), 0, Occurs_Unbounded)
			require.Equal(1, doc.ContainerCount())
			require.Equal(rec, doc.ContainerDef("rec"))
		})
	})
}

func Test_AppDef_AddODoc(t *testing.T) {
	require := require.New(t)

	app := newAppDef()

	t.Run("must be ok to add document", func(t *testing.T) {
		doc := app.AddODoc(NewQName("test", "doc"))
		require.Equal(DefKind_ODoc, doc.Kind())
		require.Equal(doc, app.ODoc(doc.QName()))

		t.Run("must be ok to add child", func(t *testing.T) {
			rec := app.AddORecord(NewQName("test", "rec"))
			require.Equal(DefKind_ORecord, rec.Kind())
			require.Equal(rec, app.ORecord(rec.QName()))

			doc.AddContainer("rec", rec.QName(), 0, Occurs_Unbounded)
			require.Equal(1, doc.ContainerCount())
			require.Equal(rec, doc.ContainerDef("rec"))
		})
	})
}

func Test_AppDef_AddObject(t *testing.T) {
	require := require.New(t)

	app := newAppDef()

	t.Run("must be ok to add object", func(t *testing.T) {
		obj := app.AddObject(NewQName("test", "obj"))
		require.Equal(DefKind_Object, obj.Kind())
		require.Equal(obj, app.Object(obj.QName()))

		t.Run("must be ok to add child", func(t *testing.T) {
			elm := app.AddElement(NewQName("test", "elm"))
			require.Equal(DefKind_Element, elm.Kind())
			require.Equal(elm, app.Element(elm.QName()))

			obj.AddContainer("rec", elm.QName(), 0, Occurs_Unbounded)
			require.Equal(1, obj.ContainerCount())
			require.Equal(elm, obj.ContainerDef("rec"))
		})
	})
}
