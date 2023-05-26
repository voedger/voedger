/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package uniques

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

func TestUniques(t *testing.T) {

	testName := appdef.NewQName("test", "doc")

	testAppDef := func(ver uint) appdef.IAppDef {
		app := appdef.New()

		def := app.AddCDoc(testName)
		def.
			AddField("name", appdef.DataKind_string, true).
			AddField("surname", appdef.DataKind_string, false).
			AddField("lastName", appdef.DataKind_string, false).
			AddField("passportNumber", appdef.DataKind_string, false).
			AddField("passportSerial", appdef.DataKind_string, false)

		if ver > 1 {
			def.AddUnique("absurdUnique", []string{"lastName", "passportSerial"})
		}

		def.
			AddUnique("fullNameUnique", []string{"name", "surname", "lastName"}).
			AddUnique("passportUnique", []string{"passportSerial", "passportNumber"})

		appDef, err := app.Build()
		if err != nil {
			panic(err)
		}

		return appDef
	}

	sp := istorageimpl.Provide(istorage.ProvideMem())
	storage, _ := sp.AppStorage(istructs.AppQName_test1_app1)

	versions := vers.New()
	if err := versions.Prepare(storage); err != nil {
		panic(err)
	}

	appDef1 := testAppDef(1)

	qNames1 := qnames.New()
	if err := qNames1.Prepare(storage, versions, appDef1, nil); err != nil {
		panic(err)
	}

	if err := PrepareAppDefUniqueIDs(storage, versions, qNames1, appDef1); err != nil {
		panic(err)
	}

	require := require.New(t)

	t.Run("basic Uniques methods", func(t *testing.T) {
		def := appDef1.CDoc(testName)

		require.Equal(2, def.UniqueCount())
		require.Equal(def.UniqueCount(), func() int {
			cnt := 0
			def.Uniques(func(u appdef.IUnique) {
				cnt++
				require.Greater(u.ID(), appdef.FirstUniqueID)
			})
			return cnt
		}())
	})

	t.Run("must be able to load early stored uniques", func(t *testing.T) {
		versions2 := vers.New()
		if err := versions2.Prepare(storage); err != nil {
			panic(err)
		}

		appDef2 := testAppDef(2)

		qNames2 := qnames.New()
		if err := qNames2.Prepare(storage, versions, appDef2, nil); err != nil {
			panic(err)
		}

		if err := PrepareAppDefUniqueIDs(storage, versions2, qNames2, appDef2); err != nil {
			panic(err)
		}

		def1 := appDef1.CDoc(testName)
		require.Equal(2, def1.UniqueCount())

		def2 := appDef2.CDoc(testName)
		require.Equal(3, def2.UniqueCount())

		def1.Uniques(func(u1 appdef.IUnique) {
			u2 := def2.UniqueByName(u1.Name())
			require.Equal(u1.ID(), u2.ID())

			u2 = def2.UniqueByID(u1.ID())
			require.Equal(u1.Name(), u2.Name())
		})
	})
}

func TestUniquesErrors(t *testing.T) {
	testName := appdef.NewQName("test", "doc")

	testAppDef := func() appdef.IAppDef {
		app := appdef.New()
		def := app.AddCDoc(testName)
		def.
			AddField("name", appdef.DataKind_string, true).
			AddField("surname", appdef.DataKind_string, false).
			AddField("lastName", appdef.DataKind_string, false).
			AddField("passportNumber", appdef.DataKind_string, false).
			AddField("passportSerial", appdef.DataKind_string, false)
		def.
			AddUnique("fullNameUnique", []string{"name", "surname", "lastName"}).
			AddUnique("passportUnique", []string{"passportSerial", "passportNumber"})

		appDef, err := app.Build()
		if err != nil {
			panic(err)
		}

		return appDef
	}

	require := require.New(t)

	t.Run("must error if unknown uniques system view version", func(t *testing.T) {
		storage := teststore.NewStorage()

		versions := vers.New()
		if err := versions.Prepare(storage); err != nil {
			panic(err)
		}

		versions.Put(vers.SysUniquesVersion, latestVersion+1) // future version

		appDef := testAppDef()

		qNames := qnames.New()
		if err := qNames.Prepare(storage, versions, appDef, nil); err != nil {
			panic(err)
		}

		err := PrepareAppDefUniqueIDs(storage, versions, qNames, appDef)
		require.ErrorIs(err, vers.ErrorInvalidVersion)
	})

	t.Run("must error if unknown definition uniques passed to Prepare()", func(t *testing.T) {
		storage := teststore.NewStorage()

		versions := vers.New()
		if err := versions.Prepare(storage); err != nil {
			panic(err)
		}

		appDef := testAppDef()

		qNames := qnames.New()
		if err := qNames.Prepare(storage, versions, appDef, nil); err != nil {
			panic(err)
		}

		t.Run("inject unknown definition to AppDef", func(t *testing.T) {
			def := appDef.(appdef.IAppDefBuilder).AddCDoc(appdef.NewQName("test", "unknown"))
			def.
				AddField("fld", appdef.DataKind_string, false)
			def.
				AddUnique("", []string{"fld"})
		})

		err := PrepareAppDefUniqueIDs(storage, versions, qNames, appDef)
		require.ErrorIs(err, qnames.ErrNameNotFound)
	})

	t.Run("must error if storage failed to write", func(t *testing.T) {
		storage := teststore.NewStorage()

		versions := vers.New()
		if err := versions.Prepare(storage); err != nil {
			panic(err)
		}

		appDef := testAppDef()

		qNames := qnames.New()
		if err := qNames.Prepare(storage, versions, appDef, nil); err != nil {
			panic(err)
		}

		t.Run("fail to write some unique id", func(t *testing.T) {
			writeError := errors.New("can not store unique")
			storage.SchedulePutError(writeError, utils.ToBytes(consts.SysView_UniquesIDs, latestVersion), nil)

			err := PrepareAppDefUniqueIDs(storage, versions, qNames, appDef)
			require.ErrorIs(err, writeError)
		})

		t.Run("fail to write uniques system view version", func(t *testing.T) {
			writeError := errors.New("can not store version")
			storage.SchedulePutError(writeError, utils.ToBytes(consts.SysView_Versions), utils.ToBytes(vers.SysUniquesVersion))

			err := PrepareAppDefUniqueIDs(storage, versions, qNames, appDef)
			require.ErrorIs(err, writeError)
		})
	})
}
