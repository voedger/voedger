/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"errors"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/isequencer"
	istorage "github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

func TestAppConfigsType_AddBuiltInConfig(t *testing.T) {
	require := require.New(t)

	appName, appID := istructs.AppQName_test1_app1, istructs.ClusterAppID_test1_app1

	t.Run("should be ok to add config for known builtin app", func(t *testing.T) {
		cfgs := make(AppConfigsType)
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		cfg := cfgs.AddBuiltInAppConfig(appName, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		require.Equal(cfg.Name, appName)
		require.Equal(cfg.ClusterAppID, appID)
		require.Equal(istructs.DefaultNumAppWorkspaces, cfg.NumAppWorkspaces())

		_, storageProvider := teststore.New(appName)
		appStructs := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider, isequencer.SequencesTrustLevel_0, nil)

		t.Run("should be ok to change appDef after add config", func(t *testing.T) {
			wsName := appdef.NewQName("test", "workspace")
			ws := adb.AddWorkspace(wsName)
			wsDescQName := appdef.NewQName("test", "WSDesc")
			ws.AddCDoc(wsDescQName)
			ws.SetDescriptor(wsDescQName)
			docName := appdef.NewQName("test", "doc")
			doc := ws.AddCDoc(docName)
			doc.AddField("field", appdef.DataKind_int64, true)
			doc.SetSingleton()
			appStr, err := appStructs.BuiltIn(appName)
			require.NoError(err)
			require.NotNil(appStr)
			t.Run("should be ok to retrieve changed doc from AppStructs", func(t *testing.T) {
				doc := appdef.CDoc(appStr.AppDef().Type, docName)
				require.Equal(docName, doc.QName())
				require.True(doc.Singleton())
				require.Equal(appdef.DataKind_int64, doc.Field("field").DataKind())
			})
		})
	})

	t.Run("misc", func(t *testing.T) {
		cfgs := make(AppConfigsType)
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		cfg := cfgs.AddBuiltInAppConfig(appName, adb)
		cfg.SetNumAppWorkspaces(42)
		_, storageProvider := teststore.New(appName)
		appStructs := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider, isequencer.SequencesTrustLevel_0, nil)
		as, err := appStructs.BuiltIn(appName)
		require.NoError(err)
		require.Equal(istructs.NumAppWorkspaces(42), as.NumAppWorkspaces())
	})

	t.Run("should be error to make invalid changes in appDef after add config", func(t *testing.T) {
		cfgs := make(AppConfigsType)
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		cfgs.AddBuiltInAppConfig(appName, adb).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		wsb.AddObject(appdef.NewQName("test", "obj")).
			AddContainer("unknown", appdef.NewQName("test", "unknown"), 0, 1) // <- error here: reference to unknown element type

		_, storageProvider := teststore.New(appName)
		appStructs := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider, isequencer.SequencesTrustLevel_0, nil)

		appStr, err := appStructs.BuiltIn(appName)
		require.Nil(appStr)
		require.ErrorIs(err, appdef.ErrNotFoundError)
	})

	t.Run("should be panics", func(t *testing.T) {
		t.Run("if add config for unknown builtin app", func(t *testing.T) {
			cfgs := make(AppConfigsType)
			appName := appdef.NewAppQName("unknown", "unknown")
			require.Panics(func() {
				cfgs.AddBuiltInAppConfig(appName, builder.New()).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
			}, require.Is(istructs.ErrAppNotFound), require.Has(appName))
		})

		t.Run("if add config with invalid appDef", func(t *testing.T) {
			cfgs := make(AppConfigsType)

			require.Panics(func() {
				_ = cfgs.AddBuiltInAppConfig(appName,
					func() appdef.IAppDefBuilder {
						adb := builder.New()
						adb.AddPackage("test", "test.com/test")
						wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
						wsb.AddObject(appdef.NewQName("test", "obj")).
							AddContainer("unknown", appdef.NewQName("test", "unknown"), 0, 1) // <- error here: reference to unknown element type
						return adb
					}())
			}, require.Is(appdef.ErrNotFoundError), require.Has(appName), require.Has("test.unknown"))
		})
	})
}

