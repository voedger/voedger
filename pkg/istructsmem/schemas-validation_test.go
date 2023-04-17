/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func TestSchemaValidation(t *testing.T) {
	require := require.New(t)

	testAppCfg := func() *AppConfigType {
		cfgs := make(AppConfigsType, 1)
		return cfgs.AddConfig(istructs.AppQName_test1_app1)
	}

	t.Run("error if empty schema name", func(t *testing.T) {
		cfg := testAppCfg()
		_ = cfg.Schemas.Add(istructs.NullQName, istructs.SchemaKind_null)

		err := cfg.prepare(nil, nil)
		require.ErrorIs(err, ErrNameMissed)
	})

	t.Run("error if invalid schema name", func(t *testing.T) {
		cfg := testAppCfg()
		_ = cfg.Schemas.Add(istructs.NewQName("test", "ups-1"), istructs.SchemaKind_null)

		err := cfg.prepare(nil, nil)
		require.ErrorIs(err, ErrInvalidName)
	})

	t.Run("panic if schema name violated", func(t *testing.T) {
		cfg := testAppCfg()
		cDocName := istructs.NewQName("test", "CDoc")
		_ = cfg.Schemas.Add(cDocName, istructs.SchemaKind_CDoc)

		require.Panics(func() {
			_ = cfg.Schemas.Add(cDocName, istructs.SchemaKind_null)
		})
	})

	t.Run("error if empty schema field name", func(t *testing.T) {
		cfg := testAppCfg()
		cDocName := istructs.NewQName("test", "CDoc")
		schema := cfg.Schemas.Add(cDocName, istructs.SchemaKind_CDoc)
		schema.AddField("", istructs.DataKind_null, false)

		err := cfg.prepare(nil, nil)
		require.ErrorIs(err, ErrNameMissed)
	})

	t.Run("error if schema has invalid field name", func(t *testing.T) {
		cfg := testAppCfg()
		cDocName := istructs.NewQName("test", "CDoc")
		schema := cfg.Schemas.Add(cDocName, istructs.SchemaKind_CDoc)
		schema.AddField("i.i", istructs.DataKind_null, false)

		err := cfg.prepare(nil, nil)
		require.ErrorIs(err, ErrInvalidName)
	})

	t.Run("error if verified field has no verification kind", func(t *testing.T) {
		cfg := testAppCfg()
		cDocName := istructs.NewQName("test", "CDoc")
		schema := cfg.Schemas.Add(cDocName, istructs.SchemaKind_CDoc)
		schema.AddVerifiedField("oops", istructs.DataKind_int32, true)

		err := cfg.prepare(nil, nil)
		require.ErrorIs(err, ErrVerificationKindMissed)
	})

	t.Run("error if deprecated containers", func(t *testing.T) {
		cfg := testAppCfg()
		schema := cfg.Schemas.Add(istructs.NewQName("test", "ViewValue"), istructs.SchemaKind_ViewRecord_Value)
		schema.AddContainer("rec", istructs.NewQName("test", "ORec"), 0, 1)

		err := cfg.prepare(nil, nil)
		require.ErrorIs(err, ErrContainersUnavailable)
	})

	t.Run("error if unnamed container", func(t *testing.T) {
		cfg := testAppCfg()
		schema := cfg.Schemas.Add(istructs.NewQName("test", "CDoc"), istructs.SchemaKind_CDoc)
		schema.AddContainer("", istructs.NullQName, 0, 1)
		err := cfg.prepare(nil, nil)
		require.ErrorIs(err, ErrNameMissed)
	})

	t.Run("error if invalid container name", func(t *testing.T) {
		cfg := testAppCfg()
		schema := cfg.Schemas.Add(istructs.NewQName("test", "CDoc"), istructs.SchemaKind_CDoc)
		schema.AddContainer("Йohn", istructs.NullQName, 0, 1)
		err := cfg.prepare(nil, nil)
		require.ErrorIs(err, ErrInvalidName)
	})

	t.Run("error if container unnamed type", func(t *testing.T) {
		cfg := testAppCfg()
		schema := cfg.Schemas.Add(istructs.NewQName("test", "CDoc"), istructs.SchemaKind_CDoc)
		schema.AddContainer("rec", istructs.NullQName, 0, 1)
		err := cfg.prepare(nil, nil)
		require.ErrorIs(err, ErrNameMissed)
	})

	t.Run("error if container unknown type", func(t *testing.T) {
		cfg := testAppCfg()
		schema := cfg.Schemas.Add(istructs.NewQName("test", "CDoc"), istructs.SchemaKind_CDoc)
		schema.AddContainer("rec", istructs.NewQName("test", "CRec"), 0, 1)
		err := cfg.prepare(nil, nil)
		require.ErrorIs(err, ErrNameNotFound)
	})

	t.Run("error if container absurd occurs", func(t *testing.T) {
		cfg := testAppCfg()
		schema := cfg.Schemas.Add(istructs.NewQName("test", "CDoc"), istructs.SchemaKind_CDoc)
		schema.AddContainer("rec", istructs.NewQName("test", "CRec"), 0, 0)
		err := cfg.prepare(nil, nil)
		require.ErrorIs(err, ErrMaxOccursMissed)
	})

	t.Run("error if container absurd occurs", func(t *testing.T) {
		cfg := testAppCfg()
		schema := cfg.Schemas.Add(istructs.NewQName("test", "CDoc"), istructs.SchemaKind_CDoc)
		schema.AddContainer("rec", istructs.NewQName("test", "CRec"), 2, 1)
		err := cfg.prepare(nil, nil)
		require.ErrorIs(err, ErrMaxOccursLessMinOccurs)
	})

	t.Run("error if unavailable container schema kind", func(t *testing.T) {
		cfg := testAppCfg()
		schema := cfg.Schemas.Add(istructs.NewQName("test", "CDoc"), istructs.SchemaKind_CDoc)
		schema.AddContainer("rec", istructs.NewQName("test", "ORec"), 0, 1)
		_ = cfg.Schemas.Add(istructs.NewQName("test", "ORec"), istructs.SchemaKind_ORecord)
		err := cfg.prepare(nil, nil)
		require.ErrorIs(err, ErrWrongSchemaStruct)
	})

	t.Run("check direct recursive schema inclusion is ok", func(t *testing.T) {
		cfg := testAppCfg()
		recName := istructs.NewQName("test", "CRec")

		docSchema := cfg.Schemas.Add(istructs.NewQName("test", "CDoc"), istructs.SchemaKind_CDoc)
		docSchema.AddContainer("rec", recName, 0, istructs.ContainerOccurs_Unbounded)

		recSchema := cfg.Schemas.Add(recName, istructs.SchemaKind_CRecord)
		recSchema.AddContainer("rec", recName, 0, istructs.ContainerOccurs_Unbounded)

		storage, err := simpleStorageProvder().AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)
		err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
		require.NoError(err)
	})

	t.Run("check indirect recursive schema inclusion is ok", func(t *testing.T) {
		cfg := testAppCfg()

		recName := istructs.NewQName("test", "CRec")
		subRecName := istructs.NewQName("test", "CSubRec")

		docSchema := cfg.Schemas.Add(istructs.NewQName("test", "CDoc"), istructs.SchemaKind_CDoc)
		docSchema.AddContainer("rec", recName, 0, istructs.ContainerOccurs_Unbounded)

		recSchema := cfg.Schemas.Add(recName, istructs.SchemaKind_CRecord)
		recSchema.AddContainer("subRec", subRecName, 0, istructs.ContainerOccurs_Unbounded)

		subRecSchema := cfg.Schemas.Add(subRecName, istructs.SchemaKind_CRecord)
		subRecSchema.AddContainer("rec", recName, 0, istructs.ContainerOccurs_Unbounded)

		storage, err := simpleStorageProvder().AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)
		err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
		require.NoError(err)
	})
}

