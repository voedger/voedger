/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package structures_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_StructuresAndRecords(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	docName, recName := appdef.NewQName("test", "doc"), appdef.NewQName("test", "rec")
	objName := appdef.NewQName("test", "obj")

	var app appdef.IAppDef

	t.Run("should be ok to add structures", func(t *testing.T) {
		adb := builder.New()
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

		obj := wsb.AddObject(objName)
		obj.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWith := func(tested types.IWithTypes) {
		t.Run("should be ok to find builded structures", func(t *testing.T) {
			findStruct := func(n appdef.QName, kind appdef.TypeKind) {
				typ := tested.Type(n)
				require.Equal(kind, typ.Kind())

				str := appdef.Structure(tested.Type, n)
				require.Equal(kind, str.Kind())

				require.Equal(wsName, str.Workspace().QName())

				require.Equal(appdef.DataKind_QName, str.SystemField_QName().DataKind())

				require.Equal(2, str.UserFieldCount())
				require.Equal(appdef.DataKind_int64, str.Field("f1").DataKind())
				require.Equal(appdef.DataKind_string, str.Field("f2").DataKind())
			}
			findStruct(docName, appdef.TypeKind_ODoc)
			findStruct(recName, appdef.TypeKind_ORecord)
			findStruct(objName, appdef.TypeKind_Object)
		})

		require.Nil(appdef.Structure(tested.Type, appdef.NewQName("test", "unknown")), "should nil if not found")

		t.Run("should be ok to enumerate structures", func(t *testing.T) {
			var str []appdef.QName
			for s := range appdef.Structures(tested.Types()) {
				if !s.IsSystem() { // skip system structures
					str = append(str, s.QName())
				}
			}
			require.Equal([]appdef.QName{docName, objName, recName}, str)
		})

		t.Run("should be ok to find builded records", func(t *testing.T) {
			findRecord := func(n appdef.QName, kind appdef.TypeKind) {
				typ := tested.Type(n)
				require.Equal(kind, typ.Kind())

				rec := appdef.Record(tested.Type, n)
				require.Equal(kind, rec.Kind())

				require.Equal(appdef.DataKind_RecordID, rec.SystemField_ID().DataKind())
			}
			findRecord(docName, appdef.TypeKind_ODoc)
			findRecord(recName, appdef.TypeKind_ORecord)
		})

		require.Nil(appdef.Record(tested.Type, appdef.NewQName("test", "unknown")), "should nil if not found")
		require.Nil(appdef.Record(tested.Type, objName), "should nil if not record")

		t.Run("should be ok to enumerate records", func(t *testing.T) {
			var recs []appdef.QName
			for r := range appdef.Records(tested.Types()) {
				if !r.IsSystem() { // skip system records
					recs = append(recs, r.QName())
				}
			}
			require.Equal([]appdef.QName{docName, recName}, recs)
		})

		t.Run("should be ok to find builded contained records", func(t *testing.T) {
			findRecord := func(n appdef.QName, kind appdef.TypeKind) {
				typ := tested.Type(n)
				require.Equal(kind, typ.Kind())

				r := appdef.Record(tested.Type, n)
				require.Equal(kind, r.Kind())

				rec, ok := r.(appdef.IContainedRecord)
				require.True(ok)

				require.Equal(appdef.DataKind_RecordID, rec.SystemField_ParentID().DataKind())
				require.Equal(appdef.DataKind_string, rec.SystemField_Container().DataKind())
			}
			findRecord(recName, appdef.TypeKind_ORecord)
		})
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}
