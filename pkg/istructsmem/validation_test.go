/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/containers"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
)

func Test_ValidEvent(t *testing.T) {
	require := require.New(t)

	var (
		app istructs.IAppStructs

		cmdCreateDoc appdef.QName = appdef.NewQName("test", "CreateDoc")
		cDocName     appdef.QName = appdef.NewQName("test", "CDoc")
		oDocName     appdef.QName = appdef.NewQName("test", "ODoc")

		cmdCreateObj         appdef.QName = appdef.NewQName("test", "CreateObj")
		cmdCreateObjUnlogged appdef.QName = appdef.NewQName("test", "CreateObjUnlogged")
		oObjName             appdef.QName = appdef.NewQName("test", "Object")

		cmdCUD appdef.QName = appdef.NewQName("test", "cudEvent")
	)

	t.Run("builds application", func(t *testing.T) {
		appDef := appdef.New()

		t.Run("must be ok to build application definition", func(t *testing.T) {
			appDef.AddCDoc(cDocName).
				AddField("Int32", appdef.DataKind_int32, true).
				AddStringField("String", false)

			oDoc := appDef.AddODoc(oDocName)
			oDoc.
				AddField("Int32", appdef.DataKind_int32, true).
				AddStringField("String", false)
			oDoc.
				AddContainer("child", oDocName, 0, 2) // ODocs should be able to contain ODocs, see #!19332

			appDef.AddObject(oObjName).
				AddField("Int32", appdef.DataKind_int32, true).
				AddStringField("String", false)
		})

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, appDef)
		cfg.Resources.Add(NewCommandFunction(cmdCreateDoc, cDocName, appdef.NullQName, appdef.NullQName, NullCommandExec))
		cfg.Resources.Add(NewCommandFunction(cmdCreateObj, oObjName, appdef.NullQName, appdef.NullQName, NullCommandExec))
		cfg.Resources.Add(NewCommandFunction(cmdCreateObjUnlogged, appdef.NullQName, oObjName, appdef.NullQName, NullCommandExec))
		cfg.Resources.Add(NewCommandFunction(cmdCUD, appdef.NullQName, appdef.NullQName, appdef.NullQName, NullCommandExec))

		storage, err := simpleStorageProvider().AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)
		err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
		require.NoError(err)

		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

		app, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.NoError(err)
	})

	t.Run("must failed build raw event if empty event command name", func(t *testing.T) {
		bld := app.Events().GetNewRawEventBuilder(
			istructs.NewRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: 25,
					PLogOffset:        100500,
					Workspace:         1,
					WLogOffset:        1050,
					QName:             appdef.NullQName,
					RegisteredAt:      123456789,
				},
			})
		require.NotNil(bld)

		_, err := bld.BuildRawEvent()
		require.ErrorIs(err, ErrNameMissed)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(err, &validateErr)
		require.Equal(ECode_EmptyDefName, validateErr.Code())
	})

	t.Run("must failed build raw event if wrong event unlogged argument name", func(t *testing.T) {
		bld := app.Events().GetNewRawEventBuilder(
			istructs.NewRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: 25,
					PLogOffset:        100500,
					Workspace:         1,
					WLogOffset:        1050,
					QName:             cmdCreateObj,
					RegisteredAt:      123456789,
				},
			})
		require.NotNil(bld)

		cmd := bld.ArgumentObjectBuilder()
		cmd.PutInt32("Int32", 29)

		cmd = bld.ArgumentUnloggedObjectBuilder()
		cmd.PutQName(appdef.SystemField_QName, oObjName)

		_, err := bld.BuildRawEvent()
		require.ErrorIs(err, ErrWrongDefinition)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(err, &validateErr)
		require.Equal(ECode_InvalidDefName, validateErr.Code())
	})

	t.Run("must failed build raw event if wrong filled unlogged argument name", func(t *testing.T) {
		bld := app.Events().GetNewRawEventBuilder(
			istructs.NewRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: 25,
					PLogOffset:        100500,
					Workspace:         1,
					WLogOffset:        1050,
					QName:             cmdCreateObjUnlogged,
					RegisteredAt:      123456789,
				},
			})
		require.NotNil(bld)

		_ = bld.ArgumentUnloggedObjectBuilder()

		_, err := bld.BuildRawEvent()
		require.ErrorIs(err, ErrNameNotFound) // Int32 missed
		validateErr := validateErrorf(0, "")
		require.ErrorAs(err, &validateErr)
		require.Equal(ECode_EmptyData, validateErr.Code())
	})

	t.Run("must failed build raw event if wrong CUD argument", func(t *testing.T) {
		bld := app.Events().GetNewRawEventBuilder(
			istructs.NewRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: 25,
					PLogOffset:        100500,
					Workspace:         1,
					WLogOffset:        1050,
					QName:             cmdCUD,
					RegisteredAt:      123456789,
				},
			})
		require.NotNil(bld)

		cud := bld.CUDBuilder()
		cud.Create(cDocName)

		_, err := bld.BuildRawEvent()
		require.ErrorIs(err, ErrNameNotFound) // sys.ID missed
		validateErr := validateErrorf(0, "")
		require.ErrorAs(err, &validateErr)
		require.Equal(ECode_EmptyData, validateErr.Code())
	})

	t.Run("test allow to create command by ODoc", func(t *testing.T) {
		bld := app.Events().GetNewRawEventBuilder(
			istructs.NewRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: 25,
					PLogOffset:        100500,
					Workspace:         1,
					WLogOffset:        1050,
					QName:             oDocName,
					RegisteredAt:      123456789,
				},
			})
		require.NotNil(bld)

		cmd := bld.ArgumentObjectBuilder()
		cmd.PutString("String", "string data")

		t.Run("must failed build raw event if fields missed", func(t *testing.T) {
			rawEvent, err := bld.BuildRawEvent()
			require.ErrorIs(err, ErrNameNotFound) // Int32 missed
			require.NotNil(rawEvent)
		})

		cmd.PutInt32("Int32", 29)

		t.Run("must failed build raw event (sys.ID field missed)", func(t *testing.T) {
			rawEvent, err := bld.BuildRawEvent()
			require.ErrorIs(err, ErrNameNotFound) // ORecord «test.ODoc» ID missed
			require.NotNil(rawEvent)
		})

		cmd.PutRecordID(appdef.SystemField_ID, 1)

		t.Run("test ok build raw event", func(t *testing.T) {
			rawEvent, err := bld.BuildRawEvent()
			require.NoError(err)
			require.NotNil(rawEvent)
		})
	})

	t.Run("test allow to create object command", func(t *testing.T) {
		bld := app.Events().GetNewRawEventBuilder(
			istructs.NewRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: 25,
					PLogOffset:        100500,
					Workspace:         1,
					WLogOffset:        1050,
					QName:             cmdCreateObj,
					RegisteredAt:      123456789,
				},
			})
		require.NotNil(bld)

		cmd := bld.ArgumentObjectBuilder()
		cmd.PutString("String", "string data")

		t.Run("must failed build raw event if fields missed", func(t *testing.T) {
			rawEvent, err := bld.BuildRawEvent()
			require.ErrorIs(err, ErrNameNotFound) // Int32 missed
			require.NotNil(rawEvent)
		})

		t.Run("must failed build raw event (unexpected argument definition)", func(t *testing.T) {
			cmd.PutQName(appdef.SystemField_QName, oDocName)
			rawEvent, err := bld.BuildRawEvent()
			require.ErrorIs(err, ErrDefChanged) // expected «test.Object», but not «test.ODoc»
			require.NotNil(rawEvent)
		})
	})

	t.Run("test deprecate to create command by CDoc", func(t *testing.T) {
		bld := app.Events().GetNewRawEventBuilder(
			istructs.NewRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: 25,
					PLogOffset:        100500,
					Workspace:         1,
					WLogOffset:        1050,
					QName:             cDocName,
					RegisteredAt:      123456789,
				},
			})
		require.NotNil(bld)

		_ = bld.ArgumentObjectBuilder()

		t.Run("must failed build raw event (unknown command name)", func(t *testing.T) {
			rawEvent, err := bld.BuildRawEvent()
			require.ErrorIs(err, ErrNameNotFound) // there are no command «test.cDoc»
			require.NotNil(rawEvent)
		})
	})

	t.Run("test deprecate create command with CDoc argument, see #!17185", func(t *testing.T) {
		bld := app.Events().GetNewRawEventBuilder(
			istructs.NewRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: 25,
					PLogOffset:        100500,
					Workspace:         1,
					WLogOffset:        1050,
					QName:             cmdCreateDoc,
					RegisteredAt:      123456789,
				},
			})
		require.NotNil(bld)

		cmd := bld.ArgumentObjectBuilder()
		cmd.PutRecordID(appdef.SystemField_ID, 1)
		cmd.PutInt32("Int32", 29)
		cmd.PutString("String", "string data")

		t.Run("must failed build raw event", func(t *testing.T) {
			rawEvent, err := bld.BuildRawEvent()
			require.ErrorIs(err, ErrWrongDefinition) // CDoc deprecated, ODoc or Object expected
			require.NotNil(rawEvent)
		})
	})

	t.Run("ODocs should be able to contain ODocs, see #!19332", func(t *testing.T) {
		bld := app.Events().GetNewRawEventBuilder(
			istructs.NewRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: 25,
					PLogOffset:        100500,
					Workspace:         1,
					WLogOffset:        1050,
					QName:             oDocName,
					RegisteredAt:      123456789,
				},
			})
		require.NotNil(bld)

		cmd := bld.ArgumentObjectBuilder()
		cmd.PutRecordID(appdef.SystemField_ID, 1)
		cmd.PutInt32("Int32", 29)
		cmd.PutString("String", "string data")

		child := cmd.ElementBuilder("child")
		child.PutRecordID(appdef.SystemField_ID, 2)
		child.PutInt32("Int32", 29)
		child.PutString("String", "string data")

		t.Run("test ok build raw event", func(t *testing.T) {
			rawEvent, err := bld.BuildRawEvent()
			require.NoError(err)
			require.NotNil(rawEvent)

			cmd := rawEvent.ArgumentObject()
			require.Equal(istructs.RecordID(1), cmd.AsRecordID(appdef.SystemField_ID))
			require.Equal(int32(29), cmd.AsInt32("Int32"))
			require.Equal("string data", cmd.AsString("String"))

			cnt := 0
			cmd.Elements("child", func(child istructs.IElement) {
				require.Equal(istructs.RecordID(2), child.AsRecordID(appdef.SystemField_ID))
				require.Equal(istructs.RecordID(1), child.AsRecordID(appdef.SystemField_ParentID))
				require.Equal("child", child.AsString(appdef.SystemField_Container))
				require.Equal(int32(29), child.AsInt32("Int32"))
				require.Equal("string data", child.AsString("String"))
				cnt++
			})

			require.Equal(1, cnt)
		})
	})
}