func TestSchemaValidation_ViewRecords(t *testing.T) {
	require := require.New(t)

	testAppCfg := func() *AppConfigType {
		cfgs := make(AppConfigsType, 1)
		return cfgs.AddConfig(istructs.AppQName_test1_app1)
	}

	t.Run("New-style view schema creation test", func(t *testing.T) {
		cfg := testAppCfg()

		viewName := istructs.NewQName("test", "viewDrinks")

		viewSchema := cfg.Schemas.AddView(viewName)
		require.NotNil(viewSchema)
		require.Equal(viewSchema.Name(), viewName)

		require.NotNil(viewSchema.Schema())
		require.Equal(viewSchema.Schema().Kind(), istructs.SchemaKind_ViewRecord)

		require.NotNil(viewSchema.PartKeySchema())
		require.Equal(viewSchema.PartKeySchema().Kind(), istructs.SchemaKind_ViewRecord_PartitionKey)

		require.NotNil(viewSchema.ClustColsSchema())
		require.Equal(viewSchema.ClustColsSchema().Kind(), istructs.SchemaKind_ViewRecord_ClusteringColumns)

		require.NotNil(viewSchema.ValueSchema())
		require.Equal(viewSchema.ValueSchema().Kind(), istructs.SchemaKind_ViewRecord_Value)

		err := cfg.prepare(nil, nil)
		require.ErrorIs(err, coreutils.ErrFieldsMissed)
	})

	t.Run("New-style view schema chain-style creation test (see #!17865)", func(t *testing.T) {
		cfg := testAppCfg()

		viewName := istructs.NewQName("test", "viewDrinks")

		cfg.Schemas.AddView(viewName).
			AddPartField("Field1", istructs.DataKind_int32).
			AddClustColumn("Field2", istructs.DataKind_int32).
			AddValueField("Field3", istructs.DataKind_int32, false)

		storage, err := simpleStorageProvder().AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)
		err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
		require.NoError(err)
	})

	t.Run("error if empty view schema", func(t *testing.T) {
		cfg := testAppCfg()

		viewSchema := cfg.Schemas.Add(istructs.NewQName("test", "viewDrinks"), istructs.SchemaKind_ViewRecord)
		err := viewSchema.Validate(true)
		require.ErrorIs(err, ErrWrongSchemaStruct)
	})

	t.Run("error if missed some container in view schema", func(t *testing.T) {
		cfg := testAppCfg()
		viewSchema := cfg.Schemas.Add(istructs.NewQName("test", "viewDrinks"), istructs.SchemaKind_ViewRecord)

		viewSchema.AddContainer("badKeyName", istructs.NewQName("test", "viewDrinksKey"), 1, 1)
		viewSchema.AddContainer(istructs.SystemContainer_ViewClusteringCols, istructs.NewQName("test", "viewDrinksSort"), 1, 1)
		viewSchema.AddContainer(istructs.SystemContainer_ViewValue, istructs.NewQName("test", "viewDrinksValue"), 1, 1)

		err := viewSchema.Validate(true)
		require.ErrorIs(err, ErrWrongSchemaStruct)
	})

	t.Run("error if some container in view schema has wrong occurs", func(t *testing.T) {
		cfg := testAppCfg()
		viewSchema := cfg.Schemas.Add(istructs.NewQName("test", "viewDrinks"), istructs.SchemaKind_ViewRecord)

		viewSchema.AddContainer(istructs.SystemContainer_ViewPartitionKey, istructs.NewQName("test", "viewDrinksKey"), 0, 1)
		viewSchema.AddContainer(istructs.SystemContainer_ViewClusteringCols, istructs.NewQName("test", "viewDrinksSort"), 1, 1)
		viewSchema.AddContainer(istructs.SystemContainer_ViewValue, istructs.NewQName("test", "viewDrinksValue"), 1, 1)

		err := viewSchema.Validate(true)
		require.ErrorIs(err, ErrWrongSchemaStruct)

		viewSchema = cfg.Schemas.Add(istructs.NewQName("test", "viewDrinks1"), istructs.SchemaKind_ViewRecord)

		viewSchema.AddContainer(istructs.SystemContainer_ViewPartitionKey, istructs.NewQName("test", "viewDrinksKey"), 1, 2)
		viewSchema.AddContainer(istructs.SystemContainer_ViewClusteringCols, istructs.NewQName("test", "viewDrinksSort"), 1, 1)
		viewSchema.AddContainer(istructs.SystemContainer_ViewValue, istructs.NewQName("test", "viewDrinksValue"), 1, 1)

		err = viewSchema.Validate(true)
		require.ErrorIs(err, ErrWrongSchemaStruct)
	})

	t.Run("error if some container in view schema has unknown schema", func(t *testing.T) {
		cfg := testAppCfg()
		viewSchema := cfg.Schemas.Add(istructs.NewQName("test", "viewDrinks"), istructs.SchemaKind_ViewRecord)

		viewSchema.AddContainer(istructs.SystemContainer_ViewPartitionKey, istructs.NewQName("test", "viewDrinksKey"), 1, 1)
		viewSchema.AddContainer(istructs.SystemContainer_ViewClusteringCols, istructs.NewQName("test", "viewDrinksSort"), 1, 1)
		viewSchema.AddContainer(istructs.SystemContainer_ViewValue, istructs.NewQName("test", "viewDrinksValue"), 1, 1)

		_ = cfg.Schemas.Add(istructs.NewQName("test", "viewDrinksKey"), istructs.SchemaKind_ViewRecord_PartitionKey)

		err := viewSchema.Validate(true)
		require.ErrorIs(err, ErrNameNotFound)
	})

	t.Run("error if some container in view schema has wrong schema kind", func(t *testing.T) {
		cfg := testAppCfg()
		viewSchema := cfg.Schemas.Add(istructs.NewQName("test", "viewDrinks"), istructs.SchemaKind_ViewRecord)

		viewSchema.AddContainer(istructs.SystemContainer_ViewPartitionKey, istructs.NewQName("test", "viewDrinksKey"), 1, 1)
		viewSchema.AddContainer(istructs.SystemContainer_ViewClusteringCols, istructs.NewQName("test", "viewDrinksSort"), 1, 1)
		viewSchema.AddContainer(istructs.SystemContainer_ViewValue, istructs.NewQName("test", "viewDrinksValue"), 1, 1)

		_ = cfg.Schemas.Add(istructs.NewQName("test", "viewDrinksKey"), istructs.SchemaKind_ViewRecord_PartitionKey)
		_ = cfg.Schemas.Add(istructs.NewQName("test", "viewDrinksSort"), istructs.SchemaKind_ViewRecord_ClusteringColumns)
		_ = cfg.Schemas.Add(istructs.NewQName("test", "viewDrinksValue"), istructs.SchemaKind_CRecord)

		err := viewSchema.Validate(true)
		require.ErrorIs(err, ErrUnexpectedShemaKind)
	})

	t.Run("error if view partition key has no fields", func(t *testing.T) {
		cfg := testAppCfg()
		viewSchema := cfg.Schemas.AddView(istructs.NewQName("test", "viewDrinks"))

		err := viewSchema.Validate()
		require.ErrorIs(err, coreutils.ErrFieldsMissed)
	})

	t.Run("error if view clustering columns has no fields", func(t *testing.T) {
		cfg := testAppCfg()
		viewSchema := cfg.Schemas.AddView(istructs.NewQName("test", "viewDrinks"))
		viewSchema.AddPartField("fld1", istructs.DataKind_int64)

		err := viewSchema.Validate()
		require.ErrorIs(err, coreutils.ErrFieldsMissed)
	})

	t.Run("error if view partition key has variable-length fields", func(t *testing.T) {
		cfg := testAppCfg()
		viewSchema := cfg.Schemas.AddView(istructs.NewQName("test", "viewDrinks"))
		viewSchema.AddPartField("fld1", istructs.DataKind_string)
		viewSchema.AddClustColumn("fld2", istructs.DataKind_string)

		err := viewSchema.Validate()
		require.ErrorIs(err, coreutils.ErrFieldTypeMismatch)
	})

	t.Run("error if view clustering columns has not last variable-length field", func(t *testing.T) {
		cfg := testAppCfg()
		viewSchema := cfg.Schemas.AddView(istructs.NewQName("test", "viewDrinks"))
		viewSchema.AddPartField("fld1", istructs.DataKind_int64)
		viewSchema.AddClustColumn("fld2", istructs.DataKind_string)
		viewSchema.AddClustColumn("fld3", istructs.DataKind_string)

		err := viewSchema.Validate()
		require.ErrorIs(err, coreutils.ErrFieldTypeMismatch)
	})

	t.Run("Must have error if field unique violated (see #!17003)", func(t *testing.T) {
		cfg := testAppCfg()

		t.Run("test partKey/clustCols unique violated", func(t *testing.T) {
			viewSchema := cfg.Schemas.AddView(istructs.NewQName("test", "viewDrinks1"))
			viewSchema.AddPartField("Field1", istructs.DataKind_int32)
			viewSchema.AddClustColumn("Field1", istructs.DataKind_int32)
			viewSchema.AddValueField("Field3", istructs.DataKind_int32, false)

			err := viewSchema.Validate()
			require.ErrorIs(err, ErrNameUniqueViolation)
		})

		t.Run("test partKey/value unique violated", func(t *testing.T) {
			viewSchema := cfg.Schemas.AddView(istructs.NewQName("test", "viewDrinks2"))
			viewSchema.AddPartField("Field1", istructs.DataKind_int32)
			viewSchema.AddClustColumn("Field2", istructs.DataKind_int32)
			viewSchema.AddValueField("Field1", istructs.DataKind_int32, false)

			err := viewSchema.Validate()
			require.ErrorIs(err, ErrNameUniqueViolation)
		})

		t.Run("test clustCols/value unique violated", func(t *testing.T) {
			viewSchema := cfg.Schemas.AddView(istructs.NewQName("test", "viewDrinks3"))
			viewSchema.AddPartField("Field1", istructs.DataKind_int32)
			viewSchema.AddClustColumn("Field2", istructs.DataKind_int32)
			viewSchema.AddValueField("Field2", istructs.DataKind_int32, false)

			err := viewSchema.Validate()
			require.ErrorIs(err, ErrNameUniqueViolation)
		})

		t.Run("test ok if no violations", func(t *testing.T) {
			viewSchema := cfg.Schemas.AddView(istructs.NewQName("test", "viewDrinks"))
			viewSchema.AddPartField("Field1", istructs.DataKind_int32)
			viewSchema.AddClustColumn("Field2", istructs.DataKind_int32)
			viewSchema.AddValueField("Field3", istructs.DataKind_int32, false)

			err := viewSchema.Validate()
			require.NoError(err)

			t.Run("test common schema methods", func(t *testing.T) {
				schema := viewSchema.Schema()
				require.NotNil(schema)

				require.Equal(istructs.SchemaKind_ViewRecord, schema.Kind())
				require.Equal(istructs.NewQName("test", "viewDrinks"), schema.QName())

				valSchema := viewSchema.ValueSchema()
				require.NotNil(valSchema)

				fldCnt1 := 0
				valSchema.Fields(func(fieldName string, kind istructs.DataKindType) { fldCnt1++ })
				require.GreaterOrEqual(fldCnt1, 1)

				fldCnt2 := 0
				valSchema.ForEachField(func(istructs.IFieldDescr) { fldCnt2++ })
				require.Equal(fldCnt1, fldCnt2)

				cntCnt1 := 0
				schema.Containers(func(containerName string, schema istructs.QName) { cntCnt1++ })
				require.Equal(3, cntCnt1)

				cntCnt2 := 0
				schema.ForEachContainer(func(cont istructs.IContainerDescr) {
					switch cont.Name() {
					case istructs.SystemContainer_ViewValue:
						require.Equal(valSchema.QName(), cont.Schema())
						require.EqualValues(1, cont.MinOccurs())
						require.EqualValues(1, cont.MaxOccurs())
					}
					cntCnt2++
				})
				require.Equal(cntCnt1, cntCnt2)
			})
		})
	})
}

