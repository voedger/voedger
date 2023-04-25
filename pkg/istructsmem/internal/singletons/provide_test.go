/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package singletons

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
	"github.com/voedger/voedger/pkg/schemas"
)

func Test_BasicUsage(t *testing.T) {
	sp := istorageimpl.Provide(istorage.ProvideMem())
	storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

	versions := vers.NewVersions()
	if err := versions.Prepare(storage); err != nil {
		panic(err)
	}

	testName := schemas.NewQName("test", "schema")
	bld := schemas.NewSchemaCache()
	bld.Add(testName, schemas.SchemaKind_CDoc).SetSingleton()
	schemas, err := bld.Build()
	if err != nil {
		panic(err)
	}

	stones := NewSingletons()
	if err := stones.Prepare(storage, versions, schemas); err != nil {
		panic(err)
	}

	require := require.New(t)
	t.Run("basic Singletons methods", func(t *testing.T) {
		id, err := stones.GetID(testName)
		require.NoError(err)
		require.NotEqual(istructs.NullRecordID, id)

		n, err := stones.GetQName(id)
		require.NoError(err)
		require.Equal(testName, n)

		t.Run("must be able to load early stored names", func(t *testing.T) {
			otherVersions := vers.NewVersions()
			if err := otherVersions.Prepare(storage); err != nil {
				panic(err)
			}

			stones1 := NewSingletons()
			if err := stones1.Prepare(storage, versions, nil); err != nil {
				panic(err)
			}

			id1, err := stones.GetID(testName)
			require.NoError(err)
			require.Equal(id, id1)

			n1, err := stones.GetQName(id)
			require.NoError(err)
			require.Equal(testName, n1)
		})
	})
}

func test_SchemasSingletons(t *testing.T, stons *Singletons, cache schemas.SchemaCache) {
	require := require.New(t)
	cache.Schemas(
		func(s schemas.Schema) {
			if s.Singleton() {
				id, err := stons.GetID(s.QName())
				require.NoError(err)
				require.NotEqual(istructs.NullRecordID, id)
				name, err := stons.GetQName(id)
				require.NoError(err)
				require.Equal(s.QName(), name)
			}
		})
}

func Test_SingletonsGetID(t *testing.T) {

	require := require.New(t)
	cDocName := schemas.NewQName("test", "SignletonCDoc")

	stons := NewSingletons()

	t.Run("must be ok to construct Singletons", func(t *testing.T) {
		storage, versions, schemas := func() (istorage.IAppStorage, *vers.Versions, schemas.SchemaCache) {
			storage, err := istorageimpl.Provide(istorage.ProvideMem()).AppStorage(istructs.AppQName_test1_app1)
			require.NoError(err)

			versions := vers.NewVersions()
			err = versions.Prepare(storage)
			require.NoError(err)

			bld := schemas.NewSchemaCache()
			schema := bld.Add(cDocName, schemas.SchemaKind_CDoc)
			schema.AddField("f1", schemas.DataKind_QName, true)
			schema.SetSingleton()
			schemas, err := bld.Build()
			require.NoError(err)

			return storage, versions, schemas
		}()

		err := stons.Prepare(storage, versions, schemas)
		require.NoError(err)

		test_SchemasSingletons(t, stons, schemas)
	})

	testID := func(id istructs.RecordID, known bool, qname schemas.QName) {
		t.Run(fmt.Sprintf("test Singletons.GetQName(%v)", id), func(t *testing.T) {
			qName, err := stons.GetQName(id)
			if known {
				require.NoError(err)
				require.Equal(qname, qName)
			} else {
				require.ErrorIs(err, ErrIDNotFound)
				require.Equal(qName, schemas.NullQName)
			}
		})
	}

	testQName := func(qname schemas.QName, known bool) {
		t.Run(fmt.Sprintf("test Stones.GetID(%v)", qname), func(t *testing.T) {
			var id istructs.RecordID
			var err error

			id, err = stons.GetID(qname)
			if known {
				require.NoError(err)
				require.NotNil(id)

				testID(id, true, qname)
			} else {
				require.ErrorIs(err, ErrNameNotFound)
			}
		})
	}

	t.Run("check NullQName", func(t *testing.T) {
		testQName(schemas.NullQName, false)
	})

	t.Run("check known QName", func(t *testing.T) {
		testQName(cDocName, true)
	})

	t.Run("check unknown QName", func(t *testing.T) {
		testQName(schemas.NewQName("unknown", "CDoc"), false)
	})

	t.Run("check unknown id", func(t *testing.T) {
		testID(istructs.MaxSingletonID-1, false, schemas.NullQName)
	})
}

