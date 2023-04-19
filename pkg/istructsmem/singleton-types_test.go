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
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
	"github.com/voedger/voedger/pkg/schemas"
)

func test_SchemasSingletons(t *testing.T, cfg *AppConfigType) {
	require := require.New(t)
	cfg.Schemas.EnumSchemas(
		func(s schemas.Schema) {
			if s.Singleton() {
				_, err := cfg.singletons.qNameToID(s.QName())
				require.NoError(err)
			}
		})
}

func Test_singletonsCacheType_qNamesToID(t *testing.T) {

	require := require.New(t)
	cDocName := istructs.NewQName("test", "SignletonCDoc")

	var cfg *AppConfigType

	t.Run("must be ok to construct app config", func(t *testing.T) {

		schemas := schemas.NewSchemaCache()
		t.Run("must be ok to build schemas", func(t *testing.T) {
			schema := schemas.Add(cDocName, istructs.SchemaKind_CDoc)
			schema.AddField("f1", istructs.DataKind_QName, true)
			schema.SetSingleton()
		})

		cfgs := make(AppConfigsType, 1)
		cfg = cfgs.AddConfig(istructs.AppQName_test1_app1, schemas)

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvder())
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.NoError(err)

		test_SchemasSingletons(t, cfg)
	})

	testID := func(id istructs.RecordID, known bool, qname istructs.QName) {
		t.Run(fmt.Sprintf("test idToQName(%v)", id), func(t *testing.T) {
			qName, err := cfg.singletons.idToQName(id)
			if known {
				require.NoError(err)
				require.Equal(qname, qName)
			} else {
				require.ErrorIs(err, ErrIDNotFound)
				require.Equal(qName, istructs.NullQName)
			}
		})
	}

	testQName := func(qname istructs.QName, known bool) {
		t.Run(fmt.Sprintf("test qNameToID(%v)", qname), func(t *testing.T) {
			var id istructs.RecordID
			var err error

			id, err = cfg.singletons.qNameToID(qname)
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
		testQName(istructs.NullQName, false)
	})

	t.Run("check known QName", func(t *testing.T) {
		testQName(cDocName, true)
	})

	t.Run("check unknown QName", func(t *testing.T) {
		testQName(istructs.NewQName("unknown", "CDoc"), false)
	})

	t.Run("check unknown id", func(t *testing.T) {
		testID(cfg.singletons.lastID+1, false, istructs.NullQName)
	})
}

