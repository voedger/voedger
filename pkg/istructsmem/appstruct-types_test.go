/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
	"github.com/voedger/voedger/pkg/schemas"
)

func TestAppConfigsType(t *testing.T) {

	require := require.New(t)

	asf := istorage.ProvideMem()
	storages := istorageimpl.Provide(asf)

	cfgs := make(AppConfigsType)
	for app, id := range istructs.ClusterApps {
		cfg := cfgs.AddConfig(app, schemas.NewSchemaCache())
		require.NotNil(cfg)
		require.Equal(cfg.Name, app)
		require.Equal(cfg.QNameID, id)
	}

	appStructsProvider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storages)
	require.NoError(err)

	t.Run("Create configs for all known cluster apps", func(t *testing.T) {

		for app := range istructs.ClusterApps {
			appStructs, err := appStructsProvider.AppStructs(app)
			require.NotNil(appStructs)
			require.NoError(err)
		}
	})

	t.Run("Panic if create config for unknown app", func(t *testing.T) {
		app := istructs.NewAppQName("unknownOwner", "unknownApplication")
		appStructs, err := appStructsProvider.AppStructs(app)
		require.Nil(appStructs)
		require.ErrorIs(err, istructs.ErrAppNotFound)
	})

	t.Run("Retrieve configs for all known cluster apps", func(t *testing.T) {
		for app, id := range istructs.ClusterApps {
			cfg := cfgs.GetConfig(app)
			require.NotNil(cfg)
			require.Equal(cfg.Name, app)
			require.Equal(cfg.QNameID, id)

			storage, err := storages.AppStorage(app)
			require.NotNil(storage)
			require.NoError(err)
			require.Equal(cfg.storage, storage)
		}
	})

	t.Run("Panic if retrieve config for unknown app", func(t *testing.T) {
		app := istructs.NewAppQName("unknownOwner", "unknownApplication")
		require.Panics(func() {
			_ = cfgs.GetConfig(app)
		})
	})

}

func TestErrorsAppConfigsType(t *testing.T) {
	require := require.New(t)

	storage := newTestStorage()
	storageProvider := newTestStorageProvider(storage)

	t.Run("must error if error while read versions", func(t *testing.T) {
		schemas := schemas.NewSchemaCache()
		t.Run("must be ok to build schemas", func(t *testing.T) {
			schemas.Add(istructs.NewQName("test", "CDoc"), istructs.SchemaKind_CDoc)
			require.NoError(schemas.ValidateSchemas())
		})

		cfgs1 := make(AppConfigsType, 1)
		_ = cfgs1.AddConfig(istructs.AppQName_test1_app1, schemas)
		provider1, err := Provide(cfgs1, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider1.AppStructs(istructs.AppQName_test1_app1)
		require.NoError(err, err)

		testError := errors.New("test error")
		pKey := utils.ToBytes(consts.SysView_Versions)
		storage.sheduleGetError(testError, pKey, nil) // error here
		defer storage.reset()

		cfgs2 := make(AppConfigsType, 1)
		_ = cfgs2.AddConfig(istructs.AppQName_test1_app1, schemas)
		provider2, err := Provide(cfgs2, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)
		_, err = provider2.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if damaged data while read versions", func(t *testing.T) {
		cfgs1 := make(AppConfigsType, 1)
		_ = cfgs1.AddConfig(istructs.AppQName_test1_app1, schemas.NewSchemaCache())
		provider1, err := Provide(cfgs1, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)

		_, err = provider1.AppStructs(istructs.AppQName_test1_app1)
		require.NoError(err, err)

		pKey := utils.ToBytes(consts.SysView_Versions)
		storage.sheduleGetDamage(func(b *[]byte) { (*b)[0] = 255 /* error here */ }, pKey, nil)

		cfgs2 := make(AppConfigsType, 1)
		_ = cfgs2.AddConfig(istructs.AppQName_test1_app1, schemas.NewSchemaCache())
		provider2, err := Provide(cfgs2, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		require.NoError(err)
		_, err = provider2.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, vers.ErrorInvalidVersion)
	})
}