func TestAppConfigsType_GetConfig(t *testing.T) {
	require := require.New(t)

	asf := mem.Provide(testingu.MockTime)
	storages := istorageimpl.Provide(asf)

	cfgs := make(AppConfigsType)
	for app, id := range istructs.ClusterApps {
		cfg := cfgs.AddBuiltInAppConfig(app, builder.New())
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		require.NotNil(cfg)
		require.Equal(cfg.Name, app)
		require.Equal(cfg.ClusterAppID, id)
	}

	appStructsProvider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storages, isequencer.SequencesTrustLevel_0, nil)

	t.Run("must be ok to create configs for all known cluster apps", func(t *testing.T) {
		for app := range istructs.ClusterApps {
			appStructs, err := appStructsProvider.BuiltIn(app)
			require.NotNil(appStructs)
			require.NoError(err)
		}
	})

	t.Run("must be ok to retrieve configs for all known cluster apps", func(t *testing.T) {
		for app, id := range istructs.ClusterApps {
			cfg := cfgs.GetConfig(app)
			require.NotNil(cfg)
			require.Equal(cfg.Name, app)
			require.Equal(cfg.ClusterAppID, id)

			storage, err := storages.AppStorage(app)
			require.NotNil(storage)
			require.NoError(err)
			require.Equal(cfg.storage, storage)
		}
	})

	t.Run("must be error to create config for unknown app", func(t *testing.T) {
		app := appdef.NewAppQName("unknownOwner", "unknownApplication")
		appStructs, err := appStructsProvider.BuiltIn(app)
		require.Nil(appStructs)
		require.ErrorIs(err, istructs.ErrAppNotFound)
	})

	t.Run("must be panic to retrieve config for unknown app", func(t *testing.T) {
		app := appdef.NewAppQName("unknownOwner", "unknownApplication")
		require.Panics(func() {
			_ = cfgs.GetConfig(app)
		}, require.Is(istructs.ErrAppNotFound), require.Has(app))
	})

}

func TestErrorsAppConfigsType(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	wsName := appdef.NewQName("test", "workspace")
	docName, recName := appdef.NewQName("test", "doc"), appdef.NewQName("test", "rec")

	appDef := func() appdef.IAppDefBuilder {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		ws := adb.AddWorkspace(wsName)
		wsDescQName := appdef.NewQName("test", "WSDesc")
		ws.AddCDoc(wsDescQName)
		ws.SetDescriptor(wsDescQName)
		doc := ws.AddCDoc(docName)
		doc.SetSingleton()
		doc.AddField("f1", appdef.DataKind_string, true)
		doc.AddContainer("rec", recName, 0, 1)
		doc.AddUnique(appdef.UniqueQName(docName, "f1"), []appdef.FieldName{"f1"})
		ws.AddCRecord(recName)
		return adb
	}()

	storage, storageProvider := teststore.New(appName)

	t.Run("must be ok to provide app structure", func(t *testing.T) {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddBuiltInAppConfig(appName, appDef)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider, isequencer.SequencesTrustLevel_0, nil)

		as, err := provider.BuiltIn(appName)
		require.NoError(err)
		require.NotNil(as)
		require.Equal(docName, appdef.CDoc(as.AppDef().Type, docName).QName())
		require.Equal(recName, appdef.CRecord(as.AppDef().Type, recName).QName())
	})

	t.Run("must be error to provide app structure if error while read versions", func(t *testing.T) {
		testError := errors.New("test error")
		storage.ScheduleGetError(testError, utils.ToBytes(consts.SysView_Versions), nil) // error here
		defer storage.Reset()

		cfgs := make(AppConfigsType, 1)
		cfgs.AddBuiltInAppConfig(appName, appDef).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider, isequencer.SequencesTrustLevel_0, nil)
		_, err := provider.BuiltIn(appName)
		require.ErrorIs(err, testError)
	})

	t.Run("must be error to provide app structure if error while read QNames system view", func(t *testing.T) {
		storage.ScheduleGetDamage(
			func(b *[]byte) { (*b)[0] = 255 }, // <- invalid QNames system view version
			utils.ToBytes(consts.SysView_Versions),
			utils.ToBytes(vers.SysQNamesVersion))
		defer storage.Reset()

		cfgs := make(AppConfigsType, 1)
		cfgs.AddBuiltInAppConfig(appName, appDef).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider, isequencer.SequencesTrustLevel_0, nil)
		_, err := provider.BuiltIn(appName)
		require.ErrorIs(err, vers.ErrorInvalidVersion)
	})

	t.Run("must be error to provide app structure if error while read Containers system view", func(t *testing.T) {
		storage.ScheduleGetDamage(
			func(b *[]byte) { (*b)[0] = 255 }, // <- invalid Containers system view version
			utils.ToBytes(consts.SysView_Versions),
			utils.ToBytes(vers.SysContainersVersion))
		defer storage.Reset()

		cfgs := make(AppConfigsType, 1)
		cfgs.AddBuiltInAppConfig(appName, appDef).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider, isequencer.SequencesTrustLevel_0, nil)
		_, err := provider.BuiltIn(appName)
		require.ErrorIs(err, vers.ErrorInvalidVersion)
	})

	t.Run("must be error to provide app structure if error while read Singletons system view", func(t *testing.T) {
		storage.ScheduleGetDamage(
			func(b *[]byte) { (*b)[0] = 255 }, // <- invalid Containers system view version
			utils.ToBytes(consts.SysView_Versions),
			utils.ToBytes(vers.SysSingletonsVersion))
		defer storage.Reset()

		cfgs := make(AppConfigsType, 1)
		cfgs.AddBuiltInAppConfig(appName, appDef).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider, isequencer.SequencesTrustLevel_0, nil)
		_, err := provider.BuiltIn(appName)
		require.ErrorIs(err, vers.ErrorInvalidVersion)
	})
	t.Run("unable set NumAppPartititons after prepare()", func(t *testing.T) {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddBuiltInAppConfig(appName, appDef)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider, isequencer.SequencesTrustLevel_0, nil)
		_, err := provider.BuiltIn(appName)
		require.NoError(err)
		require.Panics(func() { cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces) })
	})

	t.Run("unable to use IAppDefBuilder after prepare()", func(t *testing.T) {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddBuiltInAppConfig(appName, appDef)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider, isequencer.SequencesTrustLevel_0, nil)
		_, err := provider.BuiltIn(appName)
		require.NoError(err)
		require.Panics(func() { cfg.AppDefBuilder() })
	})

	t.Run("unable to work is NumAppWorkspaces is not set", func(t *testing.T) {
		cfgs := make(AppConfigsType, 1)
		cfgs.AddBuiltInAppConfig(appName, appDef)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider, isequencer.SequencesTrustLevel_0, nil)
		_, err := provider.BuiltIn(appName)
		require.Error(err, require.Is(ErrNumAppWorkspacesNotSetError), require.Has(appName))
	})
}

