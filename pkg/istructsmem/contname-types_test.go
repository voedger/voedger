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
	"github.com/voedger/voedger/pkg/iratesce"
	istorage "github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
)

func Test_containerNameCache_nameToID(t *testing.T) {

	cNames := &test.AppCfg.cNames

	testID := func(id containerNameIDType, known bool, name string) {
		t.Run(fmt.Sprintf("test idToName(%v)", id), func(t *testing.T) {
			require := require.New(t)

			n, err := cNames.idToName(id)
			if known {
				require.NoError(err)
				require.Equal(name, n)
			} else {
				require.ErrorIs(err, ErrIDNotFound)
			}
		})
	}

	testName := func(name string, known bool) {
		t.Run(fmt.Sprintf("test nameToID(%v)", name), func(t *testing.T) {
			require := require.New(t)

			var id containerNameIDType
			var err error

			id, err = cNames.nameToID(name)
			if known {
				require.NoError(err)
				require.NotNil(id)

				testID(id, true, name)
			} else {
				require.ErrorIs(err, ErrNameNotFound)
			}
		})
	}

	t.Run("check empty name", func(t *testing.T) {
		testName("", true)
	})

	t.Run("check known name", func(t *testing.T) {
		testName(test.basketIdent, true)
	})

	t.Run("check unknown name", func(t *testing.T) {
		testName("unknownContainer123", false)
	})

	t.Run("check unknown id", func(t *testing.T) {
		testID(cNames.lastID+1, false, "")
	})

	t.Run("check access from multiple threads", func(t *testing.T) {
		wg := sync.WaitGroup{}

		testerGood := func() {
			for _, schema := range test.AppCfg.Schemas.schemas {
				if schema.kind != istructs.SchemaKind_ViewRecord {
					for n := range schema.containers {
						testName(n, true)
					}
				}
			}
			wg.Done()
		}

		testerBad := func(num int) {
			for i := 0; i < 15; i++ {
				testName(fmt.Sprintf("test%dErrorName%d", num, i), false)
			}
			wg.Done()
		}

		for i := 0; i < 1; i++ {
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

func Test_containerNameCache_Errors(t *testing.T) {
	require := require.New(t)
	testError := fmt.Errorf("test error")
	storage := newTestStorage()
	storageProvider := newTestStorageProvider(storage)

	t.Run("must error if unknown version of sys.Container view", func(t *testing.T) {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)
		cfg.storage = storage

		err := cfg.versions.putVersion(verSysContainers, 0xFF)
		require.NoError(err)

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, ErrorInvalidVersion)

		// clear the storage
		storage = newTestStorage()
		storageProvider = newTestStorageProvider(storage)
	})

	t.Run("must error if unable store version of sys.Containers view", func(t *testing.T) {
		storage.shedulePutError(testError, toBytes(uint16(QNameIDSysVesions)), toBytes(uint16(verSysContainers)))
		defer storage.reset()

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)
		obj := cfg.Schemas.Add(istructs.NewQName("test", "object"), istructs.SchemaKind_Object)
		obj.AddContainer("containerName", istructs.NewQName("test", "element"), 0, 1)
		_ = cfg.Schemas.Add(istructs.NewQName("test", "element"), istructs.SchemaKind_Element)
		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if maximum container names is exceeded", func(t *testing.T) {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)

		schema := cfg.Schemas.Add(istructs.NewQName("test", "object"), istructs.SchemaKind_Object)
		for i := 0; i <= 0xFFFF; i++ {
			schema.AddContainer(fmt.Sprintf("cont%d", i), istructs.NewQName("test", "element"), 0, 1)
		}
		_ = cfg.Schemas.Add(istructs.NewQName("test", "element"), istructs.SchemaKind_Element)

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, ErrContainerNameIDsExceeds)
	})

	t.Run("must error if retrieve ID for some container name from storage is failed", func(t *testing.T) {
		containerName := "ErrorContainerName"
		testError := fmt.Errorf("test error")

		storage.sheduleGetError(testError, nil, []byte(containerName))
		storage.shedulePutError(testError, nil, []byte(containerName))
		defer storage.reset()

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)
		schema := cfg.Schemas.Add(istructs.NewQName("test", "object"), istructs.SchemaKind_Object)
		schema.AddContainer(containerName, istructs.NewQName("test", "element"), 0, 1)
		_ = cfg.Schemas.Add(istructs.NewQName("test", "element"), istructs.SchemaKind_Element)

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if some some container name from storage is not valid identifier", func(t *testing.T) {

		t.Run("crack storage by put invalid container identifier into sys.Containers view", func(t *testing.T) {
			err := storage.Put(
				toBytes(uint16(QNameIDSysVesions)),
				toBytes(uint16(verSysContainers)),
				toBytes(uint16(verSysContainersLastest)),
			)
			require.NoError(err)

			err = storage.Put(
				toBytes(uint16(QNameIDSysContainers), uint16(verSysContainersLastest)),
				[]byte("error-container-name"),
				toBytes(uint16(0xFFFE)),
			)
			require.NoError(err)
		})

		cfgs := make(AppConfigsType, 1)
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1)

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, ErrInvalidName)
	})
}

func Test_containerNameCache_ReuseStorage(t *testing.T) {

	require := require.New(t)

	appCfgWithStorage := func(storage istorage.IAppStorage) *AppConfigType {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(istructs.AppQName_test1_app1)

		schema := cfg.Schemas.Add(istructs.NewQName("test", "object"), istructs.SchemaKind_Object)
		schema.AddContainer("element", istructs.NewQName("test", "element"), 0, 1)
		_ = cfg.Schemas.Add(istructs.NewQName("test", "element"), istructs.SchemaKind_Element)

		err := cfg.prepare(nil, storage)
		require.NoError(err)

		return cfg
	}

	appCfg := func() *AppConfigType {
		asf := istorage.ProvideMem()
		sp := istorageimpl.Provide(asf)
		appStorage, err := sp.AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)
		return appCfgWithStorage(appStorage)
	}

	t.Run("must use equal container id if storage reused", func(t *testing.T) {
		cfg1 := appCfg()
		id1, err := cfg1.cNames.nameToID("element")
		require.NoError(err)

		cfg2 := appCfgWithStorage(cfg1.storage)
		id2, err := cfg2.cNames.nameToID("element")
		require.NoError(err)

		require.Equal(id1, id2)
	})

}
