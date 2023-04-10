/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/voedger/pkg/iratesce"
	"github.com/untillpro/voedger/pkg/istructs"
)

func Test_qNameCacheType_qNamesToID(t *testing.T) {

	qNames := &test.AppCfg.qNames

	testID := func(id QNameID, known bool, qname istructs.QName) {
		t.Run(fmt.Sprintf("test idToName(%v)", id), func(t *testing.T) {
			require := require.New(t)

			n, err := qNames.idToQName(id)
			if known {
				require.NoError(err)
				require.Equal(qname, n)
			} else {
				require.ErrorIs(err, ErrIDNotFound)
			}
		})
	}

	testQName := func(qname istructs.QName, known bool) {
		t.Run(fmt.Sprintf("test QNameToID(%v)", qname), func(t *testing.T) {
			require := require.New(t)

			var id QNameID
			var err error

			id, err = qNames.qNameToID(qname)
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
		testQName(istructs.NullQName, true)
	})

	t.Run("check known QName", func(t *testing.T) {
		testQName(test.saleCmdName, true)
	})

	t.Run("check unknown QName", func(t *testing.T) {
		testQName(istructs.NewQName("unknown", "abc555def"), false)
	})

	t.Run("check unknown id", func(t *testing.T) {
		testID(qNames.lastID+1, false, istructs.NullQName)
	})

	t.Run("check access from multiple threads", func(t *testing.T) {
		wg := sync.WaitGroup{}

		testerGood := func() {
			for name := range test.AppCfg.Schemas.schemas {
				testQName(name, true)
			}
			wg.Done()
		}

		testerBad := func(num int) {
			for i := 0; i < 15; i++ {
				testQName(istructs.NewQName(fmt.Sprintf("test%d", num), fmt.Sprintf("test%d", i)), false)
			}
			wg.Done()
		}

		for i := 0; i < 15; i++ {
			wg.Add(1)
			if i%2 == 0 {
				go testerGood()
			} else {
				go testerBad(i)
			}
		}
		wg.Wait()
	})
}

func Test_qNameCache_Errors(t *testing.T) {

	require := require.New(t)
	testError := fmt.Errorf("test error")

	t.Run("must error if unknown version of sys.QName view", func(t *testing.T) {
		storage := newTestStorage()
		storageProvider := newTestStorageProvider(storage)

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)
		cfg.storage = storage

		err := cfg.versions.putVersion(verSysQNames, 0xFF)
		require.NoError(err)

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, ErrorInvalidVersion)
	})

	t.Run("must error if unable store version of sys.QName view", func(t *testing.T) {

		storage := newTestStorage()
		storage.shedulePutError(testError, toBytes(uint16(QNameIDSysVesions)), toBytes(uint16(verSysQNames)))
		storageProvider := newTestStorageProvider(storage)

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)
		_ = cfg.Schemas.Add(istructs.NewQName("test", "object"), istructs.SchemaKind_Object)
		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if maximum QNames is exceeded by Schemas", func(t *testing.T) {
		storage := newTestStorage()
		storageProvider := newTestStorageProvider(storage)

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)

		for i := 0; i <= 0xFFFF; i++ {
			_ = cfg.Schemas.Add(istructs.NewQName("test", fmt.Sprintf("object%d", i)), istructs.SchemaKind_Object)
		}

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, ErrQNameIDsExceeds)
	})

	t.Run("must error if maximum QNames is exceeded by Resources", func(t *testing.T) {
		storage := newTestStorage()
		storageProvider := newTestStorageProvider(storage)

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)

		for i := 0; i <= 0xFFFF; i++ {
			_ = cfg.Resources.Add(NewCommandFunction(
				istructs.NewQName("test", fmt.Sprintf("command%d", i)),
				istructs.NullQName, istructs.NullQName, istructs.NullQName, NullCommandExec))
		}

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, ErrQNameIDsExceeds)
	})

	t.Run("must error if retrieve ID for some schema from storage is failed", func(t *testing.T) {
		schemaName := istructs.NewQName("test", "ErrorSchema")
		storage := newTestStorage()
		storageProvider := newTestStorageProvider(storage)
		storage.sheduleGetError(testError, nil, []byte(schemaName.String()))
		storage.shedulePutError(testError, nil, []byte(schemaName.String()))

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)
		cfg.Schemas.Add(schemaName, istructs.SchemaKind_Object)

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if retrieve ID for some resource from storage is failed", func(t *testing.T) {
		resourceName := istructs.NewQName("test", "ErrorResource")
		storage := newTestStorage()
		storage.shedulePutError(testError, nil, []byte(resourceName.String()))
		storageProvider := newTestStorageProvider(storage)

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)
		cfg.Resources.Add(NewQueryFunction(resourceName, istructs.NullQName, istructs.NullQName, NullQueryExec))

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if some some QName from storage is not well formed", func(t *testing.T) {
		storage := newTestStorage()
		storageProvider := newTestStorageProvider(storage)

		t.Run("crack storage by put invalid QName string into sys.QNames view", func(t *testing.T) {
			err := storage.Put(
				toBytes(uint16(QNameIDSysVesions)),
				toBytes(uint16(verSysQNames)),
				toBytes(uint16(verSysQNamesLastest)),
			)
			require.NoError(err)

			err = storage.Put(
				toBytes(uint16(QNameIDSysQNames), uint16(verSysQNamesLastest)),
				[]byte("error.QName.o-o-o"),
				toBytes(uint16(0xFFFE)),
			)
			require.NoError(err)
		})

		cfgs := make(AppConfigsType, 1)
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1)

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.Error(err)
		require.Contains(err.Error(), "invalid string representation")
	})
}

func Test_qNameCache_ReuseStorage(t *testing.T) {

	require := require.New(t)

	testQNameA := istructs.NewQName("test", "objectA")
	testQNameB := istructs.NewQName("test", "objectB")
	testQNameC := istructs.NewQName("test", "objectC")

	storage := newTestStorage()

	appCfg1 := func() *AppConfigType {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)
		_ = cfg.Schemas.Add(testQNameA, istructs.SchemaKind_Object)
		_ = cfg.Schemas.Add(testQNameC, istructs.SchemaKind_Object)
		err := cfg.prepare(nil, storage)
		require.NoError(err)
		return cfg
	}

	appCfg2 := func() *AppConfigType {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)
		_ = cfg.Schemas.Add(testQNameA, istructs.SchemaKind_Object)
		_ = cfg.Schemas.Add(testQNameB, istructs.SchemaKind_Object)
		_ = cfg.Schemas.Add(testQNameC, istructs.SchemaKind_Object)
		err := cfg.prepare(nil, storage)
		require.NoError(err)
		return cfg
	}

	t.Run("must use equal QNameID if storage reused", func(t *testing.T) {
		cfg1 := appCfg1()
		idA1, err := cfg1.qNames.qNameToID(testQNameA)
		require.NoError(err)
		idC1, err := cfg1.qNames.qNameToID(testQNameC)
		require.NoError(err)

		_, err = cfg1.qNames.qNameToID(testQNameB)
		require.ErrorIs(err, ErrNameNotFound)

		cfg2 := appCfg2()
		idA2, err := cfg2.qNames.qNameToID(testQNameA)
		require.NoError(err)
		require.Equal(idA1, idA2)
		idC2, err := cfg2.qNames.qNameToID(testQNameC)
		require.NoError(err)
		require.Equal(idC1, idC2)

		_, err = cfg2.qNames.qNameToID(testQNameB)
		require.NoError(err)
	})
}
