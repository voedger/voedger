/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AppDef_AddGDoc(t *testing.T) {
	require := require.New(t)

	var app IAppDef
	docName, recName := NewQName("test", "doc"), NewQName("test", "rec")

	t.Run("must be ok to add document", func(t *testing.T) {
		appDef := New()

		doc := appDef.AddGDoc(docName)
		require.Equal(DefKind_GDoc, doc.Kind())
		require.Equal(doc, appDef.GDoc(docName))

		t.Run("must be ok to add doc fields", func(t *testing.T) {
			doc.
				AddField("f1", DataKind_int64, true).
				AddField("f2", DataKind_string, false)
			require.Equal(2, doc.UserFieldCount())
		})

		t.Run("must be ok to add child", func(t *testing.T) {
			rec := appDef.AddGRecord(recName)
			require.Equal(DefKind_GRecord, rec.Kind())
			require.Equal(rec, appDef.GRecord(recName))

			doc.AddContainer("rec", recName, 0, Occurs_Unbounded)
			require.Equal(1, doc.ContainerCount())
			require.Equal(rec, doc.Container("rec").Def())

			t.Run("must be ok to add rec fields", func(t *testing.T) {
				rec.
					AddField("f1", DataKind_int64, true).
					AddField("f2", DataKind_string, false)
				require.Equal(2, rec.UserFieldCount())
			})
		})

		require.Equal(2, appDef.DefCount())

		t.Run("must be ok to build", func(t *testing.T) {
			a, err := appDef.Build()
			require.NoError(err)
			require.NotNil(a)

			app = a
		})
	})

	require.NotNil(app)

	t.Run("must be ok to find builded doc", func(t *testing.T) {
		def := app.Def(docName)
		require.Equal(DefKind_GDoc, def.Kind())

		d, ok := def.(IGDoc)
		require.True(ok)
		require.Equal(DefKind_GDoc, d.Kind())

		doc := app.GDoc(docName)
		require.Equal(DefKind_GDoc, doc.Kind())
		require.Equal(d, doc)

		require.Equal(2, doc.UserFieldCount())
		require.Equal(DataKind_int64, doc.Field("f1").DataKind())

		require.Equal(1, doc.ContainerCount())
		require.Equal(recName, doc.Container("rec").QName())
		require.Equal(DefKind_GRecord, doc.Container("rec").Def().Kind())

		t.Run("must be ok to find builded record", func(t *testing.T) {
			def := app.Def(recName)
			require.Equal(DefKind_GRecord, def.Kind())

			r, ok := def.(IGRecord)
			require.True(ok)
			require.Equal(DefKind_GRecord, r.Kind())

			rec := app.GRecord(recName)
			require.Equal(DefKind_GRecord, rec.Kind())
			require.Equal(r, rec)

			require.Equal(2, rec.UserFieldCount())
			require.Equal(DataKind_int64, rec.Field("f1").DataKind())

			require.Equal(0, rec.ContainerCount())
		})
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
		apb := New()
		require.Panics(func() {
			apb.AddGDoc(NullQName)
		})
	})

	t.Run("panic if name is invalid", func(t *testing.T) {
		apb := New()
		require.Panics(func() {
			apb.AddGDoc(NewQName("naked", "🔫"))
		})
	})

	t.Run("panic if definition with name already exists", func(t *testing.T) {
		testName := NewQName("test", "dupe")
		apb := New()
		apb.AddGDoc(testName)
		require.Panics(func() {
			apb.AddGDoc(testName)
		})
	})
}
