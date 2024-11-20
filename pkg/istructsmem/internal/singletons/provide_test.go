/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package singletons

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

func Test_BasicUsage(t *testing.T) {
	sp := istorageimpl.Provide(mem.Provide())
	storage, err := sp.AppStorage(istructs.AppQName_test1_app1)
	require.NoError(t, err)

	versions := vers.New()
	if err := versions.Prepare(storage); err != nil {
		panic(err)
	}

	testName := appdef.NewQName("test", "doc")

	testAppDef := func() appdef.IAppDef {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")
		ws := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
		doc := ws.AddCDoc(testName)
		doc.SetSingleton()
		appDef, err := adb.Build()
		if err != nil {
			panic(err)
		}
		return appDef
	}

	appDef := testAppDef()

	stones := New()
	if err := stones.Prepare(storage, versions, appDef); err != nil {
		panic(err)
	}

	require := require.New(t)
	t.Run("basic Singletons methods", func(t *testing.T) {
		id, err := stones.ID(testName)
		require.NoError(err)
		require.NotEqual(istructs.NullRecordID, id)

		t.Run("must be able to load early stored singletons", func(t *testing.T) {
			versions1 := vers.New()
			if err := versions1.Prepare(storage); err != nil {
				panic(err)
			}

			appDef1 := testAppDef()

			stones1 := New()
			if err := stones1.Prepare(storage, versions1, appDef1); err != nil {
				panic(err)
			}

			id1, err := stones1.ID(testName)
			require.NoError(err)
			require.Equal(id, id1)
		})
	})
}

func test_AppDefSingletons(t *testing.T, appDef appdef.IAppDef, st *Singletons) {
	require := require.New(t)
	for s := range appdef.Singletons(appDef.Types) {
		if s.Singleton() {
			id, err := st.ID(s.QName())
			require.NoError(err)
			require.NotEqual(istructs.NullRecordID, id)
		}
	}
}

