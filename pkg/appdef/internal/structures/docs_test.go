/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package structures_test

import (
	"iter"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
)

type docFixture struct {
	name       string
	docKind    appdef.TypeKind
	recKind    appdef.TypeKind
	skipSystem bool
	addDoc     func(ws appdef.IWorkspaceBuilder, n appdef.QName) appdef.IDocBuilder
	addRec     func(ws appdef.IWorkspaceBuilder, n appdef.QName) appdef.IContainedRecordBuilder
	findDoc    func(f appdef.FindType, n appdef.QName) appdef.IDoc
	findRec    func(f appdef.FindType, n appdef.QName) appdef.IRecord
	enumDocs   func(types appdef.TypesSlice) iter.Seq[appdef.IDoc]
	enumRecs   func(types appdef.TypesSlice) iter.Seq[appdef.IRecord]
}

func seqAsDoc[T appdef.IDoc](s iter.Seq[T]) iter.Seq[appdef.IDoc] {
	return func(visit func(appdef.IDoc) bool) {
		for v := range s {
			if !visit(v) {
				return
			}
		}
	}
}

func seqAsRec[T appdef.IRecord](s iter.Seq[T]) iter.Seq[appdef.IRecord] {
	return func(visit func(appdef.IRecord) bool) {
		for v := range s {
			if !visit(v) {
				return
			}
		}
	}
}

var docFixtures = []docFixture{
	{
		name: "CDoc", docKind: appdef.TypeKind_CDoc, recKind: appdef.TypeKind_CRecord, skipSystem: true,
		addDoc:   func(ws appdef.IWorkspaceBuilder, n appdef.QName) appdef.IDocBuilder { return ws.AddCDoc(n) },
		addRec:   func(ws appdef.IWorkspaceBuilder, n appdef.QName) appdef.IContainedRecordBuilder { return ws.AddCRecord(n) },
		findDoc:  func(f appdef.FindType, n appdef.QName) appdef.IDoc { return appdef.CDoc(f, n) },
		findRec:  func(f appdef.FindType, n appdef.QName) appdef.IRecord { return appdef.CRecord(f, n) },
		enumDocs: func(ts appdef.TypesSlice) iter.Seq[appdef.IDoc] { return seqAsDoc(appdef.CDocs(ts)) },
		enumRecs: func(ts appdef.TypesSlice) iter.Seq[appdef.IRecord] { return seqAsRec(appdef.CRecords(ts)) },
	},
	{
		name: "GDoc", docKind: appdef.TypeKind_GDoc, recKind: appdef.TypeKind_GRecord,
		addDoc:   func(ws appdef.IWorkspaceBuilder, n appdef.QName) appdef.IDocBuilder { return ws.AddGDoc(n) },
		addRec:   func(ws appdef.IWorkspaceBuilder, n appdef.QName) appdef.IContainedRecordBuilder { return ws.AddGRecord(n) },
		findDoc:  func(f appdef.FindType, n appdef.QName) appdef.IDoc { return appdef.GDoc(f, n) },
		findRec:  func(f appdef.FindType, n appdef.QName) appdef.IRecord { return appdef.GRecord(f, n) },
		enumDocs: func(ts appdef.TypesSlice) iter.Seq[appdef.IDoc] { return seqAsDoc(appdef.GDocs(ts)) },
		enumRecs: func(ts appdef.TypesSlice) iter.Seq[appdef.IRecord] { return seqAsRec(appdef.GRecords(ts)) },
	},
	{
		name: "ODoc", docKind: appdef.TypeKind_ODoc, recKind: appdef.TypeKind_ORecord,
		addDoc:   func(ws appdef.IWorkspaceBuilder, n appdef.QName) appdef.IDocBuilder { return ws.AddODoc(n) },
		addRec:   func(ws appdef.IWorkspaceBuilder, n appdef.QName) appdef.IContainedRecordBuilder { return ws.AddORecord(n) },
		findDoc:  func(f appdef.FindType, n appdef.QName) appdef.IDoc { return appdef.ODoc(f, n) },
		findRec:  func(f appdef.FindType, n appdef.QName) appdef.IRecord { return appdef.ORecord(f, n) },
		enumDocs: func(ts appdef.TypesSlice) iter.Seq[appdef.IDoc] { return seqAsDoc(appdef.ODocs(ts)) },
		enumRecs: func(ts appdef.TypesSlice) iter.Seq[appdef.IRecord] { return seqAsRec(appdef.ORecords(ts)) },
	},
	{
		name: "WDoc", docKind: appdef.TypeKind_WDoc, recKind: appdef.TypeKind_WRecord,
		addDoc:   func(ws appdef.IWorkspaceBuilder, n appdef.QName) appdef.IDocBuilder { return ws.AddWDoc(n) },
		addRec:   func(ws appdef.IWorkspaceBuilder, n appdef.QName) appdef.IContainedRecordBuilder { return ws.AddWRecord(n) },
		findDoc:  func(f appdef.FindType, n appdef.QName) appdef.IDoc { return appdef.WDoc(f, n) },
		findRec:  func(f appdef.FindType, n appdef.QName) appdef.IRecord { return appdef.WRecord(f, n) },
		enumDocs: func(ts appdef.TypesSlice) iter.Seq[appdef.IDoc] { return seqAsDoc(appdef.WDocs(ts)) },
		enumRecs: func(ts appdef.TypesSlice) iter.Seq[appdef.IRecord] { return seqAsRec(appdef.WRecords(ts)) },
	},
}

