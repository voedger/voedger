/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package singletons

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

func Test_BasicUsage(t *testing.T) {
	sp := istorageimpl.Provide(istorage.ProvideMem())
	storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

	versions := vers.New()
	if err := versions.Prepare(storage); err != nil {
		panic(err)
	}

	testName := appdef.NewQName("test", "doc")
	app := appdef.New()
	app.Add(testName, appdef.DefKind_CDoc).SetSingleton()
	appDef, err := app.Build()
	if err != nil {
		panic(err)
	}

	stones := New()
	if err := stones.Prepare(storage, versions, appDef); err != nil {
		panic(err)
	}

	require := require.New(t)
	t.Run("basic Singletons methods", func(t *testing.T) {
		id, err := stones.GetID(testName)
		require.NoError(err)
		require.NotEqual(istructs.NullRecordID, id)

		n, err := stones.GetQName(id)
		require.NoError(err)
		require.Equal(testName, n)

		t.Run("must be able to load early stored names", func(t *testing.T) {
			otherVersions := vers.New()
			if err := otherVersions.Prepare(storage); err != nil {
				panic(err)
			}

			stones1 := New()
			if err := stones1.Prepare(storage, versions, nil); err != nil {
				panic(err)
			}

			id1, err := stones.GetID(testName)
			require.NoError(err)
			require.Equal(id, id1)

			n1, err := stones.GetQName(id)
			require.NoError(err)
			require.Equal(testName, n1)
		})
	})
}

func test_AppDefSingletons(t *testing.T, appDef appdef.IAppDef, stons *Singletons) {
	require := require.New(t)
	appDef.Defs(
		func(d appdef.IDef) {
			if d.Singleton() {
				id, err := stons.GetID(d.QName())
				require.NoError(err)
				require.NotEqual(istructs.NullRecordID, id)
				name, err := stons.GetQName(id)
				require.NoError(err)
				require.Equal(d.QName(), name)
			}
		})
}