func Test_Singletons_Errors(t *testing.T) {

	require := require.New(t)
	cDocName := schemas.NewQName("test", "SignletonCDoc")
	testError := fmt.Errorf("test error")

	t.Run("must error if unknown version of Singletons system view", func(t *testing.T) {
		storage, err := istorageimpl.Provide(istorage.ProvideMem()).AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)

		versions := vers.NewVersions()
		err = versions.Prepare(storage)
		require.NoError(err)

		err = versions.PutVersion(vers.SysSingletonsVersion, 0xFF)
		require.NoError(err)

		stone := NewSingletons()
		err = stone.Prepare(storage, versions, nil)
		require.ErrorIs(err, vers.ErrorInvalidVersion)
	})

	t.Run("must error if unable store version of Singletons system  view", func(t *testing.T) {
		storage := teststore.NewTestStorage()
		storage.SchedulePutError(testError, utils.ToBytes(consts.SysView_Versions), utils.ToBytes(vers.SysSingletonsVersion))

		versions := vers.NewVersions()
		err := versions.Prepare(storage)
		require.NoError(err)

		bld := schemas.NewSchemaCache()
		schema := bld.Add(cDocName, schemas.SchemaKind_CDoc)
		schema.AddField("f1", schemas.DataKind_QName, true)
		schema.SetSingleton()
		schemas, err := bld.Build()
		require.NoError(err)

		stone := NewSingletons()
		err = stone.Prepare(storage, versions, schemas)

		require.ErrorIs(err, testError)
	})

	t.Run("must error if maximum singletons is exceeded by CDocs", func(t *testing.T) {
		storage := teststore.NewTestStorage()

		versions := vers.NewVersions()
		err := versions.Prepare(storage)
		require.NoError(err)

		bld := schemas.NewSchemaCache()
		for id := istructs.FirstSingletonID; id <= istructs.MaxSingletonID; id++ {
			bld.Add(schemas.NewQName("test", fmt.Sprintf("CDoc_%v", id)), schemas.SchemaKind_CDoc).SetSingleton()
		}
		schemas, err := bld.Build()
		require.NoError(err)

		stons := NewSingletons()
		err = stons.Prepare(storage, versions, schemas)

		require.ErrorIs(err, ErrSingletonIDsExceeds)
	})

	t.Run("must error if store ID for some singledoc to storage is failed", func(t *testing.T) {
		schemaName := schemas.NewQName("test", "ErrorSchema")

		storage := teststore.NewTestStorage()
		storage.SchedulePutError(testError, utils.ToBytes(consts.SysView_SingletonIDs, lastestVersion), []byte(schemaName.String()))

		versions := vers.NewVersions()
		err := versions.Prepare(storage)
		require.NoError(err)

		bld := schemas.NewSchemaCache()
		bld.Add(schemaName, schemas.SchemaKind_CDoc).SetSingleton()
		schemas, err := bld.Build()
		require.NoError(err)

		stons := NewSingletons()
		err = stons.Prepare(storage, versions, schemas)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if retrieve ID for some singledoc from storage is failed", func(t *testing.T) {
		schemaName := schemas.NewQName("test", "ErrorSchema")

		storage := teststore.NewTestStorage()

		versions := vers.NewVersions()
		err := versions.Prepare(storage)
		require.NoError(err)

		bld := schemas.NewSchemaCache()
		bld.Add(schemaName, schemas.SchemaKind_CDoc).SetSingleton()
		schemas, err := bld.Build()
		require.NoError(err)

		stons := NewSingletons()
		err = stons.Prepare(storage, versions, schemas)
		require.NoError(err)

		storage.ScheduleGetError(testError, nil, []byte(schemaName.String()))
		stons1 := NewSingletons()
		err = stons1.Prepare(storage, versions, schemas)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if some some CDoc singleton QName from storage is not well formed", func(t *testing.T) {
		storage := teststore.NewTestStorage()

		versions := vers.NewVersions()
		err := versions.Prepare(storage)
		require.NoError(err)
		versions.PutVersion(vers.SysSingletonsVersion, lastestVersion)

		t.Run("crack storage by put invalid QName string into Singletons system view", func(t *testing.T) {
			err = storage.Put(
				utils.ToBytes(consts.SysView_SingletonIDs, lastestVersion),
				[]byte("error.CDoc.be-e-e"),
				utils.ToBytes(istructs.MaxSingletonID),
			)
			require.NoError(err)
		})

		stons := NewSingletons()
		err = stons.Prepare(storage, versions, nil)

		require.ErrorIs(err, schemas.ErrInvalidQNameStringRepresentation)
	})
}
