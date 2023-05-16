/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

func TestAppConfigsType(t *testing.T) {

	require := require.New(t)

	asf := istorage.ProvideMem()
	storages := istorageimpl.Provide(asf)

	cfgs := make(AppConfigsType)
	for app, id := range istructs.ClusterApps {
		cfg := cfgs.AddConfig(app, appdef.New())
		require.NotNil(cfg)
		require.Equal(cfg.Name, app)
		require.Equal(cfg.QNameID, id)
	}

	appStructsProvider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storages)

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

	storage := teststore.NewStorage()
	storageProvider := teststore.NewStorageProvider(storage)

	t.Run("must error if error while read versions", func(t *testing.T) {
		bld := appdef.New()
		t.Run("must be ok to build application definition", func(t *testing.T) {
			bld.AddCDoc(appdef.NewQName("test", "CDoc"))
		})

		cfgs1 := make(AppConfigsType, 1)
		_ = cfgs1.AddConfig(istructs.AppQName_test1_app1, bld)
		provider1 := Provide(cfgs1, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)

		_, err := provider1.AppStructs(istructs.AppQName_test1_app1)
		require.NoError(err)

		testError := errors.New("test error")
		pKey := utils.ToBytes(consts.SysView_Versions)
		storage.ScheduleGetError(testError, pKey, nil) // error here
		defer storage.Reset()

		cfgs2 := make(AppConfigsType, 1)
		_ = cfgs2.AddConfig(istructs.AppQName_test1_app1, bld)
		provider2 := Provide(cfgs2, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		_, err = provider2.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if damaged data while read versions", func(t *testing.T) {
		cfgs1 := make(AppConfigsType, 1)
		_ = cfgs1.AddConfig(istructs.AppQName_test1_app1, appdef.New())
		provider1 := Provide(cfgs1, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)

		_, err := provider1.AppStructs(istructs.AppQName_test1_app1)
		require.NoError(err)

		pKey := utils.ToBytes(consts.SysView_Versions)
		storage.ScheduleGetDamage(func(b *[]byte) { (*b)[0] = 255 /* error here */ }, pKey, nil)

		cfgs2 := make(AppConfigsType, 1)
		_ = cfgs2.AddConfig(istructs.AppQName_test1_app1, appdef.New())
		provider2 := Provide(cfgs2, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		_, err = provider2.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, vers.ErrorInvalidVersion)
	})
}
