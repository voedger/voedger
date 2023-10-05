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
		require.Equal(TypeKind_GDoc, doc.Kind())
		require.Equal(doc, appDef.GDoc(docName))

		t.Run("must be ok to add doc fields", func(t *testing.T) {
			doc.
				AddField("f1", DataKind_int64, true).
				AddField("f2", DataKind_string, false)
			require.Equal(2, doc.UserFieldCount())
		})

		t.Run("must be ok to add child", func(t *testing.T) {
			rec := appDef.AddGRecord(recName)
			require.Equal(TypeKind_GRecord, rec.Kind())
			require.Equal(rec, appDef.GRecord(recName))

			doc.AddContainer("rec", recName, 0, Occurs_Unbounded)
			require.Equal(1, doc.ContainerCount())
			require.Equal(rec, doc.Container("rec").Type())

			t.Run("must be ok to add rec fields", func(t *testing.T) {
				rec.
					AddField("f1", DataKind_int64, true).
					AddField("f2", DataKind_string, false)
				require.Equal(2, rec.UserFieldCount())
			})
		})

		t.Run("must be ok to make doc abstract", func(t *testing.T) {
			doc.SetAbstract()
			require.True(doc.Abstract())
		})

		require.Equal(2, appDef.TypeCount())

		t.Run("must be ok to build", func(t *testing.T) {
			a, err := appDef.Build()
			require.NoError(err)
			require.NotNil(a)

			app = a
		})
	})

	require.NotNil(app)

	t.Run("must be ok to find builded doc", func(t *testing.T) {
		typ := app.Type(docName)
		require.Equal(TypeKind_GDoc, typ.Kind())

		d, ok := typ.(IGDoc)
		require.True(ok)
		require.Equal(TypeKind_GDoc, d.Kind())

		doc := app.GDoc(docName)
		require.Equal(TypeKind_GDoc, doc.Kind())
		require.True(doc.IsDoc() && doc.IsGDoc())
		require.Equal(d, doc)

		require.NotNil(doc.Field(SystemField_QName))
		require.NotNil(doc.Field(SystemField_ID))

		require.Equal(2, doc.UserFieldCount())
		require.Equal(DataKind_int64, doc.Field("f1").DataKind())

		require.True(doc.Abstract())

		require.Equal(1, doc.ContainerCount())
		require.Equal(recName, doc.Container("rec").QName())
		require.Equal(TypeKind_GRecord, doc.Container("rec").Type().Kind())

		t.Run("must be ok to find builded record", func(t *testing.T) {
			typ := app.Type(recName)
			require.Equal(TypeKind_GRecord, typ.Kind())

			r, ok := typ.(IGRecord)
			require.True(ok)
			require.Equal(TypeKind_GRecord, r.Kind())
			require.True(r.IsRecord() && r.IsGRecord())

			rec := app.GRecord(recName)
			require.Equal(TypeKind_GRecord, rec.Kind())
			require.Equal(r, rec)

			require.NotNil(rec.Field(SystemField_QName))
			require.NotNil(rec.Field(SystemField_ID))
			require.NotNil(rec.Field(SystemField_ParentID))
			require.NotNil(rec.Field(SystemField_Container))

			require.Equal(2, rec.UserFieldCount())
			require.Equal(DataKind_int64, rec.Field("f1").DataKind())

			require.Zero(rec.ContainerCount())
		})
	})

	t.Run("check nil returns", func(t *testing.T) {
		unknown := NewQName("test", "unknown")
		require.Equal(NullType, app.Type(unknown))
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

	t.Run("panic if type with name already exists", func(t *testing.T) {
		testName := NewQName("test", "dupe")
		apb := New()
		apb.AddGDoc(testName)
		require.Panics(func() {
			apb.AddGDoc(testName)
		})
	})
}
