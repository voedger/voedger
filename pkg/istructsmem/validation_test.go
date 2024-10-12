/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
)

func Test_ValidEventArgs(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	adb := appdef.New()
	adb.AddPackage("test", "test.com/test")

	docName := appdef.NewQName("test", "document")
	rec1Name := appdef.NewQName("test", "record1")
	rec2Name := appdef.NewQName("test", "record2")

	objName := appdef.NewQName("test", "object")

	t.Run("must be ok to build test application", func(t *testing.T) {
		doc := adb.AddODoc(docName)
		doc.
			AddField("RequiredField", appdef.DataKind_int32, true).
			AddRefField("RefField", false, rec1Name)
		doc.
			AddContainer("child", rec1Name, 1, 1).
			AddContainer("child2", rec2Name, 0, appdef.Occurs_Unbounded)

		_ = adb.AddORecord(rec1Name)

		rec2 := adb.AddORecord(rec2Name)
		rec2.AddRefField("RequiredRefField", true, rec2Name)

		obj := adb.AddObject(objName)
		obj.AddContainer("objChild", objName, 0, appdef.Occurs_Unbounded)
	})

	cfgs := make(AppConfigsType, 1)
	cfg := cfgs.AddBuiltInAppConfig(appName, adb)
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

	provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

	app, err := provider.BuiltIn(appName)
	require.NoError(err)

	t.Run("error if event name is not a command or odoc", func(t *testing.T) {
		b := app.Events().GetNewRawEventBuilder(
			istructs.NewRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: 25,
					PLogOffset:        100500,
					Workspace:         1,
					WLogOffset:        1050,
					QName:             objName, // <- error here
					RegisteredAt:      123456789,
				}})

		_, err := b.BuildRawEvent()
		require.ErrorIs(err, ErrNameNotFound)
		require.ErrorContains(err, "command function «test.object» not found")
	})

	oDocEvent := func(sync bool) istructs.IRawEventBuilder {
		var b istructs.IRawEventBuilder
		if sync {
			b = app.Events().GetSyncRawEventBuilder(
				istructs.SyncRawEventBuilderParams{
					GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
						HandlingPartition: 25,
						PLogOffset:        100500,
						Workspace:         1,
						WLogOffset:        1050,
						QName:             docName,
						RegisteredAt:      123456789,
					},
					Device:   1,
					SyncedAt: 123456789,
				})
		} else {
			b = app.Events().GetNewRawEventBuilder(
				istructs.NewRawEventBuilderParams{
					GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
						HandlingPartition: 25,
						PLogOffset:        100500,
						Workspace:         1,
						WLogOffset:        1050,
						QName:             docName,
						RegisteredAt:      123456789,
					},
				})
		}
		return b
	}

	t.Run("error if empty doc", func(t *testing.T) {
		e := oDocEvent(false)
		_, err := e.BuildRawEvent()
		require.ErrorIs(err, ErrNameNotFound)
		require.ErrorContains(err, "ODoc «test.document» misses required field «sys.ID»")
	})

	t.Run("errors in argument IDs and refs", func(t *testing.T) {

		t.Run("error if not raw ID in new event", func(t *testing.T) {
			e := oDocEvent(false)
			doc := e.ArgumentObjectBuilder()
			doc.PutRecordID(appdef.SystemField_ID, 123456789012345) // <- error here
			doc.PutInt32("RequiredField", 7)
			_, err := e.BuildRawEvent()
			require.ErrorIs(err, ErrRawRecordIDRequired)
			require.ErrorContains(err, "ODoc «test.document» should use raw record ID (not «123456789012345»)")
		})

		t.Run("error if repeatedly uses record ID", func(t *testing.T) {
			e := oDocEvent(false)
			doc := e.ArgumentObjectBuilder()
			doc.PutRecordID(appdef.SystemField_ID, 1)
			doc.PutInt32("RequiredField", 7)
			rec := doc.ChildBuilder("child")
			rec.PutRecordID(appdef.SystemField_ID, 1) // <- error here
			_, err := e.BuildRawEvent()
			require.ErrorIs(err, ErrRecordIDUniqueViolation)
			require.ErrorContains(err, "ODoc «test.document» repeatedly uses record ID «1» in ORecord «child: test.record1»")
		})

		t.Run("error if ref to unknown id", func(t *testing.T) {
			e := oDocEvent(false)
			doc := e.ArgumentObjectBuilder()
			doc.PutRecordID(appdef.SystemField_ID, 1)
			doc.PutInt32("RequiredField", 7)
			doc.PutRecordID("RefField", 7) // <- error here
			_, err := e.BuildRawEvent()
			require.ErrorIs(err, ErrRecordIDNotFound)
			require.ErrorContains(err, "ODoc «test.document» field «RefField» refers to unknown record ID «7»")
		})

		t.Run("error if ref to id from invalid target QName", func(t *testing.T) {
			e := oDocEvent(false)
			doc := e.ArgumentObjectBuilder()
			doc.PutRecordID(appdef.SystemField_ID, 1)
			doc.PutInt32("RequiredField", 7)
			doc.PutRecordID("RefField", 1) // <- error here
			_, err := e.BuildRawEvent()
			require.ErrorIs(err, ErrWrongRecordID)
			require.ErrorContains(err, "ODoc «test.document» field «RefField» refers to record ID «1» that has unavailable target QName «test.document»")
		})
	})

	t.Run("error if invalid argument QName", func(t *testing.T) {
		e := oDocEvent(false)
		doc := e.ArgumentObjectBuilder()

		doc.(interface{ clear() }).clear()

		doc.PutQName(appdef.SystemField_QName, rec1Name) // <- error here
		doc.PutRecordID(appdef.SystemField_ID, 1)
		_, err := e.BuildRawEvent()
		require.ErrorIs(err, ErrWrongType)
		require.ErrorContains(err, "event «test.document» argument uses wrong type «test.record1», expected «test.document»")
	})

	t.Run("error if invalid unlogged argument QName", func(t *testing.T) {
		e := oDocEvent(false)
		doc := e.ArgumentObjectBuilder()
		doc.PutRecordID(appdef.SystemField_ID, 1)
		doc.PutInt32("RequiredField", 7)

		unl := e.ArgumentUnloggedObjectBuilder()
		unl.PutQName(appdef.SystemField_QName, objName) // <- error here

		_, err := e.BuildRawEvent()
		require.ErrorIs(err, ErrWrongType)
		require.ErrorContains(err, "event «test.document» unlogged argument uses wrong type «test.object»")
	})

	t.Run("error if argument not valid", func(t *testing.T) {

		t.Run("error if misses required field", func(t *testing.T) {
			e := oDocEvent(false)
			doc := e.ArgumentObjectBuilder()
			doc.PutRecordID(appdef.SystemField_ID, 1)
			_, err := e.BuildRawEvent()
			require.ErrorIs(err, ErrNameNotFound)
			require.ErrorContains(err, "ODoc «test.document» misses required field «RequiredField»")
		})

		t.Run("error if required ref field filled with NullRecordID", func(t *testing.T) {
			e := oDocEvent(false)
			doc := e.ArgumentObjectBuilder()
			doc.PutRecordID(appdef.SystemField_ID, 1)
			doc.PutInt32("RequiredField", 7)
			rec := doc.ChildBuilder("child2")
			rec.PutRecordID(appdef.SystemField_ID, 2)
			rec.PutRecordID("RequiredRefField", istructs.NullRecordID) // <- error here
			_, err := e.BuildRawEvent()
			require.ErrorIs(err, ErrWrongRecordID)
			require.ErrorContains(err, "ORecord «child2: test.record2» required ref field «RequiredRefField» has NullRecordID")
		})

		t.Run("error if corrupted argument container structure", func(t *testing.T) {

			t.Run("error if min occurs violated", func(t *testing.T) {
				e := oDocEvent(false)
				doc := e.ArgumentObjectBuilder()
				doc.PutRecordID(appdef.SystemField_ID, 1)
				doc.PutInt32("RequiredField", 7)

				_, err := e.BuildRawEvent()
				require.ErrorIs(err, ErrMinOccursViolation)
				require.ErrorContains(err, "ODoc «test.document» container «child» has not enough occurrences (0, minimum 1)")
			})

			t.Run("error if max occurs exceeded", func(t *testing.T) {
				e := oDocEvent(false)
				doc := e.ArgumentObjectBuilder()
				doc.PutRecordID(appdef.SystemField_ID, 1)
				doc.PutInt32("RequiredField", 7)

				doc.ChildBuilder("child").
					PutRecordID(appdef.SystemField_ID, 2)

				doc.ChildBuilder("child").
					PutRecordID(appdef.SystemField_ID, 3) // <- error here

				_, err := e.BuildRawEvent()
				require.ErrorIs(err, ErrMaxOccursViolation)
				require.ErrorContains(err, "ODoc «test.document» container «child» has too many occurrences (2, maximum 1)")
			})

			t.Run("error if unknown container used", func(t *testing.T) {
				e := oDocEvent(false)
				doc := e.ArgumentObjectBuilder()
				doc.PutRecordID(appdef.SystemField_ID, 1)
				doc.PutInt32("RequiredField", 7)

				rec := doc.ChildBuilder("child")
				rec.PutRecordID(appdef.SystemField_ID, 2)
				rec.PutString(appdef.SystemField_Container, "objChild") // <- error here
				_, err := e.BuildRawEvent()
				require.ErrorIs(err, ErrNameNotFound)
				require.ErrorContains(err, "ODoc «test.document» child[0] has unknown container name «objChild»")
			})

			t.Run("error if invalid QName used for container", func(t *testing.T) {
				e := oDocEvent(false)
				doc := e.ArgumentObjectBuilder()
				doc.PutRecordID(appdef.SystemField_ID, 1)
				doc.PutInt32("RequiredField", 7)

				rec := doc.ChildBuilder("child")
				rec.PutRecordID(appdef.SystemField_ID, 2)
				rec.PutString(appdef.SystemField_Container, "child2") // <- error here
				_, err := e.BuildRawEvent()
				require.ErrorIs(err, ErrWrongType)
				require.ErrorContains(err, "ODoc «test.document» child[0] ORecord «child2: test.record1» has wrong type name, expected «test.record2»")
			})

			t.Run("error if wrong parent ID", func(t *testing.T) {
				e := oDocEvent(false)
				doc := e.ArgumentObjectBuilder()
				doc.PutRecordID(appdef.SystemField_ID, 1)
				doc.PutInt32("RequiredField", 7)

				rec := doc.ChildBuilder("child")
				rec.PutRecordID(appdef.SystemField_ID, 2)
				rec.PutRecordID(appdef.SystemField_ParentID, 2) // <- error here
				_, err := e.BuildRawEvent()
				require.ErrorIs(err, ErrWrongRecordID)
				require.ErrorContains(err, "ODoc «test.document» child[0] ORecord «child: test.record1» has wrong parent id «2», expected «1»")
			})

			t.Run("must ok if parent ID if omitted", func(t *testing.T) {
				e := oDocEvent(false)
				doc := e.ArgumentObjectBuilder()
				doc.PutRecordID(appdef.SystemField_ID, 1)
				doc.PutInt32("RequiredField", 7)

				rec := doc.ChildBuilder("child")
				rec.PutRecordID(appdef.SystemField_ID, 2)
				rec.PutRecordID(appdef.SystemField_ParentID, istructs.NullRecordID) // <- to restore omitted parent ID
				_, err := e.BuildRawEvent()
				require.NoError(err)

				t.Run("check restored parent ID", func(t *testing.T) {
					d, err := doc.Build()
					require.NoError(err)
					cnt := 0
					for c := range d.Children("child") {
						cnt++
						require.EqualValues(1, c.AsRecordID(appdef.SystemField_ParentID))
					}
					require.Equal(1, cnt)
				})
			})
		})
	})
}

