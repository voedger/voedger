/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_AppDef_StructuresAndRecords(t *testing.T) {
	require := require.New(t)

	wsName := NewQName("test", "workspace")
	docName, recName := NewQName("test", "doc"), NewQName("test", "rec")
	objName := NewQName("test", "obj")

	var app IAppDef

	t.Run("should be ok to add structures", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		doc := wsb.AddODoc(docName)
		doc.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)
		doc.AddContainer("rec", recName, 0, Occurs_Unbounded)
		rec := wsb.AddORecord(recName)
		rec.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)

		obj := wsb.AddObject(objName)
		obj.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWithStructures := func(tested IWithStructures) {
		t.Run("should be ok to find builded structures", func(t *testing.T) {
			findStruct := func(n QName, kind TypeKind) {
				typ := tested.(IWithTypes).Type(n)
				require.Equal(kind, typ.Kind())

				doc := tested.Structure(n)
				require.Equal(kind, doc.Kind())

				require.Equal(2, doc.UserFieldCount())
				require.Equal(DataKind_int64, doc.Field("f1").DataKind())
				require.Equal(DataKind_string, doc.Field("f2").DataKind())
			}
			findStruct(docName, TypeKind_ODoc)
			findStruct(recName, TypeKind_ORecord)
			findStruct(objName, TypeKind_Object)
		})

		require.Nil(tested.Structure(NewQName("test", "unknown")), "should nil if not found")

		t.Run("should be ok to enumerate structures", func(t *testing.T) {
			var str []QName
			for s := range tested.Structures {
				str = append(str, s.QName())
			}
			require.Equal(str, []QName{docName, objName, recName})
		})
	}

	testWithStructures(app)
	testWithStructures(app.Workspace(wsName))

	testWithRecords := func(tested IWithRecords) {
		t.Run("should be ok to find builded records", func(t *testing.T) {
			findRecord := func(n QName, kind TypeKind) {
				typ := tested.(IWithTypes).Type(n)
				require.Equal(kind, typ.Kind())

				doc := tested.Record(n)
				require.Equal(kind, doc.Kind())

				require.Equal(2, doc.UserFieldCount())
				require.Equal(DataKind_int64, doc.Field("f1").DataKind())
				require.Equal(DataKind_string, doc.Field("f2").DataKind())
			}
			findRecord(docName, TypeKind_ODoc)
			findRecord(recName, TypeKind_ORecord)
		})

		require.Nil(tested.Record(NewQName("test", "unknown")), "should nil if not found")
		require.Nil(tested.Record(objName), "should nil if not record")

		t.Run("should be ok to enumerate records", func(t *testing.T) {
			var recs []QName
			for s := range tested.Records {
				recs = append(recs, s.QName())
			}
			require.Equal(recs, []QName{docName, recName})
		})
	}

	testWithRecords(app)
	testWithRecords(app.Workspace(wsName))
}