func Test_ValidElement(t *testing.T) {
	require := require.New(t)

	test := test()

	appDef := appdef.New()

	t.Run("must be ok to build test application definition", func(t *testing.T) {

		t.Run("build object definition", func(t *testing.T) {
			objDef := appDef.AddObject(appdef.NewQName("test", "object"))
			objDef.
				AddField("int32Field", appdef.DataKind_int32, true).
				AddField("int64Field", appdef.DataKind_int64, false).
				AddField("float32Field", appdef.DataKind_float32, false).
				AddField("float64Field", appdef.DataKind_float64, false).
				AddField("bytesField", appdef.DataKind_bytes, false).
				AddStringField("strField", false).
				AddField("qnameField", appdef.DataKind_QName, false).
				AddField("recIDField", appdef.DataKind_RecordID, false)
			objDef.
				AddContainer("child", appdef.NewQName("test", "element"), 1, appdef.Occurs_Unbounded)

			elDef := appDef.AddElement(appdef.NewQName("test", "element"))
			elDef.
				AddField("int32Field", appdef.DataKind_int32, true).
				AddField("int64Field", appdef.DataKind_int64, false).
				AddField("float32Field", appdef.DataKind_float32, false).
				AddField("float64Field", appdef.DataKind_float64, false).
				AddField("bytesField", appdef.DataKind_bytes, false).
				AddStringField("strField", false).
				AddField("qnameField", appdef.DataKind_QName, false).
				AddField("boolField", appdef.DataKind_bool, false).
				AddField("recIDField", appdef.DataKind_RecordID, false)
			elDef.
				AddContainer("grandChild", appdef.NewQName("test", "grandChild"), 0, 1)

			subElDef := appDef.AddElement(appdef.NewQName("test", "grandChild"))
			subElDef.
				AddField("recIDField", appdef.DataKind_RecordID, false)
		})

		t.Run("build ODoc definition", func(t *testing.T) {
			docDef := appDef.AddODoc(appdef.NewQName("test", "document"))
			docDef.
				AddField("int32Field", appdef.DataKind_int32, true).
				AddField("int64Field", appdef.DataKind_int64, false).
				AddField("float32Field", appdef.DataKind_float32, false).
				AddField("float64Field", appdef.DataKind_float64, false).
				AddField("bytesField", appdef.DataKind_bytes, false).
				AddStringField("strField", false).
				AddField("qnameField", appdef.DataKind_QName, false).
				AddField("recIDField", appdef.DataKind_RecordID, false)
			docDef.
				AddContainer("child", appdef.NewQName("test", "record"), 1, appdef.Occurs_Unbounded)

			recDef := appDef.AddORecord(appdef.NewQName("test", "record"))
			recDef.
				AddField("int32Field", appdef.DataKind_int32, true).
				AddField("int64Field", appdef.DataKind_int64, false).
				AddField("float32Field", appdef.DataKind_float32, false).
				AddField("float64Field", appdef.DataKind_float64, false).
				AddField("bytesField", appdef.DataKind_bytes, false).
				AddStringField("strField", false).
				AddField("qnameField", appdef.DataKind_QName, false).
				AddField("boolField", appdef.DataKind_bool, false).
				AddField("recIDField", appdef.DataKind_RecordID, false)
		})
	})

	cfgs := make(AppConfigsType, 1)
	cfg := cfgs.AddConfig(test.appName, appDef)

	storage, err := simpleStorageProvider().AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)
	err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
	require.NoError(err)

	t.Run("test build object", func(t *testing.T) {
		t.Run("must error if null-name object", func(t *testing.T) {
			obj := func() istructs.IObjectBuilder {
				o := makeObject(cfg, appdef.NullQName)
				return &o
			}()
			_, err := obj.Build()
			require.ErrorIs(err, ErrNameMissed)
		})

		t.Run("must error if unknown-name object", func(t *testing.T) {
			obj := func() istructs.IObjectBuilder {
				o := makeObject(cfg, appdef.NewQName("test", "unknownDef"))
				return &o
			}()
			_, err := obj.Build()
			require.ErrorIs(err, ErrNameNotFound)
		})

		t.Run("must error if invalid definition kind object", func(t *testing.T) {
			obj := func() istructs.IObjectBuilder {
				o := makeObject(cfg, appdef.NewQName("test", "element"))
				return &o
			}()
			_, err := obj.Build()
			require.ErrorIs(err, ErrUnexpectedDefKind)
		})

		obj := func() istructs.IObjectBuilder {
			o := makeObject(cfg, appdef.NewQName("test", "object"))
			return &o
		}()

		t.Run("must error if empty object", func(t *testing.T) {
			_, err := obj.Build()
			require.ErrorIs(err, ErrNameNotFound)
		})

		obj.PutInt32("int32Field", 555)
		t.Run("must error if no nested child", func(t *testing.T) {
			_, err := obj.Build()
			require.ErrorIs(err, ErrMinOccursViolation)
		})

		child := obj.ElementBuilder("child")
		t.Run("must error if nested child has no required field", func(t *testing.T) {
			_, err := obj.Build()
			require.ErrorIs(err, ErrNameNotFound)
		})

		child.PutInt32("int32Field", 777)
		t.Run("must have no error if ok", func(t *testing.T) {
			_, err := obj.Build()
			require.NoError(err)
		})

		gChild := child.ElementBuilder("grandChild")
		require.NotNil(gChild)
		t.Run("must ok grand children", func(t *testing.T) {
			_, err := obj.Build()
			require.NoError(err)
		})

		t.Run("must error if unknown child name", func(t *testing.T) {
			gChild.PutString(appdef.SystemField_Container, "unknownName")
			_, err := obj.Build()
			require.ErrorIs(err, containers.ErrContainerNotFound)
		})
	})

	t.Run("test build operation document", func(t *testing.T) {
		doc := func() istructs.IObjectBuilder {
			d := makeObject(cfg, appdef.NewQName("test", "document"))
			return &d
		}()
		require.NotNil(doc)

		t.Run("must error if empty document", func(t *testing.T) {
			_, err := doc.Build()
			require.ErrorIs(err, ErrNameNotFound)
		})

		doc.PutRecordID(appdef.SystemField_ID, 1)
		doc.PutInt32("int32Field", 555)
		t.Run("must error if no nested document record", func(t *testing.T) {
			_, err := doc.Build()
			require.ErrorIs(err, ErrMinOccursViolation)
		})

		rec := doc.ElementBuilder("child")
		require.NotNil(rec)

		t.Run("must error if empty child record", func(t *testing.T) {
			_, err := doc.Build()
			require.ErrorIs(err, ErrNameNotFound)
		})

		rec.PutRecordID(appdef.SystemField_ID, 2)
		rec.PutInt32("int32Field", 555)

		t.Run("must error if wrong record parent", func(t *testing.T) {
			rec.PutRecordID(appdef.SystemField_ParentID, 77)
			_, err := doc.Build()
			require.ErrorIs(err, ErrWrongRecordID)
		})

		t.Run("must restore parent if empty record parent", func(t *testing.T) {
			rec.PutRecordID(appdef.SystemField_ParentID, istructs.NullRecordID)
			_, err := doc.Build()
			require.NoError(err)
		})
	})
}