func Test_ValidSysCudEvent(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	adb := appdef.New()
	adb.AddPackage("test", "test.com/test")

	docName := appdef.NewQName("test", "document")
	rec1Name := appdef.NewQName("test", "record1")
	rec2Name := appdef.NewQName("test", "record2")

	objName := appdef.NewQName("test", "object")

	t.Run("must be ok to build test application", func(t *testing.T) {
		doc := adb.AddCDoc(docName)
		doc.
			AddField("RequiredField", appdef.DataKind_int32, true).
			AddRefField("RefField", false, rec1Name)
		doc.
			AddContainer("child", rec1Name, 0, appdef.Occurs_Unbounded).
			AddContainer("child2", rec2Name, 0, appdef.Occurs_Unbounded)

		_ = adb.AddCRecord(rec1Name)
		_ = adb.AddCRecord(rec2Name)

		obj := adb.AddObject(objName)
		obj.AddContainer("objChild", objName, 0, appdef.Occurs_Unbounded)
	})

	cfgs := make(AppConfigsType, 1)
	cfg := cfgs.AddBuiltInAppConfig(appName, adb)
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

	provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

	app, err := provider.BuiltIn(appName)
	require.NoError(err)

	cudRawEvent := func(sync bool) istructs.IRawEventBuilder {
		var b istructs.IRawEventBuilder
		if sync {
			b = app.Events().GetSyncRawEventBuilder(
				istructs.SyncRawEventBuilderParams{
					GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
						HandlingPartition: 25,
						PLogOffset:        100500,
						Workspace:         1,
						WLogOffset:        1050,
						QName:             istructs.QNameCommandCUD,
						RegisteredAt:      123456789,
					},
					Device:   1,
					SyncedAt: 123456789,
				})
		} else {
			b = app.Events().GetNewRawEventBuilder(
				istructs.NewRawEventBuilderParams{
					GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
						HandlingPartition: 25,
						PLogOffset:        100500,
						Workspace:         1,
						WLogOffset:        1050,
						QName:             istructs.QNameCommandCUD,
						RegisteredAt:      123456789,
					},
				})
		}
		return b
	}

	testDocRec := func(id istructs.RecordID) istructs.IRecord {
		r := newRecord(cfg)
		r.PutQName(appdef.SystemField_QName, docName)
		r.PutRecordID(appdef.SystemField_ID, id)
		r.PutInt32("RequiredField", 7)
		require.NoError(r.build())
		return r
	}

	t.Run("error if empty CUD", func(t *testing.T) {
		e := cudRawEvent(false)
		_, err := e.BuildRawEvent()
		require.ErrorIs(err, ErrCUDsMissed)
	})

	t.Run("must error if empty CUD QName", func(t *testing.T) {
		e := cudRawEvent(false)
		_ = e.CUDBuilder().Create(appdef.NullQName) // <- error here
		_, err := e.BuildRawEvent()
		require.ErrorIs(err, ErrUnexpectedTypeKind)
		require.ErrorContains(err, "null row")
	})

	t.Run("must error if wrong CUD type kind", func(t *testing.T) {
		e := cudRawEvent(false)
		_ = e.CUDBuilder().Create(objName) // <- error here
		_, err := e.BuildRawEvent()
		require.ErrorIs(err, ErrUnexpectedTypeKind)
		require.ErrorContains(err, "Object «test.object»")
	})

	t.Run("test raw IDs in CUD.Create", func(t *testing.T) {

		t.Run("must require for new raw event", func(t *testing.T) {
			e := cudRawEvent(false)
			rec := e.CUDBuilder().Create(docName)
			rec.PutRecordID(appdef.SystemField_ID, 123456789012345) // <- error here
			_, err := e.BuildRawEvent()
			require.ErrorIs(err, ErrRawRecordIDRequired)
			require.ErrorContains(err, "should use raw record ID (not «123456789012345»)")
		})

		t.Run("no error for sync events", func(t *testing.T) {
			e := cudRawEvent(true)
			d := e.CUDBuilder().Create(docName)
			d.PutRecordID(appdef.SystemField_ID, 123456789012345)
			d.PutInt32("RequiredField", 1)
			_, err := e.BuildRawEvent()
			require.NoError(err)
		})
	})

	t.Run("must error if raw id in CUD.Update", func(t *testing.T) {
		e := cudRawEvent(false)
		_ = e.CUDBuilder().Update(testDocRec(1)) // <- error here
		_, err := e.BuildRawEvent()
		require.ErrorIs(err, ErrRawRecordIDUnexpected)
		require.ErrorContains(err, "unexpectedly uses raw record ID «1»")
	})

	t.Run("must error if ID duplication", func(t *testing.T) {

		t.Run("raw ID duplication", func(t *testing.T) {
			e := cudRawEvent(false)
			d := e.CUDBuilder().Create(docName)
			d.PutRecordID(appdef.SystemField_ID, 1)
			d.PutInt32("RequiredField", 7)

			r := e.CUDBuilder().Create(rec1Name)
			r.PutRecordID(appdef.SystemField_ParentID, 1)
			r.PutString(appdef.SystemField_Container, "child")
			r.PutRecordID(appdef.SystemField_ID, 1) // <- error here

			_, err := e.BuildRawEvent()
			require.ErrorIs(err, ErrRecordIDUniqueViolation)
			require.ErrorContains(err, "repeatedly uses record ID «1»")
		})

		t.Run("storage ID duplication in Update", func(t *testing.T) {
			e := cudRawEvent(true)

			c := e.CUDBuilder().Create(docName)
			c.PutRecordID(appdef.SystemField_ID, 123456789012345)
			c.PutInt32("RequiredField", 7)

			u := e.CUDBuilder().Update(testDocRec(123456789012345)) // <- error here
			u.PutInt32("RequiredField", 7)

			_, err := e.BuildRawEvent()
			require.ErrorIs(err, ErrRecordIDUniqueViolation)
			require.ErrorContains(err, "repeatedly uses record ID «123456789012345»")
		})

	})

	t.Run("must error if invalid ID refs", func(t *testing.T) {

		t.Run("must error if unknown ID refs", func(t *testing.T) {
			e := cudRawEvent(false)
			d := e.CUDBuilder().Create(docName)
			d.PutRecordID(appdef.SystemField_ID, 1)
			d.PutInt32("RequiredField", 1)
			d.PutRecordID("RefField", 7) // <- error here

			_, err := e.BuildRawEvent()
			require.ErrorIs(err, ErrRecordIDNotFound)
			require.ErrorContains(err, "unknown record ID «7»")
		})

		t.Run("must error if ID refs to invalid QName", func(t *testing.T) {
			e := cudRawEvent(false)
			d := e.CUDBuilder().Create(docName)
			d.PutRecordID(appdef.SystemField_ID, 1)
			d.PutInt32("RequiredField", 1)
			d.PutRecordID("RefField", 1) // <- error here

			_, err := e.BuildRawEvent()
			require.ErrorIs(err, ErrWrongRecordID)
			require.ErrorContains(err, "refers to record ID «1» that has unavailable target QName «test.document»")
		})

		t.Run("must error if sys.Parent / sys.Container causes invalid hierarchy", func(t *testing.T) {

			t.Run("must error if container unknown for specified ParentID", func(t *testing.T) {
				e := cudRawEvent(false)
				d := e.CUDBuilder().Create(docName)
				d.PutRecordID(appdef.SystemField_ID, 1)
				d.PutInt32("RequiredField", 1)

				r := e.CUDBuilder().Create(rec1Name)
				r.PutRecordID(appdef.SystemField_ID, 2)
				r.PutRecordID(appdef.SystemField_ParentID, 1)
				r.PutString(appdef.SystemField_Container, "objChild") // <- error here

				_, err := e.BuildRawEvent()
				require.ErrorIs(err, ErrWrongRecordID)
				require.ErrorContains(err, "has no container «objChild»")
			})

			t.Run("must error if specified container has another QName", func(t *testing.T) {
				e := cudRawEvent(false)
				d := e.CUDBuilder().Create(docName)
				d.PutRecordID(appdef.SystemField_ID, 1)
				d.PutInt32("RequiredField", 1)

				c := e.CUDBuilder().Create(rec1Name)
				c.PutRecordID(appdef.SystemField_ID, 2)
				c.PutRecordID(appdef.SystemField_ParentID, 1)
				c.PutString(appdef.SystemField_Container, "child2") // <- error here

				_, err := e.BuildRawEvent()
				require.ErrorIs(err, ErrWrongRecordID)
				require.ErrorContains(err, "container «child2» has another QName «test.record2»")
			})
		})
	})
}

