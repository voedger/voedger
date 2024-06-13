/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"errors"
	"log"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

func TestAppConfigsType_AddConfig(t *testing.T) {
	require := require.New(t)

	appName, appID := istructs.AppQName_test1_app1, istructs.ClusterAppID_test1_app1

	t.Run("must be ok to add config for known app", func(t *testing.T) {
		cfgs := make(AppConfigsType)
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")
		cfg := cfgs.AddConfig(appName, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		require.Equal(cfg.Name, appName)
		require.Equal(cfg.ClusterAppID, appID)
		require.Equal(istructs.DefaultNumAppWorkspaces, cfg.NumAppWorkspaces())

		_, storageProvider := teststore.New(appName)
		appStructs := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)

		t.Run("must be ok to change appDef after add config", func(t *testing.T) {
			docName := appdef.NewQName("test", "doc")
			doc := adb.AddCDoc(docName)
			doc.AddField("field", appdef.DataKind_int64, true)
			doc.SetSingleton()
			appStr, err := appStructs.AppStructs(appName)
			require.NoError(err)
			require.NotNil(appStr)
			t.Run("should be ok to retrieve changed doc from AppStructs", func(t *testing.T) {
				doc := appStr.AppDef().CDoc(docName)
				require.Equal(docName, doc.QName())
				require.True(doc.Singleton())
				require.Equal(appdef.DataKind_int64, doc.Field("field").DataKind())
			})
		})
	})

	t.Run("misc", func(t *testing.T) {
		cfgs := make(AppConfigsType)
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")
		cfg := cfgs.AddConfig(appName, adb)
		cfg.SetNumAppWorkspaces(42)
		_, storageProvider := teststore.New(appName)
		appStructs := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		as, err := appStructs.AppStructs(appName)
		require.NoError(err)
		require.Equal(istructs.NumAppWorkspaces(42), as.NumAppWorkspaces())
	})

	t.Run("must be error to make invalid changes in appDef after add config", func(t *testing.T) {
		cfgs := make(AppConfigsType)
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		cfgs.AddConfig(appName, adb).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

		adb.AddObject(appdef.NewQName("test", "obj")).
			AddContainer("unknown", appdef.NewQName("test", "unknown"), 0, 1) // <- error here: reference to unknown element type

		_, storageProvider := teststore.New(appName)
		appStructs := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)

		appStr, err := appStructs.AppStructs(appName)
		require.Nil(appStr)
		require.ErrorIs(err, appdef.ErrNotFoundError)
	})

	t.Run("must be panic to add config for unknown app", func(t *testing.T) {
		cfgs := make(AppConfigsType)
		appName := appdef.NewAppQName("unknown", "unknown")
		require.Panics(func() {
			cfgs.AddConfig(appName, appdef.New()).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		}, require.Is(istructs.ErrAppNotFound), require.Has(appName))
	})

	t.Run("must be panic to add config with invalid appDef", func(t *testing.T) {
		cfgs := make(AppConfigsType)

		require.Panics(func() {
			_ = cfgs.AddConfig(appName,
				func() appdef.IAppDefBuilder {
					adb := appdef.New()
					adb.AddPackage("test", "test.com/test")
					adb.AddObject(appdef.NewQName("test", "obj")).
						AddContainer("unknown", appdef.NewQName("test", "unknown"), 0, 1) // <- error here: reference to unknown element type
					return adb
				}())
		}, require.Is(appdef.ErrNotFoundError), require.Has(appName), require.Has("test.unknown"))
	})
}