func Test_ValidCUD(t *testing.T) {
	require := require.New(t)

	appDef := appdef.New()

	docName := appdef.NewQName("test", "document")
	rec1Name := appdef.NewQName("test", "record1")
	rec2Name := appdef.NewQName("test", "record2")

	objName := appdef.NewQName("test", "object")
	elemName := appdef.NewQName("test", "element")

	t.Run("must be ok to build test application definition", func(t *testing.T) {
		docDef := appDef.AddCDoc(docName)
		docDef.
			AddField("int32Field", appdef.DataKind_int32, true).
			AddField("int64Field", appdef.DataKind_int64, false).
			AddField("float32Field", appdef.DataKind_float32, false).
			AddField("float64Field", appdef.DataKind_float64, false).
			AddField("bytesField", appdef.DataKind_bytes, false).
			AddStringField("strField", false).
			AddField("qnameField", appdef.DataKind_QName, false).
			AddRefField("recIDField", false, rec1Name)
		docDef.
			AddContainer("child", rec1Name, 0, appdef.Occurs_Unbounded).
			AddContainer("childAgain", rec2Name, 0, appdef.Occurs_Unbounded)

		rec1Def := appDef.AddCRecord(rec1Name)
		rec1Def.
			AddField("int32Field", appdef.DataKind_int32, true).
			AddField("int64Field", appdef.DataKind_int64, false).
			AddField("float32Field", appdef.DataKind_float32, false).
			AddField("float64Field", appdef.DataKind_float64, false).
			AddField("bytesField", appdef.DataKind_bytes, false).
			AddStringField("strField", false).
			AddField("qnameField", appdef.DataKind_QName, false).
			AddField("boolField", appdef.DataKind_bool, false).
			AddRefField("recIDField", false, docName)

		_ = appDef.AddCRecord(rec2Name)

		objDef := appDef.AddObject(objName)
		objDef.
			AddField("int32Field", appdef.DataKind_int32, true).
			AddField("int64Field", appdef.DataKind_int64, false).
			AddField("float32Field", appdef.DataKind_float32, false).
			AddField("float64Field", appdef.DataKind_float64, false).
			AddField("bytesField", appdef.DataKind_bytes, false).
			AddStringField("strField", false).
			AddField("qnameField", appdef.DataKind_QName, false).
			AddRefField("recIDField", false, docName, rec1Name)
		objDef.
			AddContainer("childElement", elemName, 0, appdef.Occurs_Unbounded)

		_ = appDef.AddElement(elemName)
	})

	cfgs := make(AppConfigsType, 1)
	cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, appDef)

	storage, err := simpleStorageProvider().AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)
	err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
	require.NoError(err)

	t.Run("empty CUD must be valid", func(t *testing.T) {
		cud := makeCUD(cfg)
		err := cud.build()
		require.NoError(err)
		err = cfg.validators.validCUD(&cud, false)
		require.NoError(err)
	})

	t.Run("must error if empty CUD QName", func(t *testing.T) {
		cud := makeCUD(cfg)
		_ = cud.Create(appdef.NullQName)
		err := cud.build()
		require.NoError(err)
		err = cfg.validators.validCUD(&cud, false)
		require.ErrorIs(err, ErrNameMissed)
	})

	t.Run("must error if wrong CUD definition kind", func(t *testing.T) {
		cud := makeCUD(cfg)
		c := cud.Create(objName)
		c.PutInt32("int32Field", 7)
		err := cud.build()
		require.NoError(err)
		err = cfg.validators.validCUD(&cud, false)
		require.ErrorIs(err, ErrUnexpectedDefKind)
		require.ErrorContains(err, objName.String())
	})

	t.Run("test storage ID allow / disable in CUD.Create", func(t *testing.T) {
		cud := makeCUD(cfg)
		c := cud.Create(docName)
		c.PutRecordID(appdef.SystemField_ID, 100500)
		c.PutInt32("int32Field", 7)
		err := cud.build()
		require.NoError(err)
		t.Run("no error if storage IDs is enabled", func(t *testing.T) {
			err = cfg.validators.validCUD(&cud, true)
			require.NoError(err)
		})
		t.Run("must error if storage IDs is disabled", func(t *testing.T) {
			err = cfg.validators.validCUD(&cud, false)
			require.ErrorIs(err, ErrRawRecordIDExpected)
		})
	})

	t.Run("must error if raw ID duplication", func(t *testing.T) {
		cud := makeCUD(cfg)

		c1 := cud.Create(docName)
		c1.PutRecordID(appdef.SystemField_ID, 1)
		c1.PutInt32("int32Field", 7)

		c2 := cud.Create(docName)
		c2.PutRecordID(appdef.SystemField_ID, 1)
		c2.PutInt32("int32Field", 8)

		err := cud.build()
		require.NoError(err)

		err = cfg.validators.validCUD(&cud, false)
		require.ErrorIs(err, ErrRecordIDUniqueViolation)
	})

	t.Run("must error if invalid ID refs", func(t *testing.T) {

		t.Run("must error if unknown ID refs", func(t *testing.T) {
			cud := makeCUD(cfg)

			c1 := cud.Create(docName)
			c1.PutRecordID(appdef.SystemField_ID, 1)
			c1.PutInt32("int32Field", 7)

			c2 := cud.Create(rec1Name)
			c2.PutString(appdef.SystemField_Container, "child")
			c2.PutRecordID(appdef.SystemField_ID, 2)
			c2.PutRecordID(appdef.SystemField_ParentID, 7)
			c2.PutInt32("int32Field", 8)
			c2.PutRecordID("recIDField", 7)

			err := cud.build()
			require.NoError(err)

			err = cfg.validators.validCUD(&cud, false)
			require.ErrorIs(err, ErrorRecordIDNotFound)
		})

		t.Run("must error if ID refs to invalid QName", func(t *testing.T) {
			cud := makeCUD(cfg)

			c1 := cud.Create(docName)
			c1.PutRecordID(appdef.SystemField_ID, 1)
			c1.PutInt32("int32Field", 7)
			c1.PutRecordID("recIDField", 1)

			err := cud.build()
			require.NoError(err)

			err = cfg.validators.validCUD(&cud, false)
			require.ErrorIs(err, ErrWrongRecordID)
			require.ErrorContains(err, "unavailable target QName «test.document»")
		})

		t.Run("must error if ParentID causes invalid references", func(t *testing.T) {

			t.Run("must error if container unknown for specified ParentID", func(t *testing.T) {
				cud := makeCUD(cfg)

				c1 := cud.Create(docName)
				c1.PutRecordID(appdef.SystemField_ID, 1)
				c1.PutInt32("int32Field", 7)

				c2 := cud.Create(rec1Name)
				c2.PutString(appdef.SystemField_Container, "childElement")
				c2.PutRecordID(appdef.SystemField_ID, 2)
				c2.PutRecordID(appdef.SystemField_ParentID, 1)
				c2.PutInt32("int32Field", 7)

				err := cud.build()
				require.NoError(err)

				err = cfg.validators.validCUD(&cud, false)
				require.ErrorIs(err, ErrWrongRecordID)
				require.ErrorContains(err, "has no container «childElement»")
			})

			t.Run("must error if specified container has another QName", func(t *testing.T) {
				cud := makeCUD(cfg)

				c1 := cud.Create(docName)
				c1.PutRecordID(appdef.SystemField_ID, 1)
				c1.PutInt32("int32Field", 7)

				c2 := cud.Create(rec1Name)
				c2.PutString(appdef.SystemField_Container, "childAgain")
				c2.PutRecordID(appdef.SystemField_ID, 2)
				c2.PutRecordID(appdef.SystemField_ParentID, 1)
				c2.PutInt32("int32Field", 7)

				err := cud.build()
				require.NoError(err)

				err = cfg.validators.validCUD(&cud, false)
				require.ErrorIs(err, ErrWrongRecordID)
				require.ErrorContains(err, "container «childAgain», which has another QName «test.record2»")
			})
		})
	})
}