func Test_ValidCommandEvent(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	adb := appdef.New()
	adb.AddPackage("test", "test.com/test")

	cmdName := appdef.NewQName("test", "command")

	oDocName := appdef.NewQName("test", "ODocument")

	wDocName := appdef.NewQName("test", "WDocument")

	t.Run("must be ok to build test application", func(t *testing.T) {
		oDoc := adb.AddODoc(oDocName)
		oDoc.AddRefField("RefField", false)

		wDoc := adb.AddWDoc(wDocName)
		wDoc.AddRefField("RefField", false, oDocName)

		adb.AddCommand(cmdName).SetParam(oDocName).SetResult(wDocName)
	})

	cfgs := make(AppConfigsType, 1)
	cfg := cfgs.AddBuiltInAppConfig(appName, adb)
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
	cfg.Resources.Add(NewCommandFunction(cmdName, NullCommandExec))

	provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

	app, err := provider.BuiltIn(appName)
	require.NoError(err)

	eventBuilder := func(sync bool) istructs.IRawEventBuilder {
		var b istructs.IRawEventBuilder
		if sync {
			b = app.Events().GetSyncRawEventBuilder(
				istructs.SyncRawEventBuilderParams{
					GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
						HandlingPartition: 25,
						PLogOffset:        100500,
						Workspace:         1,
						WLogOffset:        1050,
						QName:             cmdName,
						RegisteredAt:      123456789,
					},
					Device:   1,
					SyncedAt: 123456789,
				})
		} else {
			b = app.Events().GetNewRawEventBuilder(
				istructs.NewRawEventBuilderParams{
					GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
						HandlingPartition: 25,
						PLogOffset:        100500,
						Workspace:         1,
						WLogOffset:        1050,
						QName:             cmdName,
						RegisteredAt:      123456789,
					},
				})
		}
		return b
	}

	t.Run("must be ok to ref from result to argument", func(t *testing.T) {
		e := eventBuilder(false)
		obj := e.ArgumentObjectBuilder()
		obj.PutRecordID(appdef.SystemField_ID, 1)
		res := e.CUDBuilder().Create(wDocName)
		res.PutRecordID(appdef.SystemField_ID, 2)
		res.PutRecordID("RefField", 1)

		_, err := e.BuildRawEvent()
		require.NoError(err)
	})

	t.Run("must error if repeatedly uses record ID", func(t *testing.T) {

		t.Run("repeated raw record ID in new event", func(t *testing.T) {
			e := eventBuilder(false)
			obj := e.ArgumentObjectBuilder()
			obj.PutRecordID(appdef.SystemField_ID, 1)
			res := e.CUDBuilder().Create(wDocName)
			res.PutRecordID(appdef.SystemField_ID, 1) // <- error here

			_, err := e.BuildRawEvent()
			require.ErrorIs(err, ErrRecordIDUniqueViolation)
			require.ErrorContains(err, "repeatedly uses record ID «1»")
		})

		t.Run("repeated storage record ID in synced event", func(t *testing.T) {
			e := eventBuilder(false)
			obj := e.ArgumentObjectBuilder()
			obj.PutRecordID(appdef.SystemField_ID, 123456789012345)
			res := e.CUDBuilder().Create(wDocName)
			res.PutRecordID(appdef.SystemField_ID, 123456789012345) // <- error here

			_, err := e.BuildRawEvent()
			require.ErrorIs(err, ErrRecordIDUniqueViolation)
			require.ErrorContains(err, "repeatedly uses record ID «123456789012345»")
		})
	})

	t.Run("must error if invalid references", func(t *testing.T) {

		t.Run("must error to ref from argument to result", func(t *testing.T) {
			e := eventBuilder(false)
			obj := e.ArgumentObjectBuilder()
			obj.PutRecordID(appdef.SystemField_ID, 1)
			obj.PutRecordID("RefField", 2)

			res := e.CUDBuilder().Create(wDocName)
			res.PutRecordID(appdef.SystemField_ID, 2)

			_, err := e.BuildRawEvent()
			require.ErrorIs(err, ErrRecordIDNotFound)
			require.ErrorContains(err, "unknown record ID «2»")
		})

	})
}