func TestAppConfigsType_GetConfig(t *testing.T) {
	require := require.New(t)

	asf := mem.Provide()
	storages := istorageimpl.Provide(asf)

	cfgs := make(AppConfigsType)
	for app, id := range istructs.ClusterApps {
		cfg := cfgs.AddConfig(app, appdef.New())
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		require.NotNil(cfg)
		require.Equal(cfg.Name, app)
		require.Equal(cfg.ClusterAppID, id)
	}

	appStructsProvider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storages)

	t.Run("must be ok to create configs for all known cluster apps", func(t *testing.T) {
		for app := range istructs.ClusterApps {
			appStructs, err := appStructsProvider.AppStructs(app)
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
		appStructs, err := appStructsProvider.AppStructs(app)
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

	docName, recName := appdef.NewQName("test", "doc"), appdef.NewQName("test", "rec")

	appDef := func() appdef.IAppDefBuilder {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")
		doc := adb.AddCDoc(docName)
		doc.SetSingleton()
		doc.AddField("f1", appdef.DataKind_string, true)
		doc.AddContainer("rec", recName, 0, 1)
		doc.AddUnique(appdef.UniqueQName(docName, "f1"), []appdef.FieldName{"f1"})
		adb.AddCRecord(recName)
		return adb
	}()

	storage, storageProvider := teststore.New(appName)

	t.Run("must be ok to provide app structure", func(t *testing.T) {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(appName, appDef)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)

		as, err := provider.AppStructs(appName)
		require.NoError(err)
		require.NotNil(as)
		require.Equal(docName, as.AppDef().CDoc(docName).QName())
		require.Equal(recName, as.AppDef().CRecord(recName).QName())
	})

	t.Run("must be error to provide app structure if error while read versions", func(t *testing.T) {
		testError := errors.New("test error")
		storage.ScheduleGetError(testError, utils.ToBytes(consts.SysView_Versions), nil) // error here
		defer storage.Reset()

		cfgs := make(AppConfigsType, 1)
		cfgs.AddConfig(appName, appDef).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		_, err := provider.AppStructs(appName)
		require.ErrorIs(err, testError)
	})

	t.Run("must be error to provide app structure if error while read QNames system view", func(t *testing.T) {
		storage.ScheduleGetDamage(
			func(b *[]byte) { (*b)[0] = 255 }, // <- invalid QNames system view version
			utils.ToBytes(consts.SysView_Versions),
			utils.ToBytes(vers.SysQNamesVersion))
		defer storage.Reset()

		cfgs := make(AppConfigsType, 1)
		cfgs.AddConfig(appName, appDef).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		_, err := provider.AppStructs(appName)
		require.ErrorIs(err, vers.ErrorInvalidVersion)
	})

	t.Run("must be error to provide app structure if error while read Containers system view", func(t *testing.T) {
		storage.ScheduleGetDamage(
			func(b *[]byte) { (*b)[0] = 255 }, // <- invalid Containers system view version
			utils.ToBytes(consts.SysView_Versions),
			utils.ToBytes(vers.SysContainersVersion))
		defer storage.Reset()

		cfgs := make(AppConfigsType, 1)
		cfgs.AddConfig(appName, appDef).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		_, err := provider.AppStructs(appName)
		require.ErrorIs(err, vers.ErrorInvalidVersion)
	})

	t.Run("must be error to provide app structure if error while read Singletons system view", func(t *testing.T) {
		storage.ScheduleGetDamage(
			func(b *[]byte) { (*b)[0] = 255 }, // <- invalid Containers system view version
			utils.ToBytes(consts.SysView_Versions),
			utils.ToBytes(vers.SysSingletonsVersion))
		defer storage.Reset()

		cfgs := make(AppConfigsType, 1)
		cfgs.AddConfig(appName, appDef).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		_, err := provider.AppStructs(appName)
		require.ErrorIs(err, vers.ErrorInvalidVersion)
	})
	t.Run("resources validation", func(t *testing.T) {
		qName := appdef.NewQName("test", "qname")
		t.Run("query", func(t *testing.T) {
			t.Run("missing in cfg", func(t *testing.T) {
				adb := appdef.New()
				adb.AddPackage("test", "test.com/test")
				adb.AddQuery(qName)
				cfgs := make(AppConfigsType, 1)
				cfgs.AddConfig(appName, adb).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
				provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
				_, err := provider.AppStructs(appName)
				require.Error(err)
				log.Println(err)
			})
			t.Run("missing in AppDef", func(t *testing.T) {
				adb := appdef.New()
				cfgs := make(AppConfigsType, 1)
				cfg := cfgs.AddConfig(appName, adb)
				cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
				cfg.Resources.Add(NewQueryFunction(qName, nil))
				provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
				_, err := provider.AppStructs(appName)
				require.Error(err)
				log.Println(err)
			})
		})
		t.Run("command", func(t *testing.T) {
			t.Run("missing in cfg", func(t *testing.T) {
				adb := appdef.New()
				adb.AddPackage("test", "test.com/test")
				adb.AddCommand(qName)
				cfgs := make(AppConfigsType, 1)
				cfgs.AddConfig(appName, adb).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
				provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
				_, err := provider.AppStructs(appName)
				require.Error(err)
				log.Println(err)
			})
			t.Run("missing in AppDef", func(t *testing.T) {
				adb := appdef.New()
				cfgs := make(AppConfigsType, 1)
				cfg := cfgs.AddConfig(appName, adb)
				cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
				cfg.Resources.Add(NewCommandFunction(qName, nil))
				provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
				_, err := provider.AppStructs(appName)
				require.Error(err)
				log.Println(err)
			})
		})
		t.Run("projectors", func(t *testing.T) {
			t.Run("sync", func(t *testing.T) {
				t.Run("missing in cfg", func(t *testing.T) {
					adb := appdef.New()
					adb.AddPackage("test", "test.com/test")
					qName2 := appdef.NewQName("test", "qName2")
					adb.AddCDoc(qName2)
					adb.AddProjector(qName).
						SetSync(true).
						Events().Add(qName2, appdef.ProjectorEventKind_Insert)
					cfgs := make(AppConfigsType, 1)
					cfgs.AddConfig(appName, adb).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
					provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
					_, err := provider.AppStructs(appName)
					require.Error(err)
					log.Println(err)
				})
				t.Run("missing in AppDef", func(t *testing.T) {
					adb := appdef.New()
					cfgs := make(AppConfigsType, 1)
					cfg := cfgs.AddConfig(appName, adb)
					cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
					cfg.AddSyncProjectors(istructs.Projector{Name: qName})
					provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
					_, err := provider.AppStructs(appName)
					require.Error(err)
					log.Println(err)
				})
				t.Run("defined as async in cfg", func(t *testing.T) {
					adb := appdef.New()
					adb.AddPackage("test", "test.com/test")
					qName2 := appdef.NewQName("test", "qName2")
					adb.AddCDoc(qName2)
					adb.AddProjector(qName).
						SetSync(true).
						Events().Add(qName2, appdef.ProjectorEventKind_Insert)
					cfgs := make(AppConfigsType, 1)
					cfg := cfgs.AddConfig(appName, adb)
					cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
					cfg.AddAsyncProjectors(istructs.Projector{Name: qName})
					provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
					_, err := provider.AppStructs(appName)
					require.Error(err)
					log.Println(err)
				})
			})
			t.Run("async", func(t *testing.T) {
				t.Run("missing in cfg", func(t *testing.T) {
					adb := appdef.New()
					adb.AddPackage("test", "test.com/test")
					qName2 := appdef.NewQName("test", "qName2")
					adb.AddCDoc(qName2)
					adb.AddProjector(qName).
						SetSync(false).
						Events().Add(qName2, appdef.ProjectorEventKind_Insert)
					cfgs := make(AppConfigsType, 1)
					cfgs.AddConfig(appName, adb).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
					provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
					_, err := provider.AppStructs(appName)
					require.Error(err)
					log.Println(err)
				})
				t.Run("missing in AppDef", func(t *testing.T) {
					adb := appdef.New()
					cfgs := make(AppConfigsType, 1)
					cfg := cfgs.AddConfig(appName, adb)
					cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
					cfg.AddAsyncProjectors(istructs.Projector{Name: qName})
					provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
					_, err := provider.AppStructs(appName)
					require.Error(err)
					log.Println(err)
				})
				t.Run("defined as sync in cfg", func(t *testing.T) {
					adb := appdef.New()
					adb.AddPackage("test", "test.com/test")
					qName2 := appdef.NewQName("test", "qName2")
					adb.AddCDoc(qName2)
					adb.AddProjector(qName).
						SetSync(false).
						Events().Add(qName2, appdef.ProjectorEventKind_Insert)
					cfgs := make(AppConfigsType, 1)
					cfg := cfgs.AddConfig(appName, adb)
					cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
					cfg.AddSyncProjectors(istructs.Projector{Name: qName})
					provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
					_, err := provider.AppStructs(appName)
					require.Error(err)
					log.Println(err)
				})
				t.Run("defined twice in cfg", func(t *testing.T) {
					adb := appdef.New()
					adb.AddPackage("test", "test.com/test")
					qName2 := appdef.NewQName("test", "qName2")
					adb.AddCDoc(qName2)
					adb.AddProjector(qName).
						SetSync(true).
						Events().Add(qName2, appdef.ProjectorEventKind_Insert)
					cfgs := make(AppConfigsType, 1)
					cfg := cfgs.AddConfig(appName, adb)
					cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
					cfg.AddAsyncProjectors(istructs.Projector{Name: qName})
					cfg.AddSyncProjectors(istructs.Projector{Name: qName})
					provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
					_, err := provider.AppStructs(appName)
					require.Error(err)
					log.Println(err)
				})
			})
		})
	})
	t.Run("unable set NumAppPartititons after prepare()", func(t *testing.T) {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(appName, appDef)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		_, err := provider.AppStructs(appName)
		require.NoError(err)
		require.Panics(func() { cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces) })
	})

	t.Run("unable to use IAppDefBuilder after prepare()", func(t *testing.T) {
		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddConfig(appName, appDef)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		_, err := provider.AppStructs(appName)
		require.NoError(err)
		require.Panics(func() { cfg.AppDefBuilder() })
	})

	t.Run("unable to work is NumAppWorkspaces is not set", func(t *testing.T) {
		cfgs := make(AppConfigsType, 1)
		cfgs.AddConfig(appName, appDef)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		_, err := provider.AppStructs(appName)
		require.ErrorIs(err, ErrNumAppWorkspacesNotSet)
	})
}
