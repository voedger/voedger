/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_AppDef_AddGDoc(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef
	wsName := appdef.NewQName("test", "workspace")
	docName, recName := appdef.NewQName("test", "doc"), appdef.NewQName("test", "rec")

	t.Run("should be ok to add document", func(t *testing.T) {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)

		doc := ws.AddGDoc(docName)

		t.Run("should be ok to add doc fields", func(t *testing.T) {
			doc.
				AddField("f1", appdef.DataKind_int64, true).
				AddField("f2", appdef.DataKind_string, false)
		})

		t.Run("should be ok to add child", func(t *testing.T) {
			rec := ws.AddGRecord(recName)
			doc.AddContainer("rec", recName, 0, appdef.Occurs_Unbounded)

			t.Run("should be ok to add rec fields", func(t *testing.T) {
				rec.
					AddField("f1", appdef.DataKind_int64, true).
					AddField("f2", appdef.DataKind_string, false)
			})
		})

		t.Run("should be ok to make doc abstract", func(t *testing.T) {
			doc.SetAbstract()
		})

		t.Run("should be ok to build", func(t *testing.T) {
			a, err := adb.Build()
			require.NoError(err)
			require.NotNil(a)

			app = a
		})

		require.Equal(fmt.Sprint(doc), fmt.Sprint(appdef.GDoc(app.Type, docName)))
	})

	require.NotNil(app)

	testWith := func(tested testedTypes) {

		t.Run("should be ok to find builded doc", func(t *testing.T) {
			doc := appdef.GDoc(tested.Type, docName)
			require.Equal(appdef.TypeKind_GDoc, doc.Kind())
			require.Equal(tested.Type(docName), doc)

			require.Equal(wsName, doc.Workspace().QName())

			require.NotNil(doc.Field(appdef.SystemField_QName))
			require.Equal(doc.SystemField_QName(), doc.Field(appdef.SystemField_QName))
			require.NotNil(doc.Field(appdef.SystemField_ID))
			require.Equal(doc.SystemField_ID(), doc.Field(appdef.SystemField_ID))

			require.Equal(2, doc.UserFieldCount())
			require.Equal(appdef.DataKind_int64, doc.Field("f1").DataKind())

			require.True(doc.Abstract())

			require.Equal(1, doc.ContainerCount())
			require.Equal(recName, doc.Container("rec").QName())
			require.Equal(appdef.TypeKind_GRecord, doc.Container("rec").Type().Kind())

			t.Run("should be ok to find builded record", func(t *testing.T) {
				rec := appdef.GRecord(tested.Type, recName)
				require.Equal(appdef.TypeKind_GRecord, rec.Kind())
				require.Equal(app.Type(recName), rec)

				require.NotNil(rec.Field(appdef.SystemField_QName))
				require.Equal(rec.SystemField_QName(), rec.Field(appdef.SystemField_QName))
				require.NotNil(rec.Field(appdef.SystemField_ID))
				require.Equal(rec.SystemField_ID(), rec.Field(appdef.SystemField_ID))
				require.NotNil(rec.Field(appdef.SystemField_ParentID))
				require.Equal(rec.SystemField_ParentID(), rec.Field(appdef.SystemField_ParentID))
				require.NotNil(rec.Field(appdef.SystemField_Container))
				require.Equal(rec.SystemField_Container(), rec.Field(appdef.SystemField_Container))

				require.Equal(2, rec.UserFieldCount())
				require.Equal(appdef.DataKind_int64, rec.Field("f1").DataKind())

				require.Zero(rec.ContainerCount())
			})
		})

		t.Run("should be ok to enumerate docs", func(t *testing.T) {
			var docs []appdef.QName
			for doc := range appdef.GDocs(tested.Types()) {
				docs = append(docs, doc.QName())
			}
			require.Len(docs, 1)
			require.Equal(docName, docs[0])

			t.Run("should be ok to enumerate recs", func(t *testing.T) {
				var recs []appdef.QName
				for rec := range appdef.GRecords(tested.Types()) {
					recs = append(recs, rec.QName())
				}
				require.Len(recs, 1)
				require.Equal(recName, recs[0])
			})
		})

		t.Run("check nil returns", func(t *testing.T) {
			unknown := appdef.NewQName("test", "unknown")
			require.Nil(appdef.GDoc(tested.Type, unknown))
			require.Nil(appdef.GRecord(tested.Type, unknown))
		})
	}

	testWith(app)
	testWith(app.Workspace(wsName))

	t.Run("should be panics", func(t *testing.T) {

		t.Run("if no name", func(t *testing.T) {
			ws := appdef.New().AddWorkspace(wsName)
			require.Panics(func() {
				ws.AddGDoc(appdef.NullQName)
			}, require.Is(appdef.ErrMissedError))
		})

		t.Run("if invalid name", func(t *testing.T) {
			ws := appdef.New().AddWorkspace(wsName)
			require.Panics(func() {
				ws.AddGDoc(appdef.NewQName("naked", "🔫"))
			}, require.Is(appdef.ErrInvalidError), require.Has("naked.🔫"))
		})

		t.Run("if type with name already exists", func(t *testing.T) {
			testName := appdef.NewQName("test", "dupe")
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")
			ws := adb.AddWorkspace(wsName)
			ws.AddGDoc(testName)
			require.Panics(func() {
				ws.AddGRecord(testName)
			}, require.Is(appdef.ErrAlreadyExistsError), require.Has(testName.String()))
		})
	})
}