func Test_IObjectBuilderBuild(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	adb := appdef.New()
	adb.AddPackage("test", "test.com/test")

	docName := appdef.NewQName("test", "document")
	recName := appdef.NewQName("test", "record")

	t.Run("must be ok to build test application", func(t *testing.T) {
		oDoc := adb.AddODoc(docName)
		oDoc.AddField("RequiredField", appdef.DataKind_string, true)
		oDoc.AddContainer("child", recName, 0, appdef.Occurs_Unbounded)
		_ = adb.AddORecord(recName)
	})

	cfgs := make(AppConfigsType, 1)
	cfg := cfgs.AddBuiltInAppConfig(appName, adb)
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

	provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

	app, err := provider.BuiltIn(appName)
	require.NoError(err)

	eventBuilder := func() istructs.IRawEventBuilder {
		return app.Events().GetSyncRawEventBuilder(
			istructs.SyncRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: 25,
					PLogOffset:        100500,
					Workspace:         1,
					WLogOffset:        1050,
					QName:             docName,
					RegisteredAt:      123456789,
				},
				Device:   1,
				SyncedAt: 123456789,
			})
	}

	t.Run("must error if required field is empty", func(t *testing.T) {
		b := eventBuilder()
		d := b.ArgumentObjectBuilder()
		_, err := d.Build()
		require.ErrorIs(err, ErrNameNotFound)
		require.ErrorContains(err, "ODoc «test.document» misses required field «RequiredField»")
	})

	t.Run("must error if builder has empty type name", func(t *testing.T) {
		b := eventBuilder()
		d := b.ArgumentObjectBuilder()
		d.(*objectType).clear()
		_, err := d.Build()
		require.ErrorIs(err, ErrNameMissed)
		require.ErrorContains(err, "empty type name")
	})

	t.Run("must error if builder has wrong type name", func(t *testing.T) {
		b := eventBuilder()
		d := b.ArgumentObjectBuilder()
		d.(*objectType).clear()
		d.PutQName(appdef.SystemField_QName, recName) // <- error here
		_, err := d.Build()
		require.ErrorIs(err, ErrUnexpectedTypeKind)
		require.ErrorContains(err, "wrong type ORecord «test.record»")
	})

	t.Run("must error if builder has errors in IDs", func(t *testing.T) {
		b := eventBuilder()
		d := b.ArgumentObjectBuilder()
		d.PutRecordID(appdef.SystemField_ID, 1)
		r := d.ChildBuilder("child")
		r.PutRecordID(appdef.SystemField_ID, 1) // <- error here
		_, err := d.Build()
		require.ErrorIs(err, ErrRecordIDUniqueViolation)
		require.ErrorContains(err, "repeatedly uses record ID «1»")
	})
}

