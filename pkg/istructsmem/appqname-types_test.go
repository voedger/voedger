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
	"github.com/untillpro/voedger/pkg/istructsmem/internal/consts"
	"github.com/untillpro/voedger/pkg/istructsmem/internal/utils"
	"github.com/untillpro/voedger/pkg/istructsmem/internal/vers"
	"github.com/untillpro/voedger/pkg/schemas"
)

func Test_qNameCacheType_qNamesToID(t *testing.T) {
	test := test()

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
			test.AppCfg.Schemas.Schemas(func(q istructs.QName) {
				testQName(q, true)
			})
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
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, schemas.NewSchemaCache())

		cfg.versions.Prepare(storage)
		err := cfg.versions.PutVersion(vers.SysQNamesVersion, 0xFF)
		require.NoError(err)

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, ErrorInvalidVersion)
	})

	t.Run("must error if unable store version of sys.QName view", func(t *testing.T) {

		storage := newTestStorage()
		storage.shedulePutError(testError, utils.ToBytes(consts.SysView_Versions), utils.ToBytes(vers.SysQNamesVersion))
		storageProvider := newTestStorageProvider(storage)

		schemas := schemas.NewSchemaCache()
		_ = schemas.Add(istructs.NewQName("test", "object"), istructs.SchemaKind_Object)

		cfgs := make(AppConfigsType, 1)
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1, schemas)

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if maximum QNames is exceeded by Schemas", func(t *testing.T) {
		storage := newTestStorage()
		storageProvider := newTestStorageProvider(storage)

		schemas := schemas.NewSchemaCache()
		for i := 0; i <= 0xFFFF; i++ {
			_ = schemas.Add(istructs.NewQName("test", fmt.Sprintf("object%d", i)), istructs.SchemaKind_Object)
		}
		require.NoError(schemas.ValidateSchemas())

		cfgs := make(AppConfigsType, 1)
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1, schemas)

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, ErrQNameIDsExceeds)
	})

	t.Run("must error if maximum QNames is exceeded by Resources", func(t *testing.T) {
		storage := newTestStorage()
		storageProvider := newTestStorageProvider(storage)

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, schemas.NewSchemaCache())

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

		schemas := schemas.NewSchemaCache()
		_ = schemas.Add(schemaName, istructs.SchemaKind_Object)

		storage := newTestStorage()
		storageProvider := newTestStorageProvider(storage)
		storage.sheduleGetError(testError, nil, []byte(schemaName.String()))
		storage.shedulePutError(testError, nil, []byte(schemaName.String()))

		cfgs := make(AppConfigsType, 1)
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1, schemas)

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
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, schemas.NewSchemaCache())
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
				utils.ToBytes(consts.SysView_Versions),
				utils.ToBytes(vers.SysQNamesVersion),
				utils.ToBytes(uint16(verSysQNamesLastest)),
			)
			require.NoError(err)

			err = storage.Put(
				utils.ToBytes(consts.SysView_QNames, uint16(verSysQNamesLastest)),
				[]byte("error.QName.o-o-o"),
				utils.ToBytes(uint16(0xFFFE)),
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

func Test_qNameCache_ReuseStorage(t *testing.T) {

	require := require.New(t)

	testQNameA := istructs.NewQName("test", "objectA")
	testQNameB := istructs.NewQName("test", "objectB")
	testQNameC := istructs.NewQName("test", "objectC")

	storage := newTestStorage()

	appCfg1 := func() *AppConfigType {
		schemas := schemas.NewSchemaCache()
		_ = schemas.Add(testQNameA, istructs.SchemaKind_Object)
		_ = schemas.Add(testQNameC, istructs.SchemaKind_Object)

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, schemas)
		err := cfg.prepare(nil, storage)
		require.NoError(err)
		return cfg
	}

	appCfg2 := func() *AppConfigType {
		schemas := schemas.NewSchemaCache()
		_ = schemas.Add(testQNameA, istructs.SchemaKind_Object)
		_ = schemas.Add(testQNameB, istructs.SchemaKind_Object)
		_ = schemas.Add(testQNameC, istructs.SchemaKind_Object)

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, schemas)

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