func TestSchemaValidation_Singletons(t *testing.T) {
	require := require.New(t)
	storage := newTestStorage()

	testAppCfg := func() *AppConfigType {
		cfgs := make(AppConfigsType, 1)
		return cfgs.AddConfig(istructs.AppQName_test1_app1)
	}

	t.Run("must be ok to assign CDOC singleton", func(t *testing.T) {
		cfg := testAppCfg()

		docName := istructs.NewQName("test", "cdoc")
		schema := cfg.Schemas.Add(docName, istructs.SchemaKind_CDoc)
		schema.SetSingleton()

		err := cfg.prepare(nil, storage)
		require.NoError(err)

		var id istructs.RecordID

		t.Run("must be valid id for known singleton", func(t *testing.T) {
			id, err = cfg.singletons.qNameToID(docName)
			require.NoError(err)
			require.GreaterOrEqual(id, istructs.RecordID(istructs.FirstSingletonID))
			require.LessOrEqual(id, istructs.RecordID(istructs.MaxSingletonID))
		})

		t.Run("must be error if unknown singleton", func(t *testing.T) {
			id1, err := cfg.singletons.qNameToID(istructs.NewQName("test", "cdoc1"))
			require.Error(err, ErrNameNotFound)
			require.Equal(id1, istructs.NullRecordID)
		})

		t.Run("must be ok to reassign singleton from exists storage", func(t *testing.T) {
			// cfg1 := appCfgWithStorage(cfg.storage)
			cfg1 := testAppCfg()

			schema := cfg1.Schemas.Add(docName, istructs.SchemaKind_CDoc)
			schema.SetSingleton()

			err := cfg1.prepare(nil, storage)
			require.NoError(err)

			t.Run("must be reused (equal) id from exists storage", func(t *testing.T) {
				id1, err := cfg1.singletons.qNameToID(docName)
				require.NoError(err)
				require.Equal(id, id1)
			})
		})
	})

	t.Run("must be error if assign singleton not for CDOC", func(t *testing.T) {
		cfg := testAppCfg()

		docName := istructs.NewQName("test", "odoc")
		schema := cfg.Schemas.Add(docName, istructs.SchemaKind_ODoc)
		schema.SetSingleton()

		err := cfg.prepare(nil, nil)
		require.ErrorIs(err, ErrWrongSchemaStruct)
	})

	t.Run("must be error if singleton CDOC count is exceeds", func(t *testing.T) {
		cfg := testAppCfg()

		for i := istructs.FirstSingletonID; i <= istructs.MaxSingletonID; i++ {
			docName := istructs.NewQName("test", fmt.Sprintf("cdoc_%d", i))
			schema := cfg.Schemas.Add(docName, istructs.SchemaKind_CDoc)
			schema.SetSingleton()
		}

		docName := istructs.NewQName("test", "cdoc_exceed")
		schema := cfg.Schemas.Add(docName, istructs.SchemaKind_CDoc)
		schema.SetSingleton()

		storage, err := simpleStorageProvder().AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)
		err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
		require.ErrorIs(err, ErrSingletonIDsExceeds)
	})
}