func Test_VerifiedFields(t *testing.T) {
	require := require.New(t)
	test := test()

	objName := appdef.NewQName("test", "obj")

	adb := appdef.New()
	adb.AddPackage("test", "test.com/test")

	t.Run("must be ok to build application", func(t *testing.T) {
		adb.AddObject(objName).
			AddField("int32", appdef.DataKind_int32, true).
			AddField("email", appdef.DataKind_string, false).
			SetFieldVerify("email", appdef.VerificationKind_EMail).
			AddField("age", appdef.DataKind_int32, false).
			SetFieldVerify("age", appdef.VerificationKind_Any...)
	})

	cfgs := make(AppConfigsType, 1)
	cfg := cfgs.AddBuiltInAppConfig(test.appName, adb)
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

	email := "test@test.io"

	tokens := testTokensFactory().New(test.appName)
	asp := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())
	_, err := asp.BuiltIn(test.appName) // need to set cfg.app because IAppTokens are taken from cfg.app
	require.NoError(err)

	t.Run("test row verification", func(t *testing.T) {

		t.Run("ok verified value type in token", func(t *testing.T) {
			okEmailToken := func() string {
				p := payloads.VerifiedValuePayload{
					VerificationKind: appdef.VerificationKind_EMail,
					Entity:           objName,
					Field:            "email",
					Value:            email,
				}
				token, err := tokens.IssueToken(time.Minute, &p)
				require.NoError(err)
				return token
			}()

			okAgeToken := func() string {
				p := payloads.VerifiedValuePayload{
					VerificationKind: appdef.VerificationKind_Phone,
					Entity:           objName,
					Field:            "age",
					Value:            7,
				}
				token, err := tokens.IssueToken(time.Minute, &p)
				require.NoError(err)
				return token
			}()

			row := makeObject(cfg, objName, nil)
			row.PutInt32("int32", 1)
			row.PutString("email", okEmailToken)
			row.PutString("age", okAgeToken)

			_, err := row.Build()
			require.NoError(err)
		})

		t.Run("error if not token, but not string value", func(t *testing.T) {

			row := makeObject(cfg, objName, nil)
			row.PutInt32("int32", 1)
			row.PutInt32("age", 7)

			_, err := row.Build()
			require.ErrorIs(err, ErrWrongFieldType)
		})

		t.Run("error if not a token, but plain string value", func(t *testing.T) {

			row := makeObject(cfg, objName, nil)
			row.PutInt32("int32", 1)
			row.PutString("email", email)

			_, err := row.Build()
			require.ErrorIs(err, itokens.ErrInvalidToken)
		})

		t.Run("error if unexpected token kind", func(t *testing.T) {
			ukToken := func() string {
				p := payloads.VerifiedValuePayload{
					VerificationKind: appdef.VerificationKind_Phone,
					Entity:           objName,
					Field:            "email",
					Value:            email,
				}
				token, err := tokens.IssueToken(time.Minute, &p)
				require.NoError(err)
				return token
			}()

			row := makeObject(cfg, objName, nil)
			row.PutInt32("int32", 1)
			row.PutString("email", ukToken)

			_, err := row.Build()
			require.ErrorIs(err, ErrInvalidVerificationKind)
			require.ErrorContains(err, "Phone")
		})

		t.Run("error if wrong verified entity in token", func(t *testing.T) {
			weToken := func() string {
				p := payloads.VerifiedValuePayload{
					VerificationKind: appdef.VerificationKind_EMail,
					Entity:           appdef.NewQName("test", "other"),
					Field:            "email",
					Value:            email,
				}
				token, err := tokens.IssueToken(time.Minute, &p)
				require.NoError(err)
				return token
			}()

			row := makeObject(cfg, objName, nil)
			row.PutInt32("int32", 1)
			row.PutString("email", weToken)

			_, err := row.Build()
			require.ErrorIs(err, ErrInvalidName)
		})

		t.Run("error if wrong verified field in token", func(t *testing.T) {
			wfToken := func() string {
				p := payloads.VerifiedValuePayload{
					VerificationKind: appdef.VerificationKind_EMail,
					Entity:           objName,
					Field:            "otherField",
					Value:            email,
				}
				token, err := tokens.IssueToken(time.Minute, &p)
				require.NoError(err)
				return token
			}()

			row := makeObject(cfg, objName, nil)
			row.PutInt32("int32", 1)
			row.PutString("email", wfToken)

			_, err := row.Build()
			require.ErrorIs(err, ErrInvalidName)
		})

		t.Run("error if wrong verified value type in token", func(t *testing.T) {
			wtToken := func() string {
				p := payloads.VerifiedValuePayload{
					VerificationKind: appdef.VerificationKind_EMail,
					Entity:           objName,
					Field:            "email",
					Value:            3.141592653589793238,
				}
				token, err := tokens.IssueToken(time.Minute, &p)
				require.NoError(err)
				return token
			}()

			row := makeObject(cfg, objName, nil)
			row.PutInt32("int32", 1)
			row.PutString("email", wtToken)

			_, err := row.Build()
			require.ErrorIs(err, ErrWrongFieldType)
		})

	})
}

