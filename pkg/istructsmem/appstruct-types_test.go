/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"errors"
	"log"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
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

	t.Run("must be ok to add config for known app", func(t *testing.T) {
		cfgs := make(AppConfigsType)
		app, id := istructs.AppQName_test1_app1, istructs.ClusterAppID_test1_app1
		appDef := appdef.New()
		cfg := cfgs.AddConfig(app, appDef)
		require.NotNil(cfg)
		require.Equal(cfg.Name, app)
		require.Equal(cfg.ClusterAppID, id)

		_, storageProvider := teststore.New()
		appStructs := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)

		t.Run("must be ok to change appDef after add config", func(t *testing.T) {
			appDef.AddSingleton(appdef.NewQName("test", "doc")).
				AddField("field", appdef.DataKind_int64, true)
			appStr, err := appStructs.AppStructs(app)
			require.NoError(err)
			require.NotNil(appStr)
		})
	})

	t.Run("must be error to make invalid changes in appDef after add config", func(t *testing.T) {
		cfgs := make(AppConfigsType)
		appDef := appdef.New()
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1, appDef)

		appDef.AddObject(appdef.NewQName("test", "obj")).
			AddContainer("unknown", appdef.NewQName("test", "unknown"), 0, 1) // <- error here: reference to unknown element type

		_, storageProvider := teststore.New()
		appStructs := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)

		appStr, err := appStructs.AppStructs(istructs.AppQName_test1_app1)
		require.Nil(appStr)
		require.ErrorIs(err, appdef.ErrNameNotFound)
	})

	t.Run("must be panic to add config for unknown app", func(t *testing.T) {
		cfgs := make(AppConfigsType)
		require.Panics(func() {
			_ = cfgs.AddConfig(istructs.NewAppQName("unknown", "unknown"), appdef.New())
		})
	})

	t.Run("must be panic to add config with invalid appDef", func(t *testing.T) {
		cfgs := make(AppConfigsType)

		require.Panics(func() {
			_ = cfgs.AddConfig(istructs.AppQName_test1_app1,
				func() appdef.IAppDefBuilder {
					app := appdef.New()
					app.AddObject(appdef.NewQName("test", "obj")).
						AddContainer("unknown", appdef.NewQName("test", "unknown"), 0, 1) // <- error here: reference to unknown element type
					return app
				}())
		})
	})
}

func TestAppConfigsType_GetConfig(t *testing.T) {
	require := require.New(t)

	asf := mem.Provide()
	storages := istorageimpl.Provide(asf)

	cfgs := make(AppConfigsType)
	for app, id := range istructs.ClusterApps {
		cfg := cfgs.AddConfig(app, appdef.New())
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
		app := istructs.NewAppQName("unknownOwner", "unknownApplication")
		appStructs, err := appStructsProvider.AppStructs(app)
		require.Nil(appStructs)
		require.ErrorIs(err, istructs.ErrAppNotFound)
	})

	t.Run("must be panic to retrieve config for unknown app", func(t *testing.T) {
		app := istructs.NewAppQName("unknownOwner", "unknownApplication")
		require.Panics(func() {
			_ = cfgs.GetConfig(app)
		})
	})

}