func Test_NewAppStructs(t *testing.T) {

	require := require.New(t)

	name := appdef.NewAppQName("test", "myApp")
	id := istructs.FirstGeneratedAppID + 1
	wsCount := istructs.NumAppWorkspaces(10)

	_, storageProvider := teststore.New(name)
	structs := Provide(make(AppConfigsType, 1), iratesce.TestBucketsFactory, testTokensFactory(), storageProvider, isequencer.SequencesTrustLevel_0, nil)

	t.Run("should be ok to create new AppStructs", func(t *testing.T) {
		def := builder.New().MustBuild()
		str, err := structs.New(name, def, id, wsCount)
		require.NoError(err)
		require.NotNil(str)

		t.Run("should be ok to check from new AppStructs", func(t *testing.T) {
			require.Equal(name, str.AppQName())
			require.Equal(def, str.AppDef())
			require.Equal(id, str.ClusterAppID())
			require.Equal(wsCount, str.NumAppWorkspaces())

			for resName := range str.Resources().Resources {
				require.Fail("unexpected resource", resName)
			}
			require.Empty(str.SyncProjectors())
			require.Empty(str.AsyncProjectors())
			require.Empty(str.CUDValidators())
			require.Empty(str.EventValidators())
		})
	})

	t.Run("should be error to create new AppStructs", func(t *testing.T) {

		t.Run("if storage is not exists", func(t *testing.T) {
			unknown := appdef.NewAppQName("unknown", "unknown")
			def := builder.New().MustBuild()
			str, err := structs.New(unknown, def, id, wsCount)
			require.Error(err,
				require.Is(istorage.ErrStorageDoesNotExist),
				require.Has(unknown))
			require.Nil(str)
		})

		t.Run("if workspaces count is omitted", func(t *testing.T) {
			def := builder.New().MustBuild()
			str, err := structs.New(name, def, id, 0)
			require.Error(err, require.Is(ErrNumAppWorkspacesNotSetError), require.Has(name))
			require.Nil(str)
		})

	})
}