func Test_CharsFieldRestricts(t *testing.T) {
	require := require.New(t)
	test := test()

	objName := appdef.NewQName("test", "obj")

	adb := appdef.New()
	adb.AddPackage("test", "test.com/test")

	t.Run("must be ok to build application", func(t *testing.T) {
		s100Data := appdef.NewQName("test", "s100")
		emailData := appdef.NewQName("test", "email")
		mimeData := appdef.NewQName("test", "mime")

		adb.AddData(s100Data, appdef.DataKind_string, appdef.NullQName,
			appdef.MinLen(1), appdef.MaxLen(100)).SetComment("string 1..100")

		_ = adb.AddData(emailData, appdef.DataKind_string, s100Data,
			appdef.MinLen(6), appdef.Pattern(`^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$`))

		_ = adb.AddData(mimeData, appdef.DataKind_bytes, appdef.NullQName,
			appdef.MinLen(4), appdef.MaxLen(4), appdef.Pattern(`^\w+$`))

		adb.AddObject(objName).
			AddDataField("email", emailData, true).
			AddDataField("mime", mimeData, false)
	})

	cfgs := make(AppConfigsType, 1)
	cfg := cfgs.AddBuiltInAppConfig(test.appName, adb)
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

	asp := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())
	_, err := asp.BuiltIn(test.appName)
	require.NoError(err)

	t.Run("test constraints", func(t *testing.T) {

		t.Run("must be ok check good value", func(t *testing.T) {
			row := makeObject(cfg, objName, nil)
			row.PutString("email", `test@test.io`)
			row.PutBytes("mime", []byte(`abcd`))

			_, err := row.Build()
			require.NoError(err)
		})

		t.Run("must be error if length constraint violated", func(t *testing.T) {
			row := makeObject(cfg, objName, nil)
			row.PutString("email", strings.Repeat("a", 97)+".com") // 97 + 4 = 101 : too long
			row.PutBytes("mime", []byte(`abc`))                    // 3 < 4 : too short

			_, err := row.Build()
			require.ErrorIs(err, ErrDataConstraintViolation)
			require.ErrorContains(err, "string-field «email» data constraint «MaxLen: 100»")
			require.ErrorContains(err, "bytes-field «mime» data constraint «MinLen: 4»")
		})

		t.Run("must be error if pattern restricted", func(t *testing.T) {
			row := makeObject(cfg, objName, nil)
			row.PutString("email", "naked@🔫.error")
			row.PutBytes("mime", []byte(`++++`))

			_, err := row.Build()
			require.ErrorIs(err, ErrDataConstraintViolation)
			require.ErrorContains(err, "string-field «email» data constraint «Pattern:")
			require.ErrorContains(err, "bytes-field «mime» data constraint «Pattern:")
		})
	})
}