func Test_Docs(t *testing.T) {
	for _, fx := range docFixtures {
		t.Run(fx.name, func(t *testing.T) {
			require := require.New(t)

			wsName := appdef.NewQName("test", "workspace")
			docName, recName := appdef.NewQName("test", "doc"), appdef.NewQName("test", "rec")

			var app appdef.IAppDef

			t.Run("should be ok to add document", func(t *testing.T) {
				adb := builder.New()
				adb.AddPackage("test", "test.com/test")

				ws := adb.AddWorkspace(wsName)

				doc := fx.addDoc(ws, docName)
				doc.AddField("f1", appdef.DataKind_int64, true)
				doc.AddContainer("rec", recName, 0, appdef.Occurs_Unbounded)
				rec := fx.addRec(ws, recName)
				rec.AddField("f1", appdef.DataKind_int64, true)

				a, err := adb.Build()
				require.NoError(err)
				app = a
			})

			testWith := func(tested types.IWithTypes) {
				t.Run("should be ok to find builded doc", func(t *testing.T) {
					doc := fx.findDoc(tested.Type, docName)
					require.Equal(fx.docKind, doc.Kind())
					require.Equal(fx.recKind, doc.Container("rec").Type().Kind())

					t.Run("should be ok to find builded record", func(t *testing.T) {
						rec := fx.findRec(tested.Type, recName)
						require.Equal(fx.recKind, rec.Kind())
						require.Zero(rec.ContainerCount())
					})
				})

				unknownName := appdef.NewQName("test", "unknown")
				require.Nil(fx.findDoc(tested.Type, unknownName))
				require.Nil(fx.findRec(tested.Type, unknownName))

				t.Run("should be ok to enumerate docs", func(t *testing.T) {
					var docs []appdef.QName
					for doc := range fx.enumDocs(tested.Types()) {
						if fx.skipSystem && doc.IsSystem() {
							continue
						}
						docs = append(docs, doc.QName())
					}
					require.Equal([]appdef.QName{docName}, docs)
					t.Run("should be ok to enumerate recs", func(t *testing.T) {
						var recs []appdef.QName
						for rec := range fx.enumRecs(tested.Types()) {
							recs = append(recs, rec.QName())
						}
						require.Equal([]appdef.QName{recName}, recs)
					})
				})
			}

			testWith(app)
			testWith(app.Workspace(wsName))
		})
	}
}
