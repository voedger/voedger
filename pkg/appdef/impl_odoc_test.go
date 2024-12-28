/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
)

func Test_AppDef_AddODoc(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	docName, recName := appdef.NewQName("test", "doc"), appdef.NewQName("test", "rec")

	var app appdef.IAppDef

	t.Run("should be ok to add document", func(t *testing.T) {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		doc := wsb.AddODoc(docName)
		doc.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)
		doc.AddContainer("rec", recName, 0, appdef.Occurs_Unbounded)
		rec := wsb.AddORecord(recName)
		rec.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWith := func(tested testedTypes) {
		t.Run("should be ok to find builded doc", func(t *testing.T) {
			typ := tested.Type(docName)
			require.Equal(appdef.TypeKind_ODoc, typ.Kind())

			doc := appdef.ODoc(tested.Type, docName)
			require.Equal(appdef.TypeKind_ODoc, doc.Kind())
			require.Equal(typ.(appdef.IODoc), doc)

			require.Equal(wsName, doc.Workspace().QName())

			require.Equal(2, doc.UserFieldCount())
			require.Equal(appdef.DataKind_int64, doc.Field("f1").DataKind())

			require.Equal(appdef.TypeKind_ORecord, doc.Container("rec").Type().Kind())

			t.Run("should be ok to find builded record", func(t *testing.T) {
				typ := app.Type(recName)
				require.Equal(appdef.TypeKind_ORecord, typ.Kind())

				rec := appdef.ORecord(tested.Type, recName)
				require.Equal(appdef.TypeKind_ORecord, rec.Kind())
				require.Equal(typ.(appdef.IORecord), rec)

				require.Equal(wsName, rec.Workspace().QName())

				require.Equal(2, rec.UserFieldCount())
				require.Equal(appdef.DataKind_int64, rec.Field("f1").DataKind())

				require.Equal(0, rec.ContainerCount())
			})
		})

		t.Run("should be ok to enumerate docs", func(t *testing.T) {
			var docs []appdef.QName
			for doc := range appdef.ODocs(tested.Types()) {
				docs = append(docs, doc.QName())
			}
			require.Len(docs, 1)
			require.Equal(docName, docs[0])
			t.Run("should be ok to enumerate recs", func(t *testing.T) {
				var recs []appdef.QName
				for rec := range appdef.ORecords(tested.Types()) {
					recs = append(recs, rec.QName())
				}
				require.Len(recs, 1)
				require.Equal(recName, recs[0])
			})
		})

		t.Run("should nil if not found", func(t *testing.T) {
			unknown := appdef.NewQName("test", "unknown")
			require.Nil(appdef.ODoc(tested.Type, unknown))
			require.Nil(appdef.ORecord(tested.Type, unknown))
		})
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}