func Test_SingletonsGetID(t *testing.T) {

	require := require.New(t)
	cDocName := appdef.NewQName("test", "SignletonCDoc")

	stons := New()

	t.Run("must be ok to construct Singletons", func(t *testing.T) {
		storage, versions, appDef := func() (istorage.IAppStorage, *vers.Versions, appdef.IAppDef) {
			storage, err := istorageimpl.Provide(istorage.ProvideMem()).AppStorage(istructs.AppQName_test1_app1)
			require.NoError(err)

			versions := vers.New()
			err = versions.Prepare(storage)
			require.NoError(err)

			app := appdef.New()
			def := app.Add(cDocName, appdef.DefKind_CDoc)
			def.AddField("f1", appdef.DataKind_QName, true)
			def.SetSingleton()
			appDef, err := app.Build()
			require.NoError(err)

			return storage, versions, appDef
		}()

		err := stons.Prepare(storage, versions, appDef)
		require.NoError(err)

		test_AppDefSingletons(t, appDef, stons)
	})

	testID := func(id istructs.RecordID, known bool, qname appdef.QName) {
		t.Run(fmt.Sprintf("test Singletons.GetQName(%v)", id), func(t *testing.T) {
			qName, err := stons.GetQName(id)
			if known {
				require.NoError(err)
				require.Equal(qname, qName)
			} else {
				require.ErrorIs(err, ErrIDNotFound)
				require.Equal(qName, appdef.NullQName)
			}
		})
	}

	testQName := func(qname appdef.QName, known bool) {
		t.Run(fmt.Sprintf("test Stones.GetID(%v)", qname), func(t *testing.T) {
			var id istructs.RecordID
			var err error

			id, err = stons.GetID(qname)
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
		testQName(appdef.NullQName, false)
	})

	t.Run("check known QName", func(t *testing.T) {
		testQName(cDocName, true)
	})

	t.Run("check unknown QName", func(t *testing.T) {
		testQName(appdef.NewQName("unknown", "CDoc"), false)
	})

	t.Run("check unknown id", func(t *testing.T) {
		testID(istructs.MaxSingletonID-1, false, appdef.NullQName)
	})
}

func Test_Singletons_Errors(t *testing.T) {

	require := require.New(t)
	cDocName := appdef.NewQName("test", "SignletonCDoc")
	testError := fmt.Errorf("test error")

	t.Run("must error if unknown version of Singletons system view", func(t *testing.T) {
		storage, err := istorageimpl.Provide(istorage.ProvideMem()).AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)

		versions := vers.New()
		err = versions.Prepare(storage)
		require.NoError(err)

		err = versions.Put(vers.SysSingletonsVersion, 0xFF)
		require.NoError(err)

		stone := New()
		err = stone.Prepare(storage, versions, nil)
		require.ErrorIs(err, vers.ErrorInvalidVersion)
	})

	t.Run("must error if unable store version of Singletons system  view", func(t *testing.T) {
		storage := teststore.NewStorage()
		storage.SchedulePutError(testError, utils.ToBytes(consts.SysView_Versions), utils.ToBytes(vers.SysSingletonsVersion))

		versions := vers.New()
		err := versions.Prepare(storage)
		require.NoError(err)

		app := appdef.New()
		def := app.Add(cDocName, appdef.DefKind_CDoc)
		def.AddField("f1", appdef.DataKind_QName, true)
		def.SetSingleton()
		appDef, err := app.Build()
		require.NoError(err)

		stone := New()
		err = stone.Prepare(storage, versions, appDef)

		require.ErrorIs(err, testError)
	})

	t.Run("must error if maximum singletons is exceeded by CDocs", func(t *testing.T) {
		storage := teststore.NewStorage()

		versions := vers.New()
		err := versions.Prepare(storage)
		require.NoError(err)

		appDefBuilder := appdef.New()
		for id := istructs.FirstSingletonID; id <= istructs.MaxSingletonID; id++ {
			appDefBuilder.Add(appdef.NewQName("test", fmt.Sprintf("CDoc_%v", id)), appdef.DefKind_CDoc).SetSingleton()
		}
		appDef, err := appDefBuilder.Build()
		require.NoError(err)

		stons := New()
		err = stons.Prepare(storage, versions, appDef)

		require.ErrorIs(err, ErrSingletonIDsExceeds)
	})

	t.Run("must error if store ID for some singledoc to storage is failed", func(t *testing.T) {
		defName := appdef.NewQName("test", "ErrorDef")

		storage := teststore.NewStorage()
		storage.SchedulePutError(testError, utils.ToBytes(consts.SysView_SingletonIDs, lastestVersion), []byte(defName.String()))

		versions := vers.New()
		err := versions.Prepare(storage)
		require.NoError(err)

		app := appdef.New()
		app.Add(defName, appdef.DefKind_CDoc).SetSingleton()
		appDef, err := app.Build()
		require.NoError(err)

		stons := New()
		err = stons.Prepare(storage, versions, appDef)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if retrieve ID for some singledoc from storage is failed", func(t *testing.T) {
		defName := appdef.NewQName("test", "ErrorDef")

		storage := teststore.NewStorage()

		versions := vers.New()
		err := versions.Prepare(storage)
		require.NoError(err)

		app := appdef.New()
		app.Add(defName, appdef.DefKind_CDoc).SetSingleton()
		appDef, err := app.Build()
		require.NoError(err)

		stons := New()
		err = stons.Prepare(storage, versions, appDef)
		require.NoError(err)

		storage.ScheduleGetError(testError, nil, []byte(defName.String()))
		stons1 := New()
		err = stons1.Prepare(storage, versions, appDef)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if some some CDoc singleton QName from storage is not well formed", func(t *testing.T) {
		storage := teststore.NewStorage()

		versions := vers.New()
		err := versions.Prepare(storage)
		require.NoError(err)
		versions.Put(vers.SysSingletonsVersion, lastestVersion)

		t.Run("crack storage by put invalid QName string into Singletons system view", func(t *testing.T) {
			err = storage.Put(
				utils.ToBytes(consts.SysView_SingletonIDs, lastestVersion),
				[]byte("error.CDoc.be-e-e"),
				utils.ToBytes(istructs.MaxSingletonID),
			)
			require.NoError(err)
		})

		stons := New()
		err = stons.Prepare(storage, versions, nil)

		require.ErrorIs(err, appdef.ErrInvalidQNameStringRepresentation)
	})
}