func Test_SingletonsGetID(t *testing.T) {

	require := require.New(t)
	cDocName := appdef.NewQName("test", "SingletonCDoc")
	wDocName := appdef.NewQName("test", "SingletonWDoc")

	st := New()

	t.Run("must be ok to construct Singletons", func(t *testing.T) {
		storage, versions, appDef := func() (istorage.IAppStorage, *vers.Versions, appdef.IAppDef) {
			storage, err := istorageimpl.Provide(mem.Provide()).AppStorage(istructs.AppQName_test1_app1)
			require.NoError(err)

			versions := vers.New()
			err = versions.Prepare(storage)
			require.NoError(err)

			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

			{
				doc := wsb.AddCDoc(cDocName)
				doc.SetSingleton()
				doc.AddField("f1", appdef.DataKind_QName, true)
			}

			{
				doc := wsb.AddWDoc(wDocName)
				doc.SetSingleton()
				doc.AddField("f1", appdef.DataKind_QName, true)
			}

			appDef, err := adb.Build()
			require.NoError(err)

			return storage, versions, appDef
		}()

		err := st.Prepare(storage, versions, appDef)
		require.NoError(err)

		test_AppDefSingletons(t, appDef, st)
	})

	testID := func(id istructs.RecordID, known bool, qname appdef.QName) {
		t.Run(fmt.Sprintf("test Singletons QName(%v) founded", id), func(t *testing.T) {
			n, ok := st.ids[id]
			if known {
				require.True(ok)
				require.Equal(qname, n)
			} else {
				require.False(ok)
			}
		})
	}

	testQName := func(qname appdef.QName, known bool) {
		t.Run(fmt.Sprintf("test Stones.GetID(%v)", qname), func(t *testing.T) {
			var id istructs.RecordID
			var err error

			id, err = st.ID(qname)
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
		testQName(wDocName, true)
	})

	t.Run("check unknown QName", func(t *testing.T) {
		testQName(appdef.NewQName("unknown", "CDoc"), false)
		testQName(appdef.NewQName("unknown", "WDoc"), false)
	})

	t.Run("check unknown id", func(t *testing.T) {
		testID(istructs.MaxSingletonID-1, false, appdef.NullQName)
	})
}

func Test_Singletons_Errors(t *testing.T) {

	require := require.New(t)
	cDocName := appdef.NewQName("test", "SingletonCDoc")
	testError := errors.New("test error")

	t.Run("must error if unknown version of Singletons system view", func(t *testing.T) {
		storage, err := istorageimpl.Provide(mem.Provide()).AppStorage(istructs.AppQName_test1_app1)
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
		storage := teststore.NewStorage(istructs.AppQName_test1_app1)
		storage.SchedulePutError(testError, utils.ToBytes(consts.SysView_Versions), utils.ToBytes(vers.SysSingletonsVersion))

		versions := vers.New()
		err := versions.Prepare(storage)
		require.NoError(err)

		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		doc := ws.AddCDoc(cDocName)
		doc.SetSingleton()
		doc.AddField("f1", appdef.DataKind_QName, true)
		appDef, err := adb.Build()
		require.NoError(err)

		stone := New()
		err = stone.Prepare(storage, versions, appDef)

		require.ErrorIs(err, testError)
	})

	t.Run("must error if maximum singletons is exceeded", func(t *testing.T) {
		storage := teststore.NewStorage(istructs.AppQName_test1_app1)

		versions := vers.New()
		err := versions.Prepare(storage)
		require.NoError(err)

		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")
		ws := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		for id := istructs.FirstSingletonID; id <= istructs.MaxSingletonID; id++ {
			doc := ws.AddCDoc(appdef.NewQName("test", fmt.Sprintf("doc_%v", id)))
			doc.SetSingleton()
		}
		appDef, err := adb.Build()
		require.NoError(err)

		st := New()
		err = st.Prepare(storage, versions, appDef)

		require.ErrorIs(err, ErrSingletonIDsExceeds)
	})

	t.Run("must error if store ID for some singleton doc to storage is failed", func(t *testing.T) {
		defName := appdef.NewQName("test", "ErrorDef")

		storage := teststore.NewStorage(istructs.AppQName_test1_app1)
		storage.SchedulePutError(testError, utils.ToBytes(consts.SysView_SingletonIDs, latestVersion), []byte(defName.String()))

		versions := vers.New()
		err := versions.Prepare(storage)
		require.NoError(err)

		adb := appdef.New()
		ws := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
		doc := ws.AddCDoc(defName)
		doc.SetSingleton()
		appDef, err := adb.Build()
		require.NoError(err)

		st := New()
		err = st.Prepare(storage, versions, appDef)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if retrieve ID for some singleton doc from storage is failed", func(t *testing.T) {
		defName := appdef.NewQName("test", "ErrorDef")

		storage := teststore.NewStorage(istructs.AppQName_test1_app1)

		versions := vers.New()
		err := versions.Prepare(storage)
		require.NoError(err)

		adb := appdef.New()
		ws := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
		doc := ws.AddCDoc(defName)
		doc.SetSingleton()
		appDef, err := adb.Build()
		require.NoError(err)

		st := New()
		err = st.Prepare(storage, versions, appDef)
		require.NoError(err)

		storage.ScheduleGetError(testError, nil, []byte(defName.String()))
		st1 := New()
		err = st1.Prepare(storage, versions, appDef)
		require.ErrorIs(err, testError)
	})

	t.Run("must error if some some singleton QName from storage is not well formed", func(t *testing.T) {
		storage := teststore.NewStorage(istructs.AppQName_test1_app1)

		versions := vers.New()
		err := versions.Prepare(storage)
		require.NoError(err)
		versions.Put(vers.SysSingletonsVersion, latestVersion)

		t.Run("crack storage by put invalid QName string into Singletons system view", func(t *testing.T) {
			err = storage.Put(
				utils.ToBytes(consts.SysView_SingletonIDs, latestVersion),
				[]byte("error.CDoc.be-e-e"),
				utils.ToBytes(istructs.MaxSingletonID),
			)
			require.NoError(err)
		})

		st := New()
		err = st.Prepare(storage, versions, nil)

		require.ErrorIs(err, appdef.ErrConvertError)
	})
}