func TestErrorsAppConfigsType(t *testing.T) {
	require := require.New(t)

	appDef := func() appdef.IAppDefBuilder {
		app := appdef.New()
		doc := app.AddSingleton(appdef.NewQName("test", "doc"))
		doc.AddField("f1", appdef.DataKind_string, true)
		doc.AddContainer("rec", appdef.NewQName("test", "rec"), 0, 1)
		doc.AddUnique(appdef.UniqueQName(doc.QName(), "f1"), []string{"f1"})
		app.AddCRecord(appdef.NewQName("test", "rec"))
		return app
	}()

	storage, storageProvider := teststore.New()

	t.Run("must be ok to provide app structure", func(t *testing.T) {
		cfgs := make(AppConfigsType, 1)
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1, appDef)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)

		_, err := provider.AppStructs(istructs.AppQName_test1_app1)
		require.NoError(err)
	})

	t.Run("must be error to provide app structure if error while read versions", func(t *testing.T) {
		testError := errors.New("test error")
		storage.ScheduleGetError(testError, utils.ToBytes(consts.SysView_Versions), nil) // error here
		defer storage.Reset()

		cfgs := make(AppConfigsType, 1)
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1, appDef)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		_, err := provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, testError)
	})

	t.Run("must be error to provide app structure if error while read QNames system view", func(t *testing.T) {
		storage.ScheduleGetDamage(
			func(b *[]byte) { (*b)[0] = 255 }, // <- invalid QNames system view version
			utils.ToBytes(consts.SysView_Versions),
			utils.ToBytes(vers.SysQNamesVersion))
		defer storage.Reset()

		cfgs := make(AppConfigsType, 1)
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1, appDef)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		_, err := provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, vers.ErrorInvalidVersion)
	})

	t.Run("must be error to provide app structure if error while read Containers system view", func(t *testing.T) {
		storage.ScheduleGetDamage(
			func(b *[]byte) { (*b)[0] = 255 }, // <- invalid Containers system view version
			utils.ToBytes(consts.SysView_Versions),
			utils.ToBytes(vers.SysContainersVersion))
		defer storage.Reset()

		cfgs := make(AppConfigsType, 1)
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1, appDef)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		_, err := provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, vers.ErrorInvalidVersion)
	})

	t.Run("must be error to provide app structure if error while read Singletons system view", func(t *testing.T) {
		storage.ScheduleGetDamage(
			func(b *[]byte) { (*b)[0] = 255 }, // <- invalid Containers system view version
			utils.ToBytes(consts.SysView_Versions),
			utils.ToBytes(vers.SysSingletonsVersion))
		defer storage.Reset()

		cfgs := make(AppConfigsType, 1)
		_ = cfgs.AddConfig(istructs.AppQName_test1_app1, appDef)
		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
		_, err := provider.AppStructs(istructs.AppQName_test1_app1)
		require.ErrorIs(err, vers.ErrorInvalidVersion)
	})
	t.Run("resources validation", func(t *testing.T) {
		qName := appdef.NewQName("test", "qname")
		t.Run("query", func(t *testing.T) {
			t.Run("missing in cfg", func(t *testing.T) {
				adb := appdef.New()
				adb.AddQuery(qName)
				cfgs := make(AppConfigsType, 1)
				_ = cfgs.AddConfig(istructs.AppQName_test1_app1, adb)
				provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
				_, err := provider.AppStructs(istructs.AppQName_test1_app1)
				require.Error(err)
				log.Println(err)
			})
			t.Run("missing in AppDef", func(t *testing.T) {
				adb := appdef.New()
				cfgs := make(AppConfigsType, 1)
				cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, adb)
				cfg.Resources.Add(NewQueryFunction(qName, nil))
				provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
				_, err := provider.AppStructs(istructs.AppQName_test1_app1)
				require.Error(err)
				log.Println(err)
			})
		})
		t.Run("command", func(t *testing.T) {
			t.Run("missing in cfg", func(t *testing.T) {
				adb := appdef.New()
				adb.AddCommand(qName)
				cfgs := make(AppConfigsType, 1)
				_ = cfgs.AddConfig(istructs.AppQName_test1_app1, adb)
				provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
				_, err := provider.AppStructs(istructs.AppQName_test1_app1)
				require.Error(err)
				log.Println(err)
			})
			t.Run("missing in AppDef", func(t *testing.T) {
				adb := appdef.New()
				cfgs := make(AppConfigsType, 1)
				cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, adb)
				cfg.Resources.Add(NewCommandFunction(qName, nil))
				provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
				_, err := provider.AppStructs(istructs.AppQName_test1_app1)
				require.Error(err)
				log.Println(err)
			})
		})
		t.Run("projectors", func(t *testing.T) {
			t.Run("sync", func(t *testing.T) {
				t.Run("missing in cfg", func(t *testing.T) {
					adb := appdef.New()
					qName2 := appdef.NewQName("test", "qName2")
					adb.AddCDoc(qName2)
					adb.AddProjector(qName).
						SetSync(true).
						AddEvent(qName2, appdef.ProjectorEventKind_Insert)
					cfgs := make(AppConfigsType, 1)
					_ = cfgs.AddConfig(istructs.AppQName_test1_app1, adb)
					provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
					_, err := provider.AppStructs(istructs.AppQName_test1_app1)
					require.Error(err)
					log.Println(err)
				})
				t.Run("missing in AppDef", func(t *testing.T) {
					adb := appdef.New()
					cfgs := make(AppConfigsType, 1)
					cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, adb)
					cfg.AddSyncProjectors(func(partition istructs.PartitionID) istructs.Projector {
						return istructs.Projector{
							Name: qName,
						}
					})
					provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
					_, err := provider.AppStructs(istructs.AppQName_test1_app1)
					require.Error(err)
					log.Println(err)
				})
				t.Run("defined as async in cfg", func(t *testing.T) {
					adb := appdef.New()
					qName2 := appdef.NewQName("test", "qName2")
					adb.AddCDoc(qName2)
					adb.AddProjector(qName).
						SetSync(true).
						AddEvent(qName2, appdef.ProjectorEventKind_Insert)
					cfgs := make(AppConfigsType, 1)
					cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, adb)
					cfg.AddAsyncProjectors(func(partition istructs.PartitionID) istructs.Projector {
						return istructs.Projector{
							Name: qName,
						}
					})
					provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
					_, err := provider.AppStructs(istructs.AppQName_test1_app1)
					require.Error(err)
					log.Println(err)
				})
			})
			t.Run("async", func(t *testing.T) {
				t.Run("missing in cfg", func(t *testing.T) {
					adb := appdef.New()
					qName2 := appdef.NewQName("test", "qName2")
					adb.AddCDoc(qName2)
					adb.AddProjector(qName).
						SetSync(false).
						AddEvent(qName2, appdef.ProjectorEventKind_Insert)
					cfgs := make(AppConfigsType, 1)
					_ = cfgs.AddConfig(istructs.AppQName_test1_app1, adb)
					provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
					_, err := provider.AppStructs(istructs.AppQName_test1_app1)
					require.Error(err)
					log.Println(err)
				})
				t.Run("missing in AppDef", func(t *testing.T) {
					adb := appdef.New()
					cfgs := make(AppConfigsType, 1)
					cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, adb)
					cfg.AddAsyncProjectors(func(partition istructs.PartitionID) istructs.Projector {
						return istructs.Projector{
							Name: qName,
						}
					})
					provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
					_, err := provider.AppStructs(istructs.AppQName_test1_app1)
					require.Error(err)
					log.Println(err)
				})
				t.Run("defined as sync in cfg", func(t *testing.T) {
					adb := appdef.New()
					qName2 := appdef.NewQName("test", "qName2")
					adb.AddCDoc(qName2)
					adb.AddProjector(qName).
						SetSync(false).
						AddEvent(qName2, appdef.ProjectorEventKind_Insert)
					cfgs := make(AppConfigsType, 1)
					cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, adb)
					cfg.AddSyncProjectors(func(partition istructs.PartitionID) istructs.Projector {
						return istructs.Projector{
							Name: qName,
						}
					})
					provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
					_, err := provider.AppStructs(istructs.AppQName_test1_app1)
					require.Error(err)
					log.Println(err)
				})
				t.Run("defined twice in cfg", func(t *testing.T) {
					adb := appdef.New()
					qName2 := appdef.NewQName("test", "qName2")
					adb.AddCDoc(qName2)
					adb.AddProjector(qName).
						SetSync(true).
						AddEvent(qName2, appdef.ProjectorEventKind_Insert)
					cfgs := make(AppConfigsType, 1)
					cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, adb)
					cfg.AddAsyncProjectors(func(partition istructs.PartitionID) istructs.Projector {
						return istructs.Projector{
							Name: qName,
						}
					})
					cfg.AddSyncProjectors(func(partition istructs.PartitionID) istructs.Projector {
						return istructs.Projector{
							Name: qName,
						}
					})
					provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), storageProvider)
					_, err := provider.AppStructs(istructs.AppQName_test1_app1)
					require.Error(err)
					log.Println(err)
				})
			})
		})
	})
}