func TestSchemaValidation_Uniques(t *testing.T) {
	require := require.New(t)
	testAppCfg := func() *AppConfigType {
		cfgs := make(AppConfigsType, 1)
		return cfgs.AddConfig(istructs.AppQName_test1_app1)
	}

	qName := istructs.NewQName("my", "qname")
	storage, err := simpleStorageProvder().AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)

	t.Run("ok", func(t *testing.T) {
		cfg := testAppCfg()
		cfg.Schemas.Add(qName, istructs.SchemaKind_CDoc).
			AddField("name", istructs.DataKind_string, true).
			AddField("fld", istructs.DataKind_int32, true).
			AddField("str", istructs.DataKind_int64, true)
		cfg.Uniques.Add(qName, []string{"name", "fld"})
		cfg.Uniques.Add(qName, []string{"name", "str"})
		err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
		require.NoError(err)
	})

	t.Run("schema kind may not have uniques", func(t *testing.T) {
		cfg := testAppCfg()
		cfg.Schemas.Add(qName, istructs.SchemaKind_Element).
			AddField("name", istructs.DataKind_string, true)
		cfg.Uniques.Add(qName, []string{"name"})
		err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
		require.ErrorIs(err, ErrSchemaKindMayNotHaveUniques)
	})

	t.Run("unknown schema QName", func(t *testing.T) {
		cfg := testAppCfg()
		cfg.Uniques.Add(qName, []string{"name"})
		err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
		require.ErrorIs(err, ErrUnknownSchemaQName)
	})

	t.Run("empty set of key fields", func(t *testing.T) {
		cfg := testAppCfg()
		cfg.Schemas.Add(qName, istructs.SchemaKind_CDoc).
			AddField("name", istructs.DataKind_string, true)
		cfg.Uniques.Add(qName, nil)
		err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
		require.ErrorIs(err, ErrEmptySetOfKeyFields)
	})

	t.Run("key field is used more than once", func(t *testing.T) {
		cfg := testAppCfg()
		cfg.Schemas.Add(qName, istructs.SchemaKind_CDoc).
			AddField("name", istructs.DataKind_string, true)
		cfg.Uniques.Add(qName, []string{"name", "name"})
		err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
		require.ErrorIs(err, ErrKeyFieldIsUsedMoreThanOnce)
	})

	t.Run("unknown key field", func(t *testing.T) {
		cfg := testAppCfg()
		cfg.Schemas.Add(qName, istructs.SchemaKind_CDoc).
			AddField("name", istructs.DataKind_string, true)
		cfg.Uniques.Add(qName, []string{"unknown"})
		err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
		require.ErrorIs(err, ErrUnknownKeyField)
	})

	t.Run("uniques have same fields", func(t *testing.T) {
		cfg := testAppCfg()
		cfg.Schemas.Add(qName, istructs.SchemaKind_CDoc).
			AddField("name", istructs.DataKind_string, true).
			AddField("fld", istructs.DataKind_int32, true)
		cfg.Uniques.Add(qName, []string{"name", "fld"})
		cfg.Uniques.Add(qName, []string{"fld", "name"})
		err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
		require.ErrorIs(err, ErrUniquesHaveSameFields)
	})

	t.Run("key must have not more than one variable size field", func(t *testing.T) {
		cfg := testAppCfg()
		cfg.Schemas.Add(qName, istructs.SchemaKind_CDoc).
			AddField("string", istructs.DataKind_string, true).
			AddField("bytes", istructs.DataKind_bytes, true).
			AddField("int", istructs.DataKind_int32, true)
		cfg.Uniques.Add(qName, []string{"string", "int", "bytes"})
		err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
		require.ErrorIs(err, ErrKeyMustHaveNotMoreThanOneVarSizeField)
	})

	t.Run("key field must be required", func(t *testing.T) {
		cfg := testAppCfg()
		cfg.Schemas.Add(qName, istructs.SchemaKind_CDoc).
			AddField("int", istructs.DataKind_int32, false)
		cfg.Uniques.Add(qName, []string{"int"})
		err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
		require.ErrorIs(err, ErrKeyFieldMustBeRequired)
	})
}
