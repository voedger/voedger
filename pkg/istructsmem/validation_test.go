/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/containers"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/schemas"
)

func Test_ValidEvent(t *testing.T) {
	require := require.New(t)

	var (
		app istructs.IAppStructs

		cmdCreateDoc schemas.QName = schemas.NewQName("test", "CreateDoc")
		cDocName     schemas.QName = schemas.NewQName("test", "CDoc")
		oDocName     schemas.QName = schemas.NewQName("test", "ODoc")

		cmdCreateObj         schemas.QName = schemas.NewQName("test", "CreateObj")
		cmdCreateObjUnlogged schemas.QName = schemas.NewQName("test", "CreateObjUnlogged")
		oObjName             schemas.QName = schemas.NewQName("test", "Object")

		cmdCUD schemas.QName = schemas.NewQName("test", "cudEvent")
	)

	t.Run("builds app", func(t *testing.T) {
		bld := schemas.NewSchemaCache()

		t.Run("must be ok to build schemas", func(t *testing.T) {
			CDocSchema := bld.Add(cDocName, schemas.SchemaKind_CDoc)
			CDocSchema.
				AddField("Int32", schemas.DataKind_int32, true).
				AddField("String", schemas.DataKind_string, false)

			ODocSchema := bld.Add(oDocName, schemas.SchemaKind_ODoc)
			ODocSchema.
				AddField("Int32", schemas.DataKind_int32, true).
				AddField("String", schemas.DataKind_string, false).
				AddContainer("child", oDocName, 0, 2) // ODocs should be able to contain ODocs, see #!19332

			ObjSchema := bld.Add(oObjName, schemas.SchemaKind_Object)
			ObjSchema.
				AddField("Int32", schemas.DataKind_int32, true).
				AddField("String", schemas.DataKind_string, false)
		})

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, bld)
		cfg.Resources.Add(NewCommandFunction(cmdCreateDoc, cDocName, schemas.NullQName, schemas.NullQName, NullCommandExec))
		cfg.Resources.Add(NewCommandFunction(cmdCreateObj, oObjName, schemas.NullQName, schemas.NullQName, NullCommandExec))
		cfg.Resources.Add(NewCommandFunction(cmdCreateObjUnlogged, schemas.NullQName, oObjName, schemas.NullQName, NullCommandExec))
		cfg.Resources.Add(NewCommandFunction(cmdCUD, schemas.NullQName, schemas.NullQName, schemas.NullQName, NullCommandExec))

		storage, err := simpleStorageProvder().AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)
		err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
		require.NoError(err)

		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvder())

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
					QName:             schemas.NullQName,
					RegisteredAt:      123456789,
				},
			})
		require.NotNil(bld)

		_, err := bld.BuildRawEvent()
		require.ErrorIs(err, ErrNameMissed)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(err, &validateErr)
		require.Equal(ECode_EmptySchemaName, validateErr.Code())
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
		cmd.PutQName(schemas.SystemField_QName, oObjName)

		_, err := bld.BuildRawEvent()
		require.ErrorIs(err, ErrWrongSchema)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(err, &validateErr)
		require.Equal(ECode_InvalidSchemaName, validateErr.Code())
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

		cmd.PutRecordID(schemas.SystemField_ID, 1)

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

		t.Run("must failed build raw event (unexpected argument schema)", func(t *testing.T) {
			cmd.PutQName(schemas.SystemField_QName, oDocName)
			rawEvent, err := bld.BuildRawEvent()
			require.ErrorIs(err, ErrSchemaChanged) // expected «test.Object», but not «test.ODoc»
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
		cmd.PutRecordID(schemas.SystemField_ID, 1)
		cmd.PutInt32("Int32", 29)
		cmd.PutString("String", "string data")

		t.Run("must failed build raw event", func(t *testing.T) {
			rawEvent, err := bld.BuildRawEvent()
			require.ErrorIs(err, ErrWrongSchema) // CDoc deprecated, ODoc or Object expected
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
		cmd.PutRecordID(schemas.SystemField_ID, 1)
		cmd.PutInt32("Int32", 29)
		cmd.PutString("String", "string data")

		child := cmd.ElementBuilder("child")
		child.PutRecordID(schemas.SystemField_ID, 2)
		child.PutInt32("Int32", 29)
		child.PutString("String", "string data")

		t.Run("test ok build raw event", func(t *testing.T) {
			rawEvent, err := bld.BuildRawEvent()
			require.NoError(err)
			require.NotNil(rawEvent)

			cmd := rawEvent.ArgumentObject()
			require.Equal(istructs.RecordID(1), cmd.AsRecordID(schemas.SystemField_ID))
			require.Equal(int32(29), cmd.AsInt32("Int32"))
			require.Equal("string data", cmd.AsString("String"))

			cnt := 0
			cmd.Elements("child", func(child istructs.IElement) {
				require.Equal(istructs.RecordID(2), child.AsRecordID(schemas.SystemField_ID))
				require.Equal(istructs.RecordID(1), child.AsRecordID(schemas.SystemField_ParentID))
				require.Equal("child", child.AsString(schemas.SystemField_Container))
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

	bld := schemas.NewSchemaCache()

	t.Run("must be ok to build test schemas", func(t *testing.T) {

		t.Run("build object schemas", func(t *testing.T) {
			objSchema := bld.Add(schemas.NewQName("test", "object"), schemas.SchemaKind_Object)
			objSchema.
				AddField("int32Field", schemas.DataKind_int32, true).
				AddField("int64Field", schemas.DataKind_int64, false).
				AddField("float32Field", schemas.DataKind_float32, false).
				AddField("float64Field", schemas.DataKind_float64, false).
				AddField("bytesField", schemas.DataKind_bytes, false).
				AddField("strField", schemas.DataKind_string, false).
				AddField("qnameField", schemas.DataKind_QName, false).
				AddField("recIDField", schemas.DataKind_RecordID, false).
				AddContainer("child", schemas.NewQName("test", "element"), 1, schemas.Occurs_Unbounded)

			elementSchema := bld.Add(schemas.NewQName("test", "element"), schemas.SchemaKind_Element)
			elementSchema.
				AddField("int32Field", schemas.DataKind_int32, true).
				AddField("int64Field", schemas.DataKind_int64, false).
				AddField("float32Field", schemas.DataKind_float32, false).
				AddField("float64Field", schemas.DataKind_float64, false).
				AddField("bytesField", schemas.DataKind_bytes, false).
				AddField("strField", schemas.DataKind_string, false).
				AddField("qnameField", schemas.DataKind_QName, false).
				AddField("boolField", schemas.DataKind_bool, false).
				AddField("recIDField", schemas.DataKind_RecordID, false).
				AddContainer("grandChild", schemas.NewQName("test", "grandChild"), 0, 1)

			subElementSchema := bld.Add(schemas.NewQName("test", "grandChild"), schemas.SchemaKind_Element)
			subElementSchema.
				AddField("recIDField", schemas.DataKind_RecordID, false)
		})

		t.Run("build ODoc schemas", func(t *testing.T) {
			docSchema := bld.Add(schemas.NewQName("test", "document"), schemas.SchemaKind_ODoc)
			docSchema.
				AddField("int32Field", schemas.DataKind_int32, true).
				AddField("int64Field", schemas.DataKind_int64, false).
				AddField("float32Field", schemas.DataKind_float32, false).
				AddField("float64Field", schemas.DataKind_float64, false).
				AddField("bytesField", schemas.DataKind_bytes, false).
				AddField("strField", schemas.DataKind_string, false).
				AddField("qnameField", schemas.DataKind_QName, false).
				AddField("recIDField", schemas.DataKind_RecordID, false).
				AddContainer("child", schemas.NewQName("test", "record"), 1, schemas.Occurs_Unbounded)

			recordSchema := bld.Add(schemas.NewQName("test", "record"), schemas.SchemaKind_ORecord)
			recordSchema.
				AddField("int32Field", schemas.DataKind_int32, true).
				AddField("int64Field", schemas.DataKind_int64, false).
				AddField("float32Field", schemas.DataKind_float32, false).
				AddField("float64Field", schemas.DataKind_float64, false).
				AddField("bytesField", schemas.DataKind_bytes, false).
				AddField("strField", schemas.DataKind_string, false).
				AddField("qnameField", schemas.DataKind_QName, false).
				AddField("boolField", schemas.DataKind_bool, false).
				AddField("recIDField", schemas.DataKind_RecordID, false)
		})
	})

	cfgs := make(AppConfigsType, 1)
	cfg := cfgs.AddConfig(test.appName, bld)

	storage, err := simpleStorageProvder().AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)
	err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
	require.NoError(err)

	t.Run("test build object", func(t *testing.T) {
		t.Run("must error if null-name object", func(t *testing.T) {
			obj := func() istructs.IObjectBuilder {
				o := newObject(cfg, schemas.NullQName)
				return &o
			}()
			_, err := obj.Build()
			require.ErrorIs(err, ErrNameMissed)
		})

		t.Run("must error if unknown-name object", func(t *testing.T) {
			obj := func() istructs.IObjectBuilder {
				o := newObject(cfg, schemas.NewQName("test", "unknownSchema"))
				return &o
			}()
			_, err := obj.Build()
			require.ErrorIs(err, ErrNameNotFound)
		})

		t.Run("must error if invalid schema kind object", func(t *testing.T) {
			obj := func() istructs.IObjectBuilder {
				o := newObject(cfg, schemas.NewQName("test", "element"))
				return &o
			}()
			_, err := obj.Build()
			require.ErrorIs(err, ErrUnexpectedShemaKind)
		})

		obj := func() istructs.IObjectBuilder {
			o := newObject(cfg, schemas.NewQName("test", "object"))
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
		t.Run("must error if nested child has no requiered field", func(t *testing.T) {
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
		t.Run("must ok grand childs", func(t *testing.T) {
			_, err := obj.Build()
			require.NoError(err)
		})

		t.Run("must error if unknown child name", func(t *testing.T) {
			gChild.PutString(schemas.SystemField_Container, "unknownName")
			_, err := obj.Build()
			require.ErrorIs(err, containers.ErrContainerNotFound)
		})
	})

	t.Run("test build operation document", func(t *testing.T) {
		doc := func() istructs.IObjectBuilder {
			d := newObject(cfg, schemas.NewQName("test", "document"))
			return &d
		}()
		require.NotNil(doc)

		t.Run("must error if empty document", func(t *testing.T) {
			_, err := doc.Build()
			require.ErrorIs(err, ErrNameNotFound)
		})

		doc.PutRecordID(schemas.SystemField_ID, 1)
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

		rec.PutRecordID(schemas.SystemField_ID, 2)
		rec.PutInt32("int32Field", 555)

		t.Run("must error if wrong record parent", func(t *testing.T) {
			rec.PutRecordID(schemas.SystemField_ParentID, 77)
			_, err := doc.Build()
			require.ErrorIs(err, ErrWrongRecordID)
		})

		t.Run("must restore parent if empty record parent", func(t *testing.T) {
			rec.PutRecordID(schemas.SystemField_ParentID, istructs.NullRecordID)
			_, err := doc.Build()
			require.NoError(err)
		})
	})
}

func Test_ValidCUD(t *testing.T) {
	require := require.New(t)

	bld := schemas.NewSchemaCache()

	t.Run("must be ok to build test schemas", func(t *testing.T) {
		t.Run("build CDoc schemas", func(t *testing.T) {
			docSchema := bld.Add(schemas.NewQName("test", "document"), schemas.SchemaKind_CDoc)
			docSchema.
				AddField("int32Field", schemas.DataKind_int32, true).
				AddField("int64Field", schemas.DataKind_int64, false).
				AddField("float32Field", schemas.DataKind_float32, false).
				AddField("float64Field", schemas.DataKind_float64, false).
				AddField("bytesField", schemas.DataKind_bytes, false).
				AddField("strField", schemas.DataKind_string, false).
				AddField("qnameField", schemas.DataKind_QName, false).
				AddField("recIDField", schemas.DataKind_RecordID, false).
				AddContainer("child", schemas.NewQName("test", "record"), 1, schemas.Occurs_Unbounded)

			recordSchema := bld.Add(schemas.NewQName("test", "record"), schemas.SchemaKind_CRecord)
			recordSchema.
				AddField("int32Field", schemas.DataKind_int32, true).
				AddField("int64Field", schemas.DataKind_int64, false).
				AddField("float32Field", schemas.DataKind_float32, false).
				AddField("float64Field", schemas.DataKind_float64, false).
				AddField("bytesField", schemas.DataKind_bytes, false).
				AddField("strField", schemas.DataKind_string, false).
				AddField("qnameField", schemas.DataKind_QName, false).
				AddField("boolField", schemas.DataKind_bool, false).
				AddField("recIDField", schemas.DataKind_RecordID, false)

			objSchema := bld.Add(schemas.NewQName("test", "object"), schemas.SchemaKind_Object)
			objSchema.
				AddField("int32Field", schemas.DataKind_int32, true).
				AddField("int64Field", schemas.DataKind_int64, false).
				AddField("float32Field", schemas.DataKind_float32, false).
				AddField("float64Field", schemas.DataKind_float64, false).
				AddField("bytesField", schemas.DataKind_bytes, false).
				AddField("strField", schemas.DataKind_string, false).
				AddField("qnameField", schemas.DataKind_QName, false).
				AddField("recIDField", schemas.DataKind_RecordID, false)
		})
	})

	cfgs := make(AppConfigsType, 1)
	cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, bld)

	storage, err := simpleStorageProvder().AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)
	err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
	require.NoError(err)

	t.Run("empty CUD must be valid", func(t *testing.T) {
		cud := newCUD(cfg)
		err := cud.build()
		require.NoError(err)
		err = cfg.validators.validCUD(&cud, false)
		require.NoError(err)
	})

	t.Run("must error if empty CUD QName", func(t *testing.T) {
		cud := newCUD(cfg)
		_ = cud.Create(schemas.NullQName)
		err := cud.build()
		require.NoError(err)
		err = cfg.validators.validCUD(&cud, false)
		require.ErrorIs(err, ErrNameMissed)
	})

	t.Run("must error if wrong CUD schema kind", func(t *testing.T) {
		cud := newCUD(cfg)
		c := cud.Create(schemas.NewQName("test", "object"))
		c.PutInt32("int32Field", 7)
		err := cud.build()
		require.NoError(err)
		err = cfg.validators.validCUD(&cud, false)
		require.ErrorIs(err, ErrUnexpectedShemaKind)
	})

	t.Run("test storage ID allow / disable in CUD.Create", func(t *testing.T) {
		cud := newCUD(cfg)
		c := cud.Create(schemas.NewQName("test", "document"))
		c.PutRecordID(schemas.SystemField_ID, 100500)
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
		cud := newCUD(cfg)

		c1 := cud.Create(schemas.NewQName("test", "document"))
		c1.PutRecordID(schemas.SystemField_ID, 1)
		c1.PutInt32("int32Field", 7)

		c2 := cud.Create(schemas.NewQName("test", "document"))
		c2.PutRecordID(schemas.SystemField_ID, 1)
		c2.PutInt32("int32Field", 8)

		err := cud.build()
		require.NoError(err)

		err = cfg.validators.validCUD(&cud, false)
		require.ErrorIs(err, ErrRecordIDUniqueViolation)
	})

	t.Run("must error if invalid ID refs", func(t *testing.T) {
		cud := newCUD(cfg)

		c1 := cud.Create(schemas.NewQName("test", "document"))
		c1.PutRecordID(schemas.SystemField_ID, 1)
		c1.PutInt32("int32Field", 7)

		c2 := cud.Create(schemas.NewQName("test", "record"))
		c2.PutString(schemas.SystemField_Container, "child")
		c2.PutRecordID(schemas.SystemField_ID, 2)
		c2.PutRecordID(schemas.SystemField_ParentID, 7)
		c2.PutInt32("int32Field", 8)
		c2.PutRecordID("recIDField", 7)

		err := cud.build()
		require.NoError(err)

		err = cfg.validators.validCUD(&cud, false)
		require.ErrorIs(err, ErrorRecordIDNotFound)
	})
}

func Test_VerifiedFields(t *testing.T) {
	require := require.New(t)
	test := test()

	objName := schemas.NewQName("test", "Schema")

	bld := schemas.NewSchemaCache()
	t.Run("must be ok to build schemas", func(t *testing.T) {
		schema := bld.Add(objName, schemas.SchemaKind_Object)
		schema.
			AddField("int32", schemas.DataKind_int32, true).
			AddVerifiedField("email", schemas.DataKind_string, false, schemas.VerificationKind_EMail).
			AddVerifiedField("age", schemas.DataKind_int32, false, schemas.VerificationKind_Any...)
	})

	cfgs := make(AppConfigsType, 1)
	cfg := cfgs.AddConfig(test.appName, bld)

	email := "test@test.io"

	tokens := testTokensFactory().New(test.appName)
	storage, err := simpleStorageProvder().AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)
	asp := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvder())
	err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
	require.NoError(err)
	_, err = asp.AppStructs(test.appName) // need to set cfg.app because IAppTokens are taken from cfg.app
	require.NoError(err)

	t.Run("test row verification", func(t *testing.T) {

		t.Run("ok verified value type in token", func(t *testing.T) {
			okEmailToken := func() string {
				p := payloads.VerifiedValuePayload{
					VerificationKind: schemas.VerificationKind_EMail,
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
					VerificationKind: schemas.VerificationKind_Phone,
					Entity:           objName,
					Field:            "age",
					Value:            7,
				}
				token, err := tokens.IssueToken(time.Minute, &p)
				require.NoError(err)
				return token
			}()

			row := newObject(cfg, objName)
			row.PutInt32("int32", 1)
			row.PutString("email", okEmailToken)
			row.PutString("age", okAgeToken)

			_, err := row.Build()
			require.NoError(err)
		})

		t.Run("error if not token, but not string value", func(t *testing.T) {

			row := newObject(cfg, objName)
			row.PutInt32("int32", 1)
			row.PutInt32("age", 7)

			_, err := row.Build()
			require.ErrorIs(err, ErrWrongFieldType)
		})

		t.Run("error if not a token, but plain string value", func(t *testing.T) {

			row := newObject(cfg, objName)
			row.PutInt32("int32", 1)
			row.PutString("email", email)

			_, err := row.Build()
			require.ErrorIs(err, itokens.ErrInvalidToken)
		})

		t.Run("error if unexpected token kind", func(t *testing.T) {
			ukToken := func() string {
				p := payloads.VerifiedValuePayload{
					VerificationKind: schemas.VerificationKind_Phone,
					Entity:           objName,
					Field:            "email",
					Value:            email,
				}
				token, err := tokens.IssueToken(time.Minute, &p)
				require.NoError(err)
				return token
			}()

			row := newObject(cfg, objName)
			row.PutInt32("int32", 1)
			row.PutString("email", ukToken)

			_, err := row.Build()
			require.ErrorIs(err, ErrInvalidVerificationKind)
		})

		t.Run("error if wrong verified entity in token", func(t *testing.T) {
			weToken := func() string {
				p := payloads.VerifiedValuePayload{
					VerificationKind: schemas.VerificationKind_EMail,
					Entity:           schemas.NewQName("test", "other"),
					Field:            "email",
					Value:            email,
				}
				token, err := tokens.IssueToken(time.Minute, &p)
				require.NoError(err)
				return token
			}()

			row := newObject(cfg, objName)
			row.PutInt32("int32", 1)
			row.PutString("email", weToken)

			_, err := row.Build()
			require.ErrorIs(err, ErrInvalidName)
		})

		t.Run("error if wrong verified field in token", func(t *testing.T) {
			wfToken := func() string {
				p := payloads.VerifiedValuePayload{
					VerificationKind: schemas.VerificationKind_EMail,
					Entity:           objName,
					Field:            "otherField",
					Value:            email,
				}
				token, err := tokens.IssueToken(time.Minute, &p)
				require.NoError(err)
				return token
			}()

			row := newObject(cfg, objName)
			row.PutInt32("int32", 1)
			row.PutString("email", wfToken)

			_, err := row.Build()
			require.ErrorIs(err, ErrInvalidName)
		})

		t.Run("error if wrong verified value type in token", func(t *testing.T) {
			wtToken := func() string {
				p := payloads.VerifiedValuePayload{
					VerificationKind: schemas.VerificationKind_EMail,
					Entity:           objName,
					Field:            "email",
					Value:            3.141592653589793238,
				}
				token, err := tokens.IssueToken(time.Minute, &p)
				require.NoError(err)
				return token
			}()

			row := newObject(cfg, objName)
			row.PutInt32("int32", 1)
			row.PutString("email", wtToken)

			_, err := row.Build()
			require.ErrorIs(err, ErrWrongFieldType)
		})

	})
}

func Test_ValidateErrors(t *testing.T) {
	require := require.New(t)
	test := test()

	provider := Provide(test.AppConfigs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvder())

	app, err := provider.AppStructs(test.appName)
	require.NoError(err)

	t.Run("ECode_EmptySchemaName", func(t *testing.T) {
		bld := app.Events().GetSyncRawEventBuilder(
			istructs.SyncRawEventBuilderParams{
				GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
					HandlingPartition: test.partition,
					PLogOffset:        test.plogOfs,
					Workspace:         test.workspace,
					WLogOffset:        test.wlogOfs,
					QName:             schemas.NullQName,
					RegisteredAt:      test.registeredTime,
				},
				Device:   test.device,
				SyncedAt: test.syncTime,
			})
		_, buildErr := bld.BuildRawEvent()
		require.ErrorIs(buildErr, ErrNameMissed)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(buildErr, &validateErr)
		require.Equal(ECode_EmptySchemaName, validateErr.Code())
	})

	t.Run("ECode_InvalidSchemaName", func(t *testing.T) {
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
		require.Equal(ECode_InvalidSchemaName, validateErr.Code())
	})

	t.Run("ECode_InvalidSchemaKind", func(t *testing.T) {
		var app istructs.IAppStructs

		cDocName := schemas.NewQName("test", "CDoc")
		cmdCreateDoc := schemas.NewQName("test", "CreateDoc")

		t.Run("builds app", func(t *testing.T) {
			bld := schemas.NewSchemaCache()

			t.Run("must be ok to build schemas", func(t *testing.T) {
				CDocSchema := bld.Add(cDocName, schemas.SchemaKind_CDoc)
				CDocSchema.AddField("Int32", schemas.DataKind_int32, false)
			})

			cfgs := make(AppConfigsType, 1)
			cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, bld)
			cfg.Resources.Add(NewCommandFunction(cmdCreateDoc, cDocName, schemas.NullQName, schemas.NullQName, NullCommandExec))

			storage, err := simpleStorageProvder().AppStorage(istructs.AppQName_test1_app1)
			require.NoError(err)
			err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
			require.NoError(err)

			provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvder())

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
		require.ErrorIs(buildErr, ErrWrongSchema)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(buildErr, &validateErr)
		require.Equal(ECode_InvalidSchemaKind, validateErr.Code())
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
		cmd.PutRecordID(schemas.SystemField_ID, 100500100500)
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
		newRec.PutRecordID(schemas.SystemField_ID, 1)
		r := newTestCDoc(1)
		_ = cud.Update(r)

		_, buildErr := bld.BuildRawEvent()
		require.ErrorIs(buildErr, ErrRecordIDUniqueViolation)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(buildErr, &validateErr)
		require.Equal(ECode_InvalidRecordID, validateErr.Code())
	})

	t.Run("ECode_InvalidRefRecordID", func(t *testing.T) {
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
		r := newTestCDoc(7)
		_ = cud.Update(r)

		_, buildErr := bld.BuildRawEvent()
		require.ErrorIs(buildErr, ErrorRecordIDNotFound)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(buildErr, &validateErr)
		require.Equal(ECode_InvalidRefRecordID, validateErr.Code())
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
		cmd.PutRecordID(schemas.SystemField_ID, 1)
		cmd.PutString(test.buyerIdent, test.buyerValue)
		bsk := cmd.ElementBuilder(test.basketIdent)
		bsk.PutRecordID(schemas.SystemField_ID, 2)
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
		cmd.PutRecordID(schemas.SystemField_ID, 1)
		cmd.PutString(test.buyerIdent, test.buyerValue)
		bsk := cmd.ElementBuilder(test.basketIdent)
		bsk.PutRecordID(schemas.SystemField_ID, 2)
		good := bsk.ElementBuilder(test.goodIdent)
		good.PutString(schemas.SystemField_Container, test.basketIdent) // error here

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
		cmd.PutRecordID(schemas.SystemField_ID, 1)
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
		cmd.PutRecordID(schemas.SystemField_ID, 1)
		cmd.PutString(test.buyerIdent, test.buyerValue)

		bsk := cmd.ElementBuilder(test.basketIdent)
		bsk.PutRecordID(schemas.SystemField_ID, 2)

		_ = cmd.ElementBuilder(test.basketIdent)

		_, buildErr := bld.BuildRawEvent()
		require.Error(buildErr)
		validateErr := validateErrorf(0, "")
		require.ErrorAs(buildErr, &validateErr)
		require.Equal(ECode_InvalidOccursMax, validateErr.Code())
	})
}