func Test_singletonsCacheType_Errors(t *testing.T) {

	require := require.New(t)
	testError := fmt.Errorf("test error")
	storage, err := simpleStorageProvder().AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)

	t.Run("must error if unknown version of sys.Singletons view", func(t *testing.T) {

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, schemas.NewSchemaCache())

		cfg.versions.Prepare(storage)
		err := cfg.versions.PutVersion(vers.SysSingletonsVersion, 0xFF)
		require.NoError(err)

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvder())
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, ErrorInvalidVersion)

		// reset storage
		storage, err = simpleStorageProvder().AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)
	})

	t.Run("must error if unable store version of sys.Singletons view", func(t *testing.T) {

		storage := teststore.NewTestStorage()
		storage.SchedulePutError(testError, utils.ToBytes(consts.SysView_Versions), utils.ToBytes(vers.SysSingletonsVersion))
		storageProvider := teststore.NewTestStorageProvider(storage)

		schemas := schemas.NewSchemaCache()

		t.Run("must be ok to build schemas", func(t *testing.T) {
			schema := schemas.Add(istructs.NewQName("test", "CDoc"), istructs.SchemaKind_CDoc)
			schema.SetSingleton()
		})

		cfgs := make(AppConfigsType, 1)
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1, schemas)
		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if maximum singletons is exceeded by CDocs", func(t *testing.T) {

		schemas := schemas.NewSchemaCache()

		t.Run("must be ok to build schemas", func(t *testing.T) {
			for i := 0; i <= 0x200; i++ {
				schemas.Add(istructs.NewQName("test", fmt.Sprintf("CDoc%d", i)), istructs.SchemaKind_CDoc).SetSingleton()
			}
		})

		cfgs := make(AppConfigsType, 1)
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1, schemas)

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvder())
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, ErrSingletonIDsExceeds)
	})

	t.Run("must error if store ID for some singledoc to storage is failed", func(t *testing.T) {
		schemaName := istructs.NewQName("test", "ErrorSchema")

		storage := teststore.NewTestStorage()
		storage.SchedulePutError(testError, utils.ToBytes(consts.SysView_SingletonIDs, verSysSingletonsLastest), []byte(schemaName.String()))
		storageProvider := teststore.NewTestStorageProvider(storage)

		schemas := schemas.NewSchemaCache()

		t.Run("must be ok to build schemas", func(t *testing.T) {
			schemas.Add(schemaName, istructs.SchemaKind_CDoc).SetSingleton()
		})

		cfgs := make(AppConfigsType, 1)
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1, schemas)

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if retrieve ID for some singledoc from storage is failed", func(t *testing.T) {
		schemaName := istructs.NewQName("test", "ErrorSchema")

		storage := teststore.NewTestStorage()
		storage.ScheduleGetError(testError, nil, []byte(schemaName.String()))
		storage.SchedulePutError(testError, nil, []byte(schemaName.String()))
		storageProvider := teststore.NewTestStorageProvider(storage)

		schemas := schemas.NewSchemaCache()

		t.Run("must be ok to build schemas", func(t *testing.T) {
			schemas.Add(schemaName, istructs.SchemaKind_CDoc).SetSingleton()
		})

		cfgs := make(AppConfigsType, 1)
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1, schemas)

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if some some CDoc singleton QName from storage is not well formed", func(t *testing.T) {
		storage := teststore.NewTestStorage()
		storageProvider := teststore.NewTestStorageProvider(storage)

		t.Run("crack storage by put invalid QName string into sys.Singletons view", func(t *testing.T) {
			err := storage.Put(
				utils.ToBytes(consts.SysView_Versions),
				utils.ToBytes(vers.SysSingletonsVersion),
				utils.ToBytes(verSysSingletonsLastest),
			)
			require.NoError(err)

			err = storage.Put(
				utils.ToBytes(consts.SysView_SingletonIDs, verSysSingletonsLastest),
				[]byte("error.CDoc.be-e-e"),
				utils.ToBytes(uint64(istructs.MaxSingletonID)),
			)
			require.NoError(err)
		})

		cfgs := make(AppConfigsType, 1)
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1, schemas.NewSchemaCache())

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.Error(err)
		require.Contains(err.Error(), "invalid string representation")
	})
}

func Test_singletonsCacheType_ReuseStorage(t *testing.T) {

	require := require.New(t)

	testQNameA := istructs.NewQName("test", "CDocA")
	testQNameB := istructs.NewQName("test", "CDocB")
	testQNameC := istructs.NewQName("test", "CDocC")

	storage, err := simpleStorageProvder().AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)

	appCfg1 := func() *AppConfigType {
		schemas := schemas.NewSchemaCache()

		t.Run("must be ok to build schemas", func(t *testing.T) {
			schemas.Add(testQNameA, istructs.SchemaKind_CDoc).SetSingleton()
			schemas.Add(testQNameC, istructs.SchemaKind_CDoc).SetSingleton()
		})

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, schemas)

		err := cfg.prepare(nil, storage)
		require.NoError(err)
		return cfg
	}

	appCfg2 := func() *AppConfigType {
		schemas := schemas.NewSchemaCache()

		t.Run("must be ok to build schemas", func(t *testing.T) {
			schemas.Add(testQNameA, istructs.SchemaKind_CDoc).SetSingleton()
			schemas.Add(testQNameB, istructs.SchemaKind_CDoc).SetSingleton()
			schemas.Add(testQNameC, istructs.SchemaKind_CDoc).SetSingleton()
		})

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, schemas)
		err := cfg.prepare(nil, storage)
		require.NoError(err)
		return cfg
	}

	t.Run("must use equal singleton IDs if storage reused", func(t *testing.T) {
		cfg1 := appCfg1()
		idA1, err := cfg1.singletons.qNameToID(testQNameA)
		require.NoError(err)
		idC1, err := cfg1.singletons.qNameToID(testQNameC)
		require.NoError(err)
		test_SchemasSingletons(t, cfg1)

		_, err = cfg1.singletons.qNameToID(testQNameB)
		require.ErrorIs(err, ErrNameNotFound)

		cfg2 := appCfg2()
		idA2, err := cfg2.singletons.qNameToID(testQNameA)
		require.NoError(err)
		idC2, err := cfg2.singletons.qNameToID(testQNameC)
		require.NoError(err)
		test_SchemasSingletons(t, cfg2)

		_, err = cfg2.singletons.qNameToID(testQNameB)
		require.NoError(err)

		require.Equal(idA1, idA2, "sic!")
		require.Equal(idC1, idC2, "sic!")
	})
}