func Test_VerifiedFields(t *testing.T) {
	require := require.New(t)
	test := test()

	objName := appdef.NewQName("test", "obj")

	appDef := appdef.New()
	t.Run("must be ok to build application definition", func(t *testing.T) {
		def := appDef.AddObject(objName)
		def.
			AddField("int32", appdef.DataKind_int32, true).
			AddStringField("email", false).
			SetFieldVerify("email", appdef.VerificationKind_EMail).
			AddField("age", appdef.DataKind_int32, false).
			SetFieldVerify("age", appdef.VerificationKind_Any...)
	})

	cfgs := make(AppConfigsType, 1)
	cfg := cfgs.AddConfig(test.appName, appDef)

	email := "test@test.io"

	tokens := testTokensFactory().New(test.appName)
	storage, err := simpleStorageProvider().AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)
	asp := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())
	err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
	require.NoError(err)
	_, err = asp.AppStructs(test.appName) // need to set cfg.app because IAppTokens are taken from cfg.app
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

			row := makeObject(cfg, objName)
			row.PutInt32("int32", 1)
			row.PutString("email", okEmailToken)
			row.PutString("age", okAgeToken)

			_, err := row.Build()
			require.NoError(err)
		})

		t.Run("error if not token, but not string value", func(t *testing.T) {

			row := makeObject(cfg, objName)
			row.PutInt32("int32", 1)
			row.PutInt32("age", 7)

			_, err := row.Build()
			require.ErrorIs(err, ErrWrongFieldType)
		})

		t.Run("error if not a token, but plain string value", func(t *testing.T) {

			row := makeObject(cfg, objName)
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

			row := makeObject(cfg, objName)
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

			row := makeObject(cfg, objName)
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

			row := makeObject(cfg, objName)
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

			row := makeObject(cfg, objName)
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

	appDef := appdef.New()
	t.Run("must be ok to build application definition", func(t *testing.T) {
		def := appDef.AddObject(objName)
		def.
			AddStringField("email", true, appdef.MinLen(6), appdef.MaxLen(100), appdef.Pattern(`^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$`)).
			AddBytesField("mime", false, appdef.MinLen(4), appdef.MaxLen(4), appdef.Pattern(`^\w+$`))
	})

	cfgs := make(AppConfigsType, 1)
	cfg := cfgs.AddConfig(test.appName, appDef)

	storage, err := simpleStorageProvider().AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)
	asp := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())
	err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
	require.NoError(err)
	_, err = asp.AppStructs(test.appName)
	require.NoError(err)

	t.Run("test field restricts", func(t *testing.T) {

		t.Run("must be ok check good value", func(t *testing.T) {
			row := makeObject(cfg, objName)
			row.PutString("email", `test@test.io`)
			row.PutBytes("mime", []byte(`abcd`))

			_, err := row.Build()
			require.NoError(err)
		})

		t.Run("must be error if min length restricted", func(t *testing.T) {
			row := makeObject(cfg, objName)
			row.PutString("email", `t@t`)
			row.PutBytes("mime", []byte(`abc`))

			_, err := row.Build()
			require.ErrorIs(err, ErrFieldValueRestricted)
			require.ErrorContains(err, "field «email» is too short")
			require.ErrorContains(err, "field «mime» is too short")
		})

		t.Run("must be error if max length restricted", func(t *testing.T) {
			row := makeObject(cfg, objName)
			row.PutString("email", fmt.Sprintf("%s.com", strings.Repeat("test", 100)))
			row.PutBytes("mime", []byte(`abcde`))

			_, err := row.Build()
			require.ErrorIs(err, ErrFieldValueRestricted)
			require.ErrorContains(err, "field «email» is too long")
			require.ErrorContains(err, "field «mime» is too long")
		})

		t.Run("must be error if pattern restricted", func(t *testing.T) {
			row := makeObject(cfg, objName)
			row.PutString("email", "naked@🔫.error")
			row.PutBytes("mime", []byte(`++++`))

			_, err := row.Build()
			require.ErrorIs(err, ErrFieldValueRestricted)
			require.ErrorContains(err, "field «email» does not match pattern")
			require.ErrorContains(err, "field «mime» does not match pattern")
		})
	})
}

