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
)

func testSchemasSingletons(t *testing.T, cfg *AppConfigType) {
	require := require.New(t)
	for _, schema := range cfg.Schemas.schemas {
		if schema.singleton.enabled {
			id, err := cfg.singletons.qNameToID(schema.name)
			require.NoError(err, err)
			require.Equal(id, schema.singleton.id)
		}
	}
}

func Test_singletonsCacheType_qNamesToID(t *testing.T) {

	require := require.New(t)
	cDocName := istructs.NewQName("test", "SignletonCDoc")

	_, cfg := func() (app istructs.IAppStructs, cfg *AppConfigType) {

		cfgs := make(AppConfigsType, 1)
		cfg = cfgs.AddConfig(istructs.AppQName_test1_app1)
		cfg.Schemas.Add(cDocName, istructs.SchemaKind_CDoc).SetSingleton()

		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvder())

		app, err := provider.AppStructs(istructs.AppQName_test1_app1)
		require.NoError(err, err)

		testSchemasSingletons(t, cfg)

		return app, cfg
	}()

	testID := func(id istructs.RecordID, known bool, qname istructs.QName) {
		t.Run(fmt.Sprintf("test idToQName(%v)", id), func(t *testing.T) {
			qName, err := cfg.singletons.idToQName(id)
			if known {
				require.NoError(err, err)
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
				require.NoError(err, err)
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
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)
		cfg.storage = storage

		err := cfg.versions.putVersion(verSysSingletons, 0xFF)
		require.NoError(err, err)

		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvder())

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, ErrorInvalidVersion)

		// reset storage
		storage, err = simpleStorageProvder().AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)
	})

	t.Run("must error if unable store version of sys.Singletons view", func(t *testing.T) {

		storage := newTestStorage()
		storage.shedulePutError(testError, toBytes(uint16(QNameIDSysVesions)), toBytes(uint16(verSysSingletons)))
		storageProvider := newTestStorageProvider(storage)

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)
		schema := cfg.Schemas.Add(istructs.NewQName("test", "CDoc"), istructs.SchemaKind_CDoc)
		schema.SetSingleton()
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if maximum singletons is exceeded by CDocs", func(t *testing.T) {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)

		for i := 0; i <= 0x200; i++ {
			schema := cfg.Schemas.Add(istructs.NewQName("test", fmt.Sprintf("CDoc%d", i)), istructs.SchemaKind_CDoc)
			schema.SetSingleton()
		}

		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvder())

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, ErrSingletonIDsExceeds)
	})

	t.Run("must error if store ID for some singledoc to storage is failed", func(t *testing.T) {
		schemaName := istructs.NewQName("test", "ErrorSchema")
		storage := newTestStorage()
		storage.shedulePutError(testError, toBytes(uint16(QNameIDSysSingletonIDs), uint16(verSysSingletonsLastest)), []byte(schemaName.String()))
		storageProvider := newTestStorageProvider(storage)

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)
		schema := cfg.Schemas.Add(schemaName, istructs.SchemaKind_CDoc)
		schema.SetSingleton()

		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if retrieve ID for some singledoc from storage is failed", func(t *testing.T) {
		schemaName := istructs.NewQName("test", "ErrorSchema")
		storage := newTestStorage()
		storage.sheduleGetError(testError, nil, []byte(schemaName.String()))
		storage.shedulePutError(testError, nil, []byte(schemaName.String()))
		storageProvider := newTestStorageProvider(storage)

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)
		schema := cfg.Schemas.Add(schemaName, istructs.SchemaKind_CDoc)
		schema.SetSingleton()

		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if some some CDoc singleton QName from storage is not well formed", func(t *testing.T) {
		storage := newTestStorage()
		storageProvider := newTestStorageProvider(storage)

		t.Run("crack storage by put invalid QName string into sys.Singletons view", func(t *testing.T) {
			err := storage.Put(
				toBytes(uint16(QNameIDSysVesions)),
				toBytes(uint16(verSysSingletons)),
				toBytes(uint16(verSysSingletonsLastest)),
			)
			require.NoError(err, err)

			err = storage.Put(
				toBytes(uint16(QNameIDSysSingletonIDs), uint16(verSysSingletonsLastest)),
				[]byte("error.CDoc.be-e-e"),
				toBytes(uint64(istructs.MaxSingletonID)),
			)
			require.NoError(err, err)
		})

		cfgs := make(AppConfigsType, 1)
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1)

		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)

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
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)
		cfg.Schemas.Add(testQNameA, istructs.SchemaKind_CDoc).SetSingleton()
		cfg.Schemas.Add(testQNameC, istructs.SchemaKind_CDoc).SetSingleton()

		err := cfg.prepare(nil, storage)
		require.NoError(err, err)
		return cfg
	}

	appCfg2 := func() *AppConfigType {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)
		cfg.Schemas.Add(testQNameA, istructs.SchemaKind_CDoc).SetSingleton()
		cfg.Schemas.Add(testQNameB, istructs.SchemaKind_CDoc).SetSingleton()
		cfg.Schemas.Add(testQNameC, istructs.SchemaKind_CDoc).SetSingleton()
		err := cfg.prepare(nil, storage)
		require.NoError(err, err)
		return cfg
	}

	t.Run("must use equal singleton IDs if storage reused", func(t *testing.T) {
		cfg1 := appCfg1()
		idA1, err := cfg1.singletons.qNameToID(testQNameA)
		require.NoError(err, err)
		idC1, err := cfg1.singletons.qNameToID(testQNameC)
		require.NoError(err, err)
		testSchemasSingletons(t, cfg1)

		_, err = cfg1.singletons.qNameToID(testQNameB)
		require.ErrorIs(err, ErrNameNotFound)

		cfg2 := appCfg2()
		idA2, err := cfg2.singletons.qNameToID(testQNameA)
		require.NoError(err, err)
		require.Equal(idA1, idA2)
		idC2, err := cfg2.singletons.qNameToID(testQNameC)
		require.NoError(err, err)
		require.Equal(idC1, idC2)
		testSchemasSingletons(t, cfg2)

		_, err = cfg2.singletons.qNameToID(testQNameB)
		require.NoError(err, err)
	})
}