func Test_ValidateErrors(t *testing.T) {
	require := require.New(t)
	test := test()

	provider := Provide(test.AppConfigs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

	app, err := provider.AppStructs(test.appName)
	require.NoError(err)

	t.Run("ECode_EmptyDefName", func(t *testing.T) {
		bld := app.Events().GetSyncRawEventBuilder(
			istructs.SyncRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: test.partition,
					PLogOffset:        test.plogOfs,
					Workspace:         test.workspace,
					WLogOffset:        test.wlogOfs,
					QName:             appdef.NullQName,
					RegisteredAt:      test.registeredTime,
				},
				Device:   test.device,
				SyncedAt: test.syncTime,
			})
		_, buildErr := bld.BuildRawEvent()
		require.ErrorIs(buildErr, ErrNameMissed)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(buildErr, &validateErr)
		require.Equal(ECode_EmptyDefName, validateErr.Code())
	})

	t.Run("ECode_InvalidDefName", func(t *testing.T) {
		bld := app.Events().GetSyncRawEventBuilder(
			istructs.SyncRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: test.partition,
					PLogOffset:        test.plogOfs,
					Workspace:         test.workspace,
					WLogOffset:        test.wlogOfs,
					QName:             test.testCDoc,
					RegisteredAt:      test.registeredTime,
				},
				Device:   test.device,
				SyncedAt: test.syncTime,
			})
		_, buildErr := bld.BuildRawEvent()
		require.ErrorIs(buildErr, ErrNameNotFound)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(buildErr, &validateErr)
		require.Equal(ECode_InvalidDefName, validateErr.Code())
	})

	t.Run("ECode_InvalidDefKind", func(t *testing.T) {
		var app istructs.IAppStructs

		cDocName := appdef.NewQName("test", "CDoc")
		cmdCreateDoc := appdef.NewQName("test", "CreateDoc")

		t.Run("builds application", func(t *testing.T) {
			appDef := appdef.New()

			t.Run("must be ok to build application definition", func(t *testing.T) {
				cDocDef := appDef.AddCDoc(cDocName)
				cDocDef.AddField("Int32", appdef.DataKind_int32, false)
			})

			cfgs := make(AppConfigsType, 1)
			cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, appDef)
			cfg.Resources.Add(NewCommandFunction(cmdCreateDoc, cDocName, appdef.NullQName, appdef.NullQName, NullCommandExec))

			storage, err := simpleStorageProvider().AppStorage(istructs.AppQName_test1_app1)
			require.NoError(err)
			err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
			require.NoError(err)

			provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

			app, err = provider.AppStructs(istructs.AppQName_test1_app1)
			require.NoError(err)
		})

		bld := app.Events().GetSyncRawEventBuilder(
			istructs.SyncRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: test.partition,
					PLogOffset:        test.plogOfs,
					Workspace:         test.workspace,
					WLogOffset:        test.wlogOfs,
					QName:             cmdCreateDoc,
					RegisteredAt:      test.registeredTime,
				},
				Device:   test.device,
				SyncedAt: test.syncTime,
			})
		_, buildErr := bld.BuildRawEvent()
		require.ErrorIs(buildErr, ErrWrongDefinition)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(buildErr, &validateErr)
		require.Equal(ECode_InvalidDefKind, validateErr.Code())
	})

	t.Run("ECode_EmptyFieldData", func(t *testing.T) {
		bld := app.Events().GetSyncRawEventBuilder(
			istructs.SyncRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: test.partition,
					PLogOffset:        test.plogOfs,
					Workspace:         test.workspace,
					WLogOffset:        test.wlogOfs,
					QName:             test.saleCmdName,
					RegisteredAt:      test.registeredTime,
				},
				Device:   test.device,
				SyncedAt: test.syncTime,
			})
		_, buildErr := bld.BuildRawEvent()
		require.ErrorIs(buildErr, ErrNameNotFound)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(buildErr, &validateErr)
		require.Equal(ECode_EmptyData, validateErr.Code())
	})

	t.Run("ECode_InvalidRawRecordID", func(t *testing.T) {
		bld := app.Events().GetSyncRawEventBuilder(
			istructs.SyncRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: test.partition,
					PLogOffset:        test.plogOfs,
					Workspace:         test.workspace,
					WLogOffset:        test.wlogOfs,
					QName:             test.saleCmdName,
					RegisteredAt:      test.registeredTime,
				},
				Device:   test.device,
				SyncedAt: test.syncTime,
			})
		cmd := bld.ArgumentObjectBuilder()
		cmd.PutRecordID(appdef.SystemField_ID, 100500100500)
		cmd.PutString(test.buyerIdent, test.buyerValue)

		_, buildErr := bld.BuildRawEvent()
		require.ErrorIs(buildErr, ErrRawRecordIDExpected)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(buildErr, &validateErr)
		require.Equal(ECode_InvalidRawRecordID, validateErr.Code())
	})

	t.Run("ECode_InvalidRecordID", func(t *testing.T) {
		bld := app.Events().GetSyncRawEventBuilder(
			istructs.SyncRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: test.partition,
					PLogOffset:        test.plogOfs,
					Workspace:         test.workspace,
					WLogOffset:        test.wlogOfs,
					QName:             test.changeCmdName,
					RegisteredAt:      test.registeredTime,
				},
				Device:   test.device,
				SyncedAt: test.syncTime,
			})

		cud := bld.CUDBuilder()
		newRec := cud.Create(test.testCDoc)
		newRec.PutRecordID(appdef.SystemField_ID, 1)
		r := newTestCDoc(1)
		_ = cud.Update(r)

		_, buildErr := bld.BuildRawEvent()
		require.ErrorIs(buildErr, ErrRecordIDUniqueViolation)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(buildErr, &validateErr)
		require.Equal(ECode_InvalidRecordID, validateErr.Code())
	})

	t.Run("ECode_InvalidRefRecordID", func(t *testing.T) {
		eventBuilder := func() istructs.IRawEventBuilder {
			return app.Events().GetSyncRawEventBuilder(
				istructs.SyncRawEventBuilderParams{
					GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
						HandlingPartition: test.partition,
						PLogOffset:        test.plogOfs,
						Workspace:         test.workspace,
						WLogOffset:        test.wlogOfs,
						QName:             test.changeCmdName,
						RegisteredAt:      test.registeredTime,
					},
					Device:   test.device,
					SyncedAt: test.syncTime,
				})
		}
		t.Run("no record to update", func(t *testing.T) {
			bld := eventBuilder()
			cud := bld.CUDBuilder()
			r := newTestCDoc(7)
			_ = cud.Update(r)

			_, buildErr := bld.BuildRawEvent()
			require.ErrorIs(buildErr, ErrorRecordIDNotFound)
			validateErr := validateErrorf(0, "")
			require.ErrorAs(buildErr, &validateErr)
			require.Equal(ECode_InvalidRefRecordID, validateErr.Code())
		})
		t.Run("0 value ref field", func(t *testing.T) {
			bld := eventBuilder()
			icud := bld.CUDBuilder()
			rec := icud.Create(test.tablePhotos)
			rec.PutRecordID(appdef.SystemField_ID, test.tempPhotoID)
			rec.PutString(test.buyerIdent, test.buyerValue)

			recRem := icud.Create(test.tablePhotoRems)
			recRem.PutRecordID(appdef.SystemField_ID, test.tempRemarkID)
			recRem.PutRecordID(appdef.SystemField_ParentID, test.tempPhotoID)
			recRem.PutString(appdef.SystemField_Container, test.remarkIdent)
			recRem.PutRecordID(test.photoIdent, istructs.NullRecordID) // 0 value in not null ref field here
			recRem.PutString(test.remarkIdent, test.remarkValue)
			_, buildErr := bld.BuildRawEvent()
			require.ErrorIs(buildErr, ErrWrongRecordID)
			validateErr := validateErrorf(0, "")
			require.ErrorAs(buildErr, &validateErr)
			require.Equal(ECode_InvalidRefRecordID, validateErr.Code())
		})
	})

	t.Run("ECode_EEmptyCUDs", func(t *testing.T) {
		bld := app.Events().GetSyncRawEventBuilder(
			istructs.SyncRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: test.partition,
					PLogOffset:        test.plogOfs,
					Workspace:         test.workspace,
					WLogOffset:        test.wlogOfs,
					QName:             istructs.QNameCommandCUD,
					RegisteredAt:      test.registeredTime,
				},
				Device:   test.device,
				SyncedAt: test.syncTime,
			})
		_, buildErr := bld.BuildRawEvent()
		require.ErrorIs(buildErr, ErrCUDsMissed)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(buildErr, &validateErr)
		require.Equal(ECode_EEmptyCUDs, validateErr.Code())
	})

	t.Run("ECode_EmptyElementName", func(t *testing.T) {
		bld := app.Events().GetSyncRawEventBuilder(
			istructs.SyncRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: test.partition,
					PLogOffset:        test.plogOfs,
					Workspace:         test.workspace,
					WLogOffset:        test.wlogOfs,
					QName:             test.saleCmdName,
					RegisteredAt:      test.registeredTime,
				},
				Device:   test.device,
				SyncedAt: test.syncTime,
			})
		cmd := bld.ArgumentObjectBuilder()
		cmd.PutRecordID(appdef.SystemField_ID, 1)
		cmd.PutString(test.buyerIdent, test.buyerValue)
		bsk := cmd.ElementBuilder(test.basketIdent)
		bsk.PutRecordID(appdef.SystemField_ID, 2)
		_ = bsk.ElementBuilder("")

		_, buildErr := bld.BuildRawEvent()
		require.ErrorIs(buildErr, ErrNameMissed)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(buildErr, &validateErr)
		require.Equal(ECode_EmptyElementName, validateErr.Code())
	})

	t.Run("ECode_InvalidElementName", func(t *testing.T) {
		bld := app.Events().GetSyncRawEventBuilder(
			istructs.SyncRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: test.partition,
					PLogOffset:        test.plogOfs,
					Workspace:         test.workspace,
					WLogOffset:        test.wlogOfs,
					QName:             test.saleCmdName,
					RegisteredAt:      test.registeredTime,
				},
				Device:   test.device,
				SyncedAt: test.syncTime,
			})
		cmd := bld.ArgumentObjectBuilder()
		cmd.PutRecordID(appdef.SystemField_ID, 1)
		cmd.PutString(test.buyerIdent, test.buyerValue)
		bsk := cmd.ElementBuilder(test.basketIdent)
		bsk.PutRecordID(appdef.SystemField_ID, 2)
		good := bsk.ElementBuilder(test.goodIdent)
		good.PutString(appdef.SystemField_Container, test.basketIdent) // error here

		_, buildErr := bld.BuildRawEvent()
		require.ErrorIs(buildErr, ErrNameNotFound)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(buildErr, &validateErr)
		require.Equal(ECode_InvalidElementName, validateErr.Code())
	})

	t.Run("ECode_InvalidOccursMin", func(t *testing.T) {
		bld := app.Events().GetSyncRawEventBuilder(
			istructs.SyncRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: test.partition,
					PLogOffset:        test.plogOfs,
					Workspace:         test.workspace,
					WLogOffset:        test.wlogOfs,
					QName:             test.saleCmdName,
					RegisteredAt:      test.registeredTime,
				},
				Device:   test.device,
				SyncedAt: test.syncTime,
			})
		cmd := bld.ArgumentObjectBuilder()
		cmd.PutRecordID(appdef.SystemField_ID, 1)
		cmd.PutString(test.buyerIdent, test.buyerValue)

		_, buildErr := bld.BuildRawEvent()
		require.Error(buildErr)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(buildErr, &validateErr)
		require.Equal(ECode_InvalidOccursMin, validateErr.Code())
	})

	t.Run("ECode_InvalidOccursMax", func(t *testing.T) {
		bld := app.Events().GetSyncRawEventBuilder(
			istructs.SyncRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: test.partition,
					PLogOffset:        test.plogOfs,
					Workspace:         test.workspace,
					WLogOffset:        test.wlogOfs,
					QName:             test.saleCmdName,
					RegisteredAt:      test.registeredTime,
				},
				Device:   test.device,
				SyncedAt: test.syncTime,
			})
		cmd := bld.ArgumentObjectBuilder()
		cmd.PutRecordID(appdef.SystemField_ID, 1)
		cmd.PutString(test.buyerIdent, test.buyerValue)

		bsk := cmd.ElementBuilder(test.basketIdent)
		bsk.PutRecordID(appdef.SystemField_ID, 2)

		_ = cmd.ElementBuilder(test.basketIdent)

		_, buildErr := bld.BuildRawEvent()
		require.Error(buildErr)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(buildErr, &validateErr)
		require.Equal(ECode_InvalidOccursMax, validateErr.Code())
	})
}
